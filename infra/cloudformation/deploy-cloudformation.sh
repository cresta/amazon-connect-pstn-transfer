#!/bin/bash

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"

# Change to project root
cd "$PROJECT_ROOT" || exit 1

echo "=== AWS Lambda CloudFormation Deployment ==="
echo ""

TEMPLATE_FILE="$SCRIPT_DIR/template.yaml"

# Prompt for Lambda implementation type
echo "Select Lambda implementation:"
echo "1) Go (default)"
echo "2) TypeScript"
read -p "Enter choice [1-2] (default: 1): " impl_choice

case "$impl_choice" in
    2)
        lambda_impl="typescript"
        CODE_ZIP="aws-lambda-connect-pstn-transfer-ts.zip"
        BUILD_SCRIPT="$PROJECT_ROOT/scripts/build-typescript-lambda.sh"
        DEFAULT_CODE_S3_KEY="aws-lambda-connect-pstn-transfer-ts.zip"
        ;;
    *)
        lambda_impl="go"
        CODE_ZIP="aws-lambda-connect-pstn-transfer.zip"
        BUILD_SCRIPT="$PROJECT_ROOT/scripts/build-go-lambda.sh"
        DEFAULT_CODE_S3_KEY="aws-lambda-connect-pstn-transfer.zip"
        ;;
esac

echo "Selected implementation: $lambda_impl"
echo ""

# Prompt for required values
read -p "Enter CloudFormation stack name (required): " stack_name
if [ -z "$stack_name" ]; then
    echo "Error: Stack name is required"
    exit 1
fi

read -p "Enter OAuth Client ID (required): " oauth_client_id
if [ -z "$oauth_client_id" ]; then
    echo "Error: OAuth Client ID is required"
    exit 1
fi

read -sp "Enter OAuth Client Secret (required): " oauth_client_secret
echo ""
if [ -z "$oauth_client_secret" ]; then
    echo "Error: OAuth Client Secret is required"
    exit 1
fi

read -p "Enter Virtual Agent Name (required, format: customers/{customer}/profiles/{profile}/virtualAgents/{virtualAgentID}): " virtual_agent_name
if [ -z "$virtual_agent_name" ]; then
    echo "Error: Virtual Agent Name is required"
    exit 1
fi

read -p "Enter Region (optional, default: us-west-2-prod): " region
if [ -z "$region" ]; then
    region="us-west-2-prod"
fi

read -p "Enter S3 bucket name for Lambda code (required): " s3_bucket
if [ -z "$s3_bucket" ]; then
    echo "Error: S3 bucket name is required"
    exit 1
fi

read -p "Enter S3 key for Lambda code (optional, default: $DEFAULT_CODE_S3_KEY): " code_s3_key
if [ -z "$code_s3_key" ]; then
    code_s3_key="$DEFAULT_CODE_S3_KEY"
fi

read -p "Enter Lambda function name (optional, default: aws-lambda-connect-pstn-transfer): " function_name
if [ -z "$function_name" ]; then
    function_name="aws-lambda-connect-pstn-transfer"
fi

read -p "Enter IAM role name (optional, default: aws-lambda-connect-pstn-transfer-role): " role_name
if [ -z "$role_name" ]; then
    role_name="aws-lambda-connect-pstn-transfer-role"
fi

echo ""

# Build the zip using the appropriate build script
echo "Building Lambda function ($lambda_impl)..."
# Remove any existing ZIP file to avoid stale artifacts
if [ -f "$CODE_ZIP" ]; then
    rm -f "$CODE_ZIP"
fi
"$BUILD_SCRIPT"
if [ $? -ne 0 ]; then
    echo "Error: Build script failed"
    exit 1
fi
if [ ! -f "$CODE_ZIP" ]; then
    echo "Error: Failed to create deployment package: $CODE_ZIP"
    exit 1
fi

# Upload to S3
echo "Uploading deployment package to S3..."
aws s3 cp "$CODE_ZIP" "s3://$s3_bucket/$code_s3_key" --no-cli-pager

if [ $? -ne 0 ]; then
    echo "Error: Failed to upload to S3"
    exit 1
fi

echo "Deployment package uploaded successfully"
echo ""

# Check if stack exists
stack_exists=$(aws cloudformation describe-stacks --stack-name "$stack_name" --query "Stacks[0].StackName" --output text 2>/dev/null)

if [ -z "$stack_exists" ]; then
    echo "Creating CloudFormation stack..."
    aws cloudformation create-stack \
        --stack-name "$stack_name" \
        --template-body file://"$TEMPLATE_FILE" \
        --parameters \
            ParameterKey=LambdaImplementation,ParameterValue="$lambda_impl" \
            ParameterKey=OAuthClientId,ParameterValue="$oauth_client_id" \
            ParameterKey=OAuthClientSecret,ParameterValue="$oauth_client_secret" \
            ParameterKey=VirtualAgentName,ParameterValue="$virtual_agent_name" \
            ParameterKey=Region,ParameterValue="$region" \
            ParameterKey=CodeS3Bucket,ParameterValue="$s3_bucket" \
            ParameterKey=CodeS3Key,ParameterValue="$code_s3_key" \
            ParameterKey=FunctionName,ParameterValue="$function_name" \
            ParameterKey=RoleName,ParameterValue="$role_name" \
        --capabilities CAPABILITY_NAMED_IAM \
        --no-cli-pager
    
    echo "Waiting for stack creation to complete..."
    aws cloudformation wait stack-create-complete --stack-name "$stack_name"
    
    if [ $? -eq 0 ]; then
        echo "Stack created successfully!"
        aws cloudformation describe-stacks --stack-name "$stack_name" --query "Stacks[0].Outputs" --output table
    else
        echo "Error: Stack creation failed"
        exit 1
    fi
else
    echo "Updating CloudFormation stack..."
    aws cloudformation update-stack \
        --stack-name "$stack_name" \
        --template-body file://"$TEMPLATE_FILE" \
        --parameters \
            ParameterKey=LambdaImplementation,ParameterValue="$lambda_impl" \
            ParameterKey=OAuthClientId,ParameterValue="$oauth_client_id" \
            ParameterKey=OAuthClientSecret,ParameterValue="$oauth_client_secret" \
            ParameterKey=VirtualAgentName,ParameterValue="$virtual_agent_name" \
            ParameterKey=Region,ParameterValue="$region" \
            ParameterKey=CodeS3Bucket,ParameterValue="$s3_bucket" \
            ParameterKey=CodeS3Key,ParameterValue="$code_s3_key" \
            ParameterKey=FunctionName,ParameterValue="$function_name" \
            ParameterKey=RoleName,ParameterValue="$role_name" \
        --capabilities CAPABILITY_NAMED_IAM \
        --no-cli-pager
    
    if [ $? -eq 0 ]; then
        echo "Waiting for stack update to complete..."
        aws cloudformation wait stack-update-complete --stack-name "$stack_name"
        
        if [ $? -eq 0 ]; then
            echo "Stack updated successfully!"
            aws cloudformation describe-stacks --stack-name "$stack_name" --query "Stacks[0].Outputs" --output table
        else
            echo "Error: Stack update failed"
            exit 1
        fi
    else
        echo "No updates to be performed or update failed"
    fi
fi

