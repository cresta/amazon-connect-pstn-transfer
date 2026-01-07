#!/bin/bash

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Change to project root
cd "$PROJECT_ROOT" || exit 1

# Function to extract JSON value
extract_json_value() {
    local key=$1
    local file=$2
    if command -v jq &> /dev/null; then
        jq -r ".$key // empty" "$file" 2>/dev/null
    else
        grep -o "\"$key\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" "$file" 2>/dev/null | sed "s/\"$key\"[[:space:]]*:[[:space:]]*\"\(.*\)\"/\1/"
    fi
}

echo "=== AWS Lambda CloudFormation Deployment ==="
echo ""

STACK_NAME="aws-lambda-connect-pstn-transfer-stack"
FUNCTION_NAME="aws-lambda-connect-pstn-transfer"
ROLE_NAME="aws-lambda-connect-pstn-transfer-role"
CODE_ZIP="aws-lambda-connect-pstn-transfer.zip"
S3_KEY="aws-lambda-connect-pstn-transfer.zip"
TEMPLATE_FILE="$SCRIPT_DIR/template.yaml"

# Check if var.json exists
if [ -f "var.json" ]; then
    echo "Found existing var.json file."
    echo ""
    
    # Extract values from var.json
    api_key=$(extract_json_value "apiKey" "var.json")
    virtual_agent_name=$(extract_json_value "virtualAgentName" "var.json")
    api_domain=$(extract_json_value "apiDomain" "var.json")
    
    # Set default if apiDomain is empty
    if [ -z "$api_domain" ]; then
        api_domain="https://api.us-west-2-prod.cresta.com"
    fi
    
    # Display current values
    echo "Current configuration:"
    echo "  API Key: ${api_key:0:10}..." # Show only first 10 chars for security
    echo "  Virtual Agent Name: $virtual_agent_name"
    echo "  API Domain: $api_domain"
    echo ""
    
    # Ask for confirmation
    read -p "Use these values? (y/n): " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        echo ""
        # Prompt for new values
        read -p "Enter API Key (required): " api_key
        if [ -z "$api_key" ]; then
            echo "Error: API Key is required"
            exit 1
        fi
        
        read -p "Enter Virtual Agent Name (required, format: customers/{customer}/profiles/{profile}/virtualAgents/{virtualAgentID}): " virtual_agent_name
        if [ -z "$virtual_agent_name" ]; then
            echo "Error: Virtual Agent Name is required"
            exit 1
        fi
        
        read -p "Enter API Domain (optional, default: https://api.us-west-2-prod.cresta.com): " api_domain
        if [ -z "$api_domain" ]; then
            api_domain="https://api.us-west-2-prod.cresta.com"
        fi
    fi
else
    # Prompt for values if var.json doesn't exist
    read -p "Enter API Key (required): " api_key
    if [ -z "$api_key" ]; then
        echo "Error: API Key is required"
        exit 1
    fi
    
    read -p "Enter Virtual Agent Name (required, format: customers/{customer}/profiles/{profile}/virtualAgents/{virtualAgentID}): " virtual_agent_name
    if [ -z "$virtual_agent_name" ]; then
        echo "Error: Virtual Agent Name is required"
        exit 1
    fi
    
    read -p "Enter API Domain (optional, default: https://api.us-west-2-prod.cresta.com): " api_domain
    if [ -z "$api_domain" ]; then
        api_domain="https://api.us-west-2-prod.cresta.com"
    fi
fi

# Validate required fields
if [ -z "$api_key" ]; then
    echo "Error: API Key is required"
    exit 1
fi

if [ -z "$virtual_agent_name" ]; then
    echo "Error: Virtual Agent Name is required"
    exit 1
fi

# Prompt for S3 bucket
read -p "Enter S3 bucket name for Lambda code (required): " s3_bucket
if [ -z "$s3_bucket" ]; then
    echo "Error: S3 bucket name is required"
    exit 1
fi

# Create or update var.json file
if command -v jq &> /dev/null; then
    jq -n \
        --arg apiKey "$api_key" \
        --arg virtualAgentName "$virtual_agent_name" \
        --arg apiDomain "$api_domain" \
        '{apiKey: $apiKey, virtualAgentName: $virtualAgentName, apiDomain: $apiDomain}' > var.json
else
    # Fallback to manual JSON creation (basic escaping)
    api_key_escaped=$(echo "$api_key" | sed 's/"/\\"/g')
    virtual_agent_name_escaped=$(echo "$virtual_agent_name" | sed 's/"/\\"/g')
    api_domain_escaped=$(echo "$api_domain" | sed 's/"/\\"/g')
    cat > var.json <<EOF
{
    "apiKey": "$api_key_escaped",
    "virtualAgentName": "$virtual_agent_name_escaped",
    "apiDomain": "$api_domain_escaped"
}
EOF
fi

echo ""
echo "Configuration saved to var.json"
echo ""

# Build the zip
echo "Building Lambda function..."
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap main.go && zip -j "$CODE_ZIP" bootstrap

if [ ! -f "$CODE_ZIP" ]; then
    echo "Error: Failed to create deployment package"
    exit 1
fi

# Upload to S3
echo "Uploading deployment package to S3..."
aws s3 cp "$CODE_ZIP" "s3://$s3_bucket/$S3_KEY" --no-cli-pager

if [ $? -ne 0 ]; then
    echo "Error: Failed to upload to S3"
    exit 1
fi

echo "Deployment package uploaded successfully"
echo ""

# Check if stack exists
stack_exists=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" --query "Stacks[0].StackName" --output text 2>/dev/null)

if [ -z "$stack_exists" ]; then
    echo "Creating CloudFormation stack..."
    aws cloudformation create-stack \
        --stack-name "$STACK_NAME" \
        --template-body file://"$TEMPLATE_FILE" \
        --parameters \
            ParameterKey=ApiKey,ParameterValue="$api_key" \
            ParameterKey=VirtualAgentName,ParameterValue="$virtual_agent_name" \
            ParameterKey=ApiDomain,ParameterValue="$api_domain" \
            ParameterKey=CodeS3Bucket,ParameterValue="$s3_bucket" \
            ParameterKey=CodeS3Key,ParameterValue="$S3_KEY" \
            ParameterKey=FunctionName,ParameterValue="$FUNCTION_NAME" \
            ParameterKey=RoleName,ParameterValue="$ROLE_NAME" \
        --capabilities CAPABILITY_NAMED_IAM \
        --no-cli-pager
    
    echo "Waiting for stack creation to complete..."
    aws cloudformation wait stack-create-complete --stack-name "$STACK_NAME"
    
    if [ $? -eq 0 ]; then
        echo "Stack created successfully!"
        aws cloudformation describe-stacks --stack-name "$STACK_NAME" --query "Stacks[0].Outputs" --output table
    else
        echo "Error: Stack creation failed"
        exit 1
    fi
else
    echo "Updating CloudFormation stack..."
    aws cloudformation update-stack \
        --stack-name "$STACK_NAME" \
        --template-body file://"$TEMPLATE_FILE" \
        --parameters \
            ParameterKey=ApiKey,ParameterValue="$api_key" \
            ParameterKey=VirtualAgentName,ParameterValue="$virtual_agent_name" \
            ParameterKey=ApiDomain,ParameterValue="$api_domain" \
            ParameterKey=CodeS3Bucket,ParameterValue="$s3_bucket" \
            ParameterKey=CodeS3Key,ParameterValue="$S3_KEY" \
            ParameterKey=FunctionName,ParameterValue="$FUNCTION_NAME" \
            ParameterKey=RoleName,ParameterValue="$ROLE_NAME" \
        --capabilities CAPABILITY_NAMED_IAM \
        --no-cli-pager
    
    if [ $? -eq 0 ]; then
        echo "Waiting for stack update to complete..."
        aws cloudformation wait stack-update-complete --stack-name "$STACK_NAME"
        
        if [ $? -eq 0 ]; then
            echo "Stack updated successfully!"
            aws cloudformation describe-stacks --stack-name "$STACK_NAME" --query "Stacks[0].Outputs" --output table
        else
            echo "Error: Stack update failed"
            exit 1
        fi
    else
        echo "No updates to be performed or update failed"
    fi
fi

