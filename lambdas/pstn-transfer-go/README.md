# PSTN Transfer Go Lambda

Go implementation of the Amazon Connect PSTN Transfer Lambda function.

## Overview

This Go implementation provides the same functionality as the TypeScript implementation:
- `get_pstn_transfer_data`: Generates PSTN transfer data (phone number and DTMF sequence)
- `get_handoff_data`: Fetches AI agent handoff data

## Development

### Prerequisites

- Go 1.22+

### Building

Build the Lambda function for Linux ARM64:

```bash
./scripts/build-go-lambda.sh
```

This creates `aws-lambda-connect-pstn-transfer-go.zip` in the project root.

### Testing

```bash
go test ./...
```

### Local Development

1. Use VS Code's debugger:
   - Select "Launch (Go)" from the debug configuration dropdown
   - When prompted, enter:
     - `oauthClientId`: OAuth 2 client ID
     - `oauthClientSecret`: OAuth 2 client secret
     - `region`: AWS region (e.g., `us-west-2-prod`)
   - The debugger will start the Lambda function as a local server on port 8080
   - Use the `event-go` task (`cmd + shift P -> Run Task -> event-go`) to send test events from `shared/testdata/events/` to the running Lambda
2. Check the debug console for output and response

## Deployment

### Script Deployment

The project includes a `deploy.sh` script in the `scripts` folder that handles the entire deployment process:

It creates a
- **IAM Role**: `aws-lambda-connect-pstn-transfer-role`
  - Includes basic Lambda execution permissions
- **Lambda Function**: `aws-lambda-connect-pstn-transfer`
  - Runtime: Amazon Linux 2023 (Custom Runtime)
  - Architecture: ARM64
  - Handler: bootstrap

> **Note**: The `deploy.sh` script will automatically create a `var.json` file in the project root with your environment variables (OAuth credentials, virtual agent name, and region) when you run it for the first time.

```bash
# Make the script executable and run the deployment
cd scripts
chmod +x deploy.sh

PROFILE=<some-aws-profile>
eval "$(aws configure export-credentials --profile $PROFILE --format env)"
./deploy.sh
```

The deployment script will:
1. Prompt for environment variables and create `var.json` (if it doesn't exist)
2. Build the Lambda function for Linux ARM64
3. Create a deployment package (zip)
4. Create or update the IAM role with necessary permissions
5. Create or update the Lambda function
6. Configure environment variables from `var.json`

### CloudFormation Deployment

The project includes a CloudFormation template for infrastructure-as-code deployment. See the [CloudFormation README](../../infra/cloudformation/README.md) for detailed instructions.

### Manual Deployment via AWS Portal

You can also deploy the Lambda function manually through the AWS Console. Follow these steps:

#### Prerequisites

1. Build the Lambda function (or download from [GitHub Releases](https://github.com/cresta/amazon-connect-pstn-transfer/releases)):
   ```bash
   ./scripts/build-go-lambda.sh
   ```
   This creates `aws-lambda-connect-pstn-transfer-go.zip` in the project root.

2. (optional) Upload the deployment package to an S3 bucket
   ```bash
   aws s3 cp aws-lambda-connect-pstn-transfer-go.zip s3://your-bucket-name/
   ```

#### Step-by-Step Deployment

1. **Navigate to AWS Lambda Console**
   - Go to AWS Console → Lambda → Functions
   - Click "Create function"

2. **Configure Function**
   - **Function name**: `aws-lambda-connect-pstn-transfer`
   - **Runtime**: Select "Custom runtime on Amazon Linux 2023" (`provided.al2023`)
   - **Architecture**: Select `ARM64`
   - Click "Create function"

3. **Upload Code**
   - In the "Code" tab, choose one of:
     - **Option A**: Upload a .zip file directly (for smaller packages)
       - Click "Upload from" → ".zip file"
       - Select `aws-lambda-connect-pstn-transfer-go.zip`
     - **Option B**: Upload from S3 (recommended for larger packages)
       - Click "Upload from" → "Amazon S3 location"
       - Enter the S3 URL: `s3://your-bucket-name/aws-lambda-connect-pstn-transfer-go.zip`
   - Click "Save"

4. **Configure Handler**
   - In the "Code" tab, scroll to "Runtime settings"
   - **Handler**: `bootstrap` (this is the executable name for Go Lambda custom runtime)
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
     - `oauthClientSecret`: Your OAuth 2 Client Secret (mark as "Encrypt" for security)
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

- **Runtime**: `provided.al2023` (Amazon Linux 2023 Custom Runtime)
  - This runtime is used for Go Lambda functions compiled as a single binary
- **Handler**: `bootstrap`
  - For Go Lambda custom runtime, the handler is always `bootstrap`, which is the name of the compiled executable
- **Architecture**: `ARM64`
  - The Lambda is compiled for ARM64 architecture for better cost efficiency
- **Timeout**: `30 seconds`
  - Maximum execution time for the function. Actual execution should take a lot less.
- **Memory**: `256 MB`
  - Memory allocation affects CPU power proportionally.

## Structure

- `main.go` - Lambda handler entry point
- `handlers.go` - Business logic handlers for each action
- `client.go` - API client for making HTTP requests
- `httpclient.go` - HTTP client with retry logic
- `auth.go` - OAuth 2 authentication with token caching
- `logger.go` - Logging utility
- `utils.go` - Utility functions (validation, parsing, etc.)
- `models.go` - Go type definitions
- `*_test.go` - Test files
