#!/bin/bash

# Build the zip
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap main.go && zip -j aws-lambda-connect-pstn-transfer.zip bootstrap

# Get the account ID
account_id=$(aws sts get-caller-identity --query "Account" --output text)

role_name="aws-lambda-connect-pstn-transfer-role"
function_name="aws-lambda-connect-pstn-transfer"

# Check if the role exists
role_exists=$(aws iam get-role --role-name $role_name --query "Role.RoleName" --output text 2>/dev/null)

if [ -z "$role_exists" ]; then
    echo "Creating IAM role..."
    # Create the role with trust policy
    aws iam create-role \
        --role-name $role_name \
        --assume-role-policy-document '{
            "Version": "2012-10-17",
            "Statement": [{
                "Effect": "Allow",
                "Principal": {
                    "Service": "lambda.amazonaws.com"
                },
                "Action": "sts:AssumeRole"
            }]
        }' \
        --no-cli-pager

    # Attach basic Lambda execution policy
    aws iam attach-role-policy \
        --role-name $role_name \
        --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole \
        --no-cli-pager

    # Wait for role to propagate
    echo "Waiting for role to propagate..."
    sleep 10
fi

# Check if the function already exists
already_exists=$(aws lambda get-function --function-name $function_name --query "Configuration.FunctionName" --output text)

if [ -z "$already_exists" ]; then
    role_arn=$(aws iam get-role --role-name $role_name --query "Role.Arn" --output text 2>/dev/null)
    # Try to create the function, if it already exists, update the code
    aws lambda create-function --function-name $function_name \
        --runtime provided.al2023 --handler bootstrap \
        --zip-file fileb://aws-lambda-connect-pstn-transfer.zip \
        --role $role_arn \
        --architectures arm64 \
        --no-cli-pager \
        --environment "{\"Variables\":$(cat var.json)}"
else
    # Update the function code
    aws lambda update-function-code --function-name $function_name \
        --zip-file fileb://aws-lambda-connect-pstn-transfer.zip \
        --no-cli-pager

    aws lambda update-function-configuration --function-name $function_name \
        --environment "{\"Variables\":$(cat var.json)}" \
        --no-cli-pager
fi
