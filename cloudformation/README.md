# CloudFormation Deployment

This directory contains CloudFormation templates for deploying the AWS Lambda function for Amazon Connect PSTN Transfer.

## Authentication

The Lambda function supports OAuth Client Credentials.

- Provide `OAuthClientId` and `OAuthClientSecret` parameters

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
# Using OAuth 2 authentication (Recommended)
aws cloudformation create-stack \
  --stack-name my-stack \
  --template-body file://template.yaml \
  --parameters \
    ParameterKey=OAuthClientId,ParameterValue=your-client-id \
    ParameterKey=OAuthClientSecret,ParameterValue=your-client-secret \
    ParameterKey=VirtualAgentName,ParameterValue=customers/... \
    ParameterKey=Region,ParameterValue=us-west-2-prod \
    ParameterKey=CodeS3Bucket,ParameterValue=my-bucket \
    ParameterKey=CodeS3Key,ParameterValue=function.zip \
    ParameterKey=FunctionName,ParameterValue=aws-lambda-connect-pstn-transfer \
    ParameterKey=RoleName,ParameterValue=aws-lambda-connect-pstn-transfer-role \
  --capabilities CAPABILITY_NAMED_IAM
```

### Using the Deployment Script

The `deploy-cloudformation.sh` script automates the build, upload, and deployment process:

```bash
./cloudformation/deploy-cloudformation.sh
```

The script will:
1. Prompt for all required values:
   - CloudFormation stack name (required)
   - Authentication method: OAuth 2
     - OAuth Client ID and OAuth Client Secret (required)
   - Virtual Agent Name (required)
   - Region (optional, defaults to `us-west-2-prod`)
   - S3 bucket name (required)
   - S3 key (optional, defaults to `aws-lambda-connect-pstn-transfer.zip`)
   - Lambda function name (optional, defaults to `aws-lambda-connect-pstn-transfer`)
   - IAM role name (optional, defaults to `aws-lambda-connect-pstn-transfer-role`)
2. Build the Lambda function for Linux ARM64
3. Create a deployment package (zip)
4. Upload the package to S3
5. Create or update the CloudFormation stack using inline parameters

**Note:** The script prompts for all values interactively. For automated deployments or CI/CD pipelines, use `parameters.json` with AWS CLI as shown above.

