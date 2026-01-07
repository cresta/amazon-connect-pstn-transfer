# CloudFormation Deployment

This directory contains CloudFormation templates for deploying the AWS Lambda function for Amazon Connect PSTN Transfer.

## Code Source

The Lambda function code is provided via **S3 bucket and key**:

```yaml
Code:
  S3Bucket: my-bucket-name
  S3Key: path/to/function.zip
```

**With versioning (optional):**
To use a versioned S3 object, you can add `S3ObjectVersion` to the `Code` property in the template:

```yaml
Code:
  S3Bucket: !Ref CodeS3Bucket
  S3Key: !Ref CodeS3Key
  S3ObjectVersion: "your-version-id-here"
```

## Usage

### Using AWS CLI

```bash
aws cloudformation create-stack \
  --stack-name my-stack \
  --template-body file://template.yaml \
  --parameters \
    ParameterKey=ApiKey,ParameterValue=your-key \
    ParameterKey=VirtualAgentName,ParameterValue=customers/... \
    ParameterKey=CodeS3Bucket,ParameterValue=my-bucket \
    ParameterKey=CodeS3Key,ParameterValue=function.zip
```

### Using the Deployment Script

The `deploy-cloudformation.sh` script automates the build, upload, and deployment process:

```bash
./cloudformation/deploy-cloudformation.sh
```

