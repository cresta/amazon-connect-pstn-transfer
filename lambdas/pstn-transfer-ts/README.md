# PSTN Transfer TypeScript Lambda

TypeScript implementation of the Amazon Connect PSTN Transfer Lambda function, matching the Go implementation exactly.

## Overview

This TypeScript implementation provides the same functionality as the Go implementation:
- `get_pstn_transfer_data`: Generates PSTN transfer data (phone number and DTMF sequence)
- `get_handoff_data`: Fetches AI agent handoff data

## Development

### Prerequisites

- Node.js 24+ 
- npm or yarn

### Setup

```bash
cd lambdas/pstn-transfer-ts
npm install
```

### Build

```bash
npm run build
```

This compiles TypeScript to JavaScript in the `dist/` directory.

### Test

```bash
npm test
```

### Package for Lambda

```bash
npm run package
```

This creates a zip file at `../aws-lambda-connect-pstn-transfer-ts.zip` ready for Lambda deployment.

### Local Development

1. Use VS Code's debugger:
   - Select "Launch (TypeScript)" from the debug configuration dropdown
   - When prompted, select a test event file from `shared/testdata/events/` (e.g., `test_get_handoff_data.json`)
   - When prompted, enter:
     - `oauthClientId`: OAuth 2 client ID
     - `oauthClientSecret`: OAuth 2 client secret
     - `region`: AWS region (e.g., `us-west-2-prod`)
   - The debugger will start and execute the Lambda function with your selected test event
2. Check the debug console for output and response

## Deployment

This TypeScript Lambda can be deployed alongside or instead of the Go implementation. Both implementations are functionally equivalent and validated using the shared integration tests in `shared/testdata/`.

### Script Deployment

The project includes a `deploy.sh` script in the `scripts` folder that handles the entire deployment process:

It creates a
- **IAM Role**: `aws-lambda-connect-pstn-transfer-role`
  - Includes basic Lambda execution permissions
- **Lambda Function**: `aws-lambda-connect-pstn-transfer`
  - Runtime: Node.js 24.x
  - Architecture: ARM64
  - Handler: handler.handler

> **Note**: The `deploy.sh` script will automatically create a `var.json` file in the project root with your environment variables (OAuth credentials, virtual agent name, and region) when you run it for the first time.

```bash
# Make the script executable and run the deployment
cd scripts
chmod +x deploy.sh

PROFILE=<some-aws-profile>
eval "$(aws configure export-credentials --profile $PROFILE --format env)"
./deploy.sh
```

When prompted, select option **2** for TypeScript implementation.

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
   cd lambdas/pstn-transfer-ts
   npm install
   npm run build
   npm run package
   ```
   This creates `aws-lambda-connect-pstn-transfer-ts.zip` in the parent directory.

2. (optional) Upload the deployment package to an S3 bucket
   ```bash
   aws s3 cp aws-lambda-connect-pstn-transfer-ts.zip s3://your-bucket-name/
   ```

#### Step-by-Step Deployment

1. **Navigate to AWS Lambda Console**
   - Go to AWS Console → Lambda → Functions
   - Click "Create function"

2. **Configure Function**
   - **Function name**: `aws-lambda-connect-pstn-transfer`
   - **Runtime**: Select "Node.js 24.x" (`nodejs24.x`)
   - **Architecture**: Select `ARM64` (recommended) or `x86_64`
   - Click "Create function"

3. **Upload Code**
   - In the "Code" tab, choose one of:
     - **Option A**: Upload a .zip file directly (for smaller packages)
       - Click "Upload from" → ".zip file"
       - Select `aws-lambda-connect-pstn-transfer-ts.zip`
     - **Option B**: Upload from S3 (recommended for larger packages)
       - Click "Upload from" → "Amazon S3 location"
       - Enter the S3 URL: `s3://your-bucket-name/aws-lambda-connect-pstn-transfer-ts.zip`
   - Click "Save"

4. **Configure Handler**
   - In the "Code" tab, scroll to "Runtime settings"
   - **Handler**: `handler.handler` (points to the exported `handler` function in `handler.js`)
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

- **Runtime**: `nodejs24.x` (Node.js 24.x)
  - This runtime provides Node.js 24.x execution environment for TypeScript/JavaScript Lambda functions
- **Handler**: `handler.handler`
  - Points to the exported `handler` function in the `handler.js` file (compiled from `src/handler.ts`)
  - Format: `{filename}.{exportedFunctionName}`
- **Architecture**: `ARM64` (recommended) or `x86_64`
  - ARM64 architecture is recommended for better cost efficiency
- **Timeout**: `30 seconds`
  - Maximum execution time for the function. Actual execution should take a lot less.
- **Memory**: `256 MB`
  - Memory allocation affects CPU power proportionally.

## Structure

- `src/handler.ts` - Main Lambda handler entry point
- `src/handlers.ts` - Business logic handlers for each action
- `src/client.ts` - API client for making HTTP requests
- `src/httpclient.ts` - HTTP client with retry logic
- `src/auth.ts` - OAuth 2 authentication with token caching
- `src/logger.ts` - Logging utility
- `src/utils.ts` - Utility functions (validation, parsing, etc.)
- `src/models.ts` - TypeScript type definitions

**Note**: The shared integration tests that validate both Go and TypeScript implementations are located in `shared/testdata/` and should be run separately. See the [Shared Tests README](../../shared/testdata/README.md) for details.
