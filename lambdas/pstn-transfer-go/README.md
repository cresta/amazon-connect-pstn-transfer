# PSTN Transfer Go Lambda

Go implementation of the Amazon Connect PSTN Transfer Lambda function.

## Overview

This Go implementation provides the same functionality as the TypeScript implementation:
- `get_pstn_transfer_data`: Generates PSTN transfer data (phone number and DTMF sequence)
- `get_handoff_data`: Fetches AI agent handoff data

## Development

### Prerequisites

- Go 1.x
- AWS CLI configured with appropriate credentials
- Visual Studio Code (recommended)
- ZIP utility

### VS Code Configuration

The project includes VS Code configurations for optimal development:

1. **Required Extensions**:
   - **dfarley1.file-picker**: Required for the `event` task to select test event files. VS Code should prompt you to install this when opening the workspace. Otherwise you can install vsix in the `.vscode` folder.

2. **Recommended Extensions**:
   - Install the recommended Go extensions for VS Code

3. **Debugging**:
   The project includes launch configurations for debugging your Lambda function locally.

4. **Tasks**:
   Predefined tasks are available for building and testing the application. The `event` task requires the `dfarley1.file-picker` extension to select test event files.

### Building

Build the Lambda function for Linux ARM64:

```bash
./scripts/build-go-lambda.sh
```

This creates `aws-lambda-connect-pstn-transfer.zip` in the project root.

### Testing

```bash
go test ./...
```

### Local Development

1. Export authentication credentials:
   - **Recommended**: For OAuth 2: `export oauthClientId=<client-id>` and `export oauthClientSecret=<client-secret>`
   - **Deprecated**: For API key: `export apiKey=<apiKey>`
2. Export other required variables: `export region=<region>` (e.g., `us-west-2-prod`). `apiDomain` is deprecated and will be constructed from `region` if not provided.
3. Export optional variables: `export supportedDtmfChars=<dtmf-chars>` (defaults to `0123456789*` if not provided).
4. Run the `build and debug` function through VS Code's debugger after making changes
5. Use the provided test event in `shared/testdata/` via `cmd + shift P -> Run Task -> event`
6. Check the debug console for output and response

## Deployment

### Environment Configuration

Create a `var.json` file in the project root with your environment variables:

**Recommended (OAuth 2 authentication):**
```json
{
    "virtualAgentName": "your-virtual-agent-resource-name",
    "region": "us-west-2-prod",
    "oauthClientId": "your-client-id",
    "oauthClientSecret": "your-client-secret"
}
```

**Deprecated (API Key authentication):**
```json
{
    "virtualAgentName": "your-virtual-agent-resource-name",
    "region": "us-west-2-prod",
    "apiDomain": "https://api.us-west-2-prod.cresta.com",
    "apiKey": "your-api-key"
}
```

Note: For prod regions (ending in `-prod`), the API domain uses `.cresta.com`; for staging regions, it uses `.cresta.ai`. This matches the behavior of `BuildAPIDomainFromRegion`.

### Manual Deployment

The project includes a `deploy.sh` script in the `scripts` folder that handles the entire deployment process:

It creates a
- **IAM Role**: `aws-lambda-connect-pstn-transfer-role`
  - Includes basic Lambda execution permissions
- **Lambda Function**: `aws-lambda-connect-pstn-transfer`
  - Runtime: Amazon Linux 2023 (Custom Runtime)
  - Architecture: ARM64
  - Handler: bootstrap

```bash
# Make the script executable and run the deployment
cd scripts
chmod +x deploy.sh

PROFILE=<some-aws-profile>
eval "$(aws configure export-credentials --profile $PROFILE --format env)"
./deploy.sh
```

The deployment script will:
1. Build the Lambda function for Linux ARM64
2. Create a deployment package (zip)
3. Create or update the IAM role with necessary permissions
4. Create or update the Lambda function
5. Configure environment variables from `var.json`

### CloudFormation Deployment

The project includes a CloudFormation template for infrastructure-as-code deployment. See the [CloudFormation README](../../infra/cloudformation/README.md) for detailed instructions.

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

## Testing

The test suite validates:
- Successful requests for both actions
- Error handling
- Parameter filtering
- Authentication (API key and OAuth 2)
- Response transformation

Tests use the standard Go testing package and can be run with `go test ./...`.
