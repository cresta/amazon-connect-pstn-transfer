# PSTN Transfer Python Lambda

Python implementation of the Amazon Connect PSTN Transfer Lambda function, matching the Go and TypeScript implementations exactly.

## Overview

This Python implementation provides the same functionality as the Go and TypeScript implementations:

- `get_pstn_transfer_data`: Generates PSTN transfer data (phone number and DTMF sequence)
- `get_handoff_data`: Fetches AI agent handoff data

## Development

### Prerequisites

- Python 3.14+
- pip or pipenv

### Setup

```bash
cd lambdas/pstn-transfer-py
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements-dev.txt
```

### Build

The Python Lambda doesn't require compilation, but you can run type checking:

```bash
mypy src
```

### Test

```bash
pytest
```

With coverage:

```bash
pytest --cov=src --cov-report=html
```

### Lint

```bash
ruff check src tests
```

### Format

```bash
ruff format src tests
```

### Package for Lambda

```bash
# From the lambdas/pstn-transfer-py directory
./package.sh
```

This creates a zip file at the project root (`aws-lambda-connect-pstn-transfer-py.zip`) ready for Lambda deployment.

### Local Development

1. Use VS Code's debugger:
   - Select "Launch (Python)" from the debug configuration dropdown
   - When prompted, select a test event file from `shared/testdata/events/` (e.g., `test_get_handoff_data.json`)
   - When prompted, enter:
     - `oauthClientId`: OAuth 2 client ID
     - `oauthClientSecret`: OAuth 2 client secret
     - `region`: AWS region (e.g., `us-west-2-prod`)
   - The debugger will start and execute the Lambda function with your selected test event
2. Check the debug console for output and response

## Deployment

This Python Lambda can be deployed alongside or instead of the Go/TypeScript implementations. All implementations are functionally equivalent and validated using the shared integration tests in `shared/testdata/`.

### Script Deployment

The project includes a `deploy.sh` script in the `scripts` folder that handles the entire deployment process:

It creates:

- **IAM Role**: `aws-lambda-connect-pstn-transfer-role`
  - Includes basic Lambda execution permissions
- **Lambda Function**: `aws-lambda-connect-pstn-transfer`
  - Runtime: Python 3.14
  - Architecture: ARM64
  - Handler: src.handler.handler

> **Note**: The `deploy.sh` script will automatically create a `var.json` file in the project root with your environment variables (OAuth credentials, virtual agent name, and region) when you run it for the first time.

```bash
# Make the script executable and run the deployment
cd scripts
chmod +x deploy.sh

PROFILE=<some-aws-profile>
eval "$(aws configure export-credentials --profile $PROFILE --format env)"
./deploy.sh
```

When prompted, select option **3** for Python implementation.

The deployment script will:

1. Prompt for environment variables and create `var.json` (if it doesn't exist)
2. Build the Lambda function
3. Create a deployment package (zip)
4. Create or update the IAM role with necessary permissions
5. Create or update the Lambda function
6. Configure environment variables from `var.json`

### CloudFormation Deployment

The project includes a CloudFormation template for infrastructure-as-code deployment. See the [CloudFormation README](../../infra/cloudformation/README.md) for detailed instructions.

### Manual Deployment via AWS Portal

You can also deploy the Lambda function manually through the AWS Console. Follow these steps:

#### Prerequisites

1. Build and package the Lambda function (or download from [GitHub Releases](https://github.com/cresta/amazon-connect-pstn-transfer/releases)):

   ```bash
   cd lambdas/pstn-transfer-py
   ./package.sh
   ```

   This creates `aws-lambda-connect-pstn-transfer-py.zip` in the parent directory.

2. (optional) Upload the deployment package to an S3 bucket
   ```bash
   aws s3 cp aws-lambda-connect-pstn-transfer-py.zip s3://your-bucket-name/
   ```

#### Step-by-Step Deployment

1. **Navigate to AWS Lambda Console**
   - Go to AWS Console → Lambda → Functions
   - Click "Create function"

2. **Configure Function**
   - **Function name**: `aws-lambda-connect-pstn-transfer`
   - **Runtime**: Select "Python 3.14" (`python3.14`)
   - **Architecture**: Select `ARM64` (recommended) or `x86_64`
   - Click "Create function"

3. **Upload Code**
   - In the "Code" tab, choose one of:
     - **Option A**: Upload a .zip file directly (for smaller packages)
       - Click "Upload from" → ".zip file"
       - Select `aws-lambda-connect-pstn-transfer-py.zip`
     - **Option B**: Upload from S3 (recommended for larger packages)
       - Click "Upload from" → "Amazon S3 location"
       - Enter the S3 URL: `s3://your-bucket-name/aws-lambda-connect-pstn-transfer-py.zip`
   - Click "Save"

4. **Configure Handler**
   - In the "Code" tab, scroll to "Runtime settings"
   - **Handler**: `src.handler.handler` (points to the `handler` function in `src/handler.py`)
   - Click "Edit" to modify, then "Save"

5. **Configure Function Settings**
   - Go to the "Configuration" tab → "General configuration"
   - Click "Edit" and configure:
     - **Timeout**: `30 seconds`
     - **Memory**: `256 MB`
   - Click "Save"

6. **Set Environment Variables**
   - Go to "Configuration" → "Environment variables"
   - Click "Edit" → "Add environment variable" for each:
     - `oauthClientId`: Your OAuth 2 Client ID
     - `oauthClientSecret`: Your OAuth 2 Client Secret
     - `region`: Your AWS region (e.g., `us-west-2-prod`)
     - `virtualAgentName`: Resource name in format `customers/{customer}/profiles/{profile}/virtualAgents/{virtualAgentID}`
   - Click "Save"

   > **Note**: These values can also be passed as parameters from your Amazon Connect flow. Parameters passed from Amazon Connect take precedence over environment variables. It's recommended to set sensitive values (like credentials) as environment variables and pass `action` and other flow-specific values as parameters.

7. **Configure IAM Role**
   - Go to "Configuration" → "Permissions"
   - Click on the role name to open IAM Console
   - Ensure the role has the `AWSLambdaBasicExecutionRole` managed policy attached
   - This allows the Lambda to write logs to CloudWatch

8. **Test the Function**
   - Go to the "Test" tab
   - Create a test event or use an existing one from the `shared/testdata/events/` folder.
     - e.g. `test_get_handoff_data.json`
   - Run the test to verify the function works correctly

#### Architecture and Configuration Details

- **Runtime**: `python3.14` (Python 3.14)
  - This runtime provides Python 3.14 execution environment for Lambda functions
- **Handler**: `src.handler.handler`
  - Points to the `handler` function in the `src/handler.py` file
  - Format: `{module_path}.{function_name}`
- **Architecture**: `ARM64` (recommended) or `x86_64`
  - ARM64 architecture is recommended for better cost efficiency
- **Timeout**: `30 seconds`
  - Maximum execution time for the function. Actual execution should take a lot less.
- **Memory**: `256 MB`
  - Memory allocation affects CPU power proportionally.

## Structure

- `src/handler.py` - Main Lambda handler entry point
- `src/handlers.py` - Business logic handlers for each action
- `src/client.py` - API client for making HTTP requests
- `src/httpclient.py` - HTTP client with retry logic
- `src/auth.py` - OAuth 2 authentication with token caching
- `src/logger.py` - Logging utility
- `src/utils.py` - Utility functions (validation, parsing, etc.)
- `src/types.py` - Python type definitions (dataclasses)
- `src/secretsmanager.py` - AWS Secrets Manager integration
- `src/version.py` - Version information
- `tests/` - Unit tests

**Note**: The shared integration tests that validate all implementations are located in `shared/testdata/` and should be run separately. See the [Shared Tests README](../../shared/testdata/README.md) for details.
