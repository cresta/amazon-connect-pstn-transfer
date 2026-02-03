# CloudFormation Deployment

This directory contains CloudFormation templates for deploying the AWS Lambda function for Amazon Connect PSTN Transfer.

## Authentication

The Lambda function supports OAuth Client Credentials. You can provide credentials in one of two ways:

**Option 1: AWS Secrets Manager** ✅ **RECOMMENDED for production**

- Provide `OAuthSecretArn` parameter with the ARN of your Secrets Manager secret
- The secret must be a JSON object with `oauthClientId` and `oauthClientSecret` fields:
  ```json
  {
    "oauthClientId": "your-client-id",
    "oauthClientSecret": "your-client-secret"
  }
  ```
- The CloudFormation template will automatically add the necessary IAM permissions to access the secret
- If provided, takes precedence over `OAuthClientId`/`OAuthClientSecret` parameters

**Option 2: Environment Variables**

- Provide `OAuthClientId` and `OAuthClientSecret` parameters
- These will be set as Lambda environment variables

## Usage

### Using AWS CLI

#### Using parameters.json (Recommended)

1. Copy `parameters.json.example` to `parameters.json` and fill in your values:

   ```bash
   cp parameters.json.example parameters.json
   # Edit parameters.json with your values
   ```

2. Upload the zip from latest [release](https://github.com/cresta/amazon-connect-pstn-transfer/releases) to an S3 bucket

3. Update `parameters.json` with your S3 bucket name and key

4. Deploy using the parameters file:
   ```bash
   aws cloudformation create-stack \
     --stack-name my-stack \
     --template-body file://template.yaml \
     --parameters file://parameters.json \
     --capabilities CAPABILITY_NAMED_IAM
   ```

#### Using Inline Parameters

Alternatively, you can specify parameters inline:

```bash
# Define your configuration variables
STACK_NAME="my-stack"
LAMBDA_IMPLEMENTATION="go"  # Options: go, typescript, python
VIRTUAL_AGENT_NAME="customers/..."
REGION="us-west-2-prod"
CODE_S3_BUCKET="my-bucket"
# Map implementation to artifact suffix (go->go, typescript->ts, python->py)
case "$LAMBDA_IMPLEMENTATION" in
    typescript) ARTIFACT_SUFFIX="ts" ;;
    python) ARTIFACT_SUFFIX="py" ;;
    *) ARTIFACT_SUFFIX="$LAMBDA_IMPLEMENTATION" ;;  # go->go
esac
CODE_S3_KEY="aws-lambda-connect-pstn-transfer-${ARTIFACT_SUFFIX}.zip"
FUNCTION_NAME="aws-lambda-connect-pstn-transfer"
ROLE_NAME="aws-lambda-connect-pstn-transfer-role"

# Authentication (choose one option)
# Option 1: AWS Secrets Manager (Recommended for production)
OAUTH_SECRET_ARN="arn:aws:secretsmanager:us-west-2:123456789012:secret:my-oauth-secret"

# Option 2: Environment variables
OAUTH_CLIENT_ID="your-client-id"
OAUTH_CLIENT_SECRET="your-client-secret"

# Deploy using Secrets Manager (Recommended)
aws cloudformation create-stack \
  --stack-name "${STACK_NAME}" \
  --template-body file://template.yaml \
  --parameters \
    ParameterKey=LambdaImplementation,ParameterValue="${LAMBDA_IMPLEMENTATION}" \
    ParameterKey=OAuthSecretArn,ParameterValue="${OAUTH_SECRET_ARN}" \
    ParameterKey=VirtualAgentName,ParameterValue="${VIRTUAL_AGENT_NAME}" \
    ParameterKey=Region,ParameterValue="${REGION}" \
    ParameterKey=CodeS3Bucket,ParameterValue="${CODE_S3_BUCKET}" \
    ParameterKey=CodeS3Key,ParameterValue="${CODE_S3_KEY}" \
    ParameterKey=FunctionName,ParameterValue="${FUNCTION_NAME}" \
    ParameterKey=RoleName,ParameterValue="${ROLE_NAME}" \
  --capabilities CAPABILITY_NAMED_IAM

# Or deploy using OAuth credentials as environment variables
aws cloudformation create-stack \
  --stack-name "${STACK_NAME}" \
  --template-body file://template.yaml \
  --parameters \
    ParameterKey=LambdaImplementation,ParameterValue="${LAMBDA_IMPLEMENTATION}" \
    ParameterKey=OAuthClientId,ParameterValue="${OAUTH_CLIENT_ID}" \
    ParameterKey=OAuthClientSecret,ParameterValue="${OAUTH_CLIENT_SECRET}" \
    ParameterKey=VirtualAgentName,ParameterValue="${VIRTUAL_AGENT_NAME}" \
    ParameterKey=Region,ParameterValue="${REGION}" \
    ParameterKey=CodeS3Bucket,ParameterValue="${CODE_S3_BUCKET}" \
    ParameterKey=CodeS3Key,ParameterValue="${CODE_S3_KEY}" \
    ParameterKey=FunctionName,ParameterValue="${FUNCTION_NAME}" \
    ParameterKey=RoleName,ParameterValue="${ROLE_NAME}" \
  --capabilities CAPABILITY_NAMED_IAM
```

### Using the Deployment Script

The `deploy-cloudformation.sh` script automates the build, upload, and deployment process:

```bash
./infra/cloudformation/deploy-cloudformation.sh
```

The script will:

1. Prompt for all required values:
   - Lambda implementation type: Go, TypeScript, or Python (default: Go)
   - CloudFormation stack name (required)
   - Authentication method:
     - Option 1: AWS Secrets Manager (recommended for production)
       - OAuth Secret ARN (required)
     - Option 2: OAuth 2 credentials as environment variables
       - OAuth Client ID and OAuth Client Secret (required)
   - Virtual Agent Name (required)
   - Region (optional, defaults to `us-west-2-prod`)
   - S3 bucket name (required)
   - S3 key (optional, defaults based on implementation type:
     - Go: `aws-lambda-connect-pstn-transfer-go.zip`
     - TypeScript: `aws-lambda-connect-pstn-transfer-ts.zip`
     - Python: `aws-lambda-connect-pstn-transfer-py.zip`)
   - Lambda function name (optional, defaults to `aws-lambda-connect-pstn-transfer`)
   - IAM role name (optional, defaults to `aws-lambda-connect-pstn-transfer-role`)
2. Build the Lambda function for Linux ARM64 (using the appropriate build script)
3. Create a deployment package (zip)
4. Upload the package to S3
5. Create or update the CloudFormation stack using inline parameters

**Note:** The template uses CloudFormation Conditions to automatically set the correct Runtime, Handler, and default S3 key based on the selected implementation type. Shared parameters (OAuth credentials, Virtual Agent Name, Region) are used for both implementations.

**Note:** The script prompts for all values interactively. For automated deployments or CI/CD pipelines, use `parameters.json` with AWS CLI as shown above.
