#!/bin/bash

# Change to project root directory (parent of scripts directory)
cd "$(dirname "$0")/.." || exit 1

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

echo "=== AWS Lambda Deployment Configuration ==="
echo ""

# Check if var.json exists
if [ -f "var.json" ]; then
    echo "Found existing var.json file."
    echo ""
    
    # Extract values from var.json
    oauth_client_id=$(extract_json_value "oauthClientId" "var.json")
    oauth_client_secret=$(extract_json_value "oauthClientSecret" "var.json")
    virtual_agent_name=$(extract_json_value "virtualAgentName" "var.json")
    region=$(extract_json_value "region" "var.json")
    
    # Set default if region is empty
    if [ -z "$region" ]; then
        region="us-west-2-prod"
    fi
    
    # Display current values
    echo "Current configuration:"
    echo "  OAuth Client ID: ${oauth_client_id:0:10}..." # Show only first 10 chars for security
    echo "  OAuth Client Secret: ${oauth_client_secret:0:10}..." # Show only first 10 chars for security
    echo "  Virtual Agent Name: $virtual_agent_name"
    echo "  Region: $region"
    echo ""
    
    # Ask for confirmation
    read -p "Use these values? (y/n): " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        echo ""
        # Prompt for new values
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
    fi
else
    # Prompt for values if var.json doesn't exist
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
fi

# Validate required fields
if [ -z "$oauth_client_id" ]; then
    echo "Error: OAuth Client ID is required"
    exit 1
fi

if [ -z "$oauth_client_secret" ]; then
    echo "Error: OAuth Client Secret is required"
    exit 1
fi

if [ -z "$virtual_agent_name" ]; then
    echo "Error: Virtual Agent Name is required"
    exit 1
fi

# Create or update var.json file
if command -v jq &> /dev/null; then
    jq -n \
        --arg oauthClientId "$oauth_client_id" \
        --arg oauthClientSecret "$oauth_client_secret" \
        --arg virtualAgentName "$virtual_agent_name" \
        --arg region "$region" \
        '{oauthClientId: $oauthClientId, oauthClientSecret: $oauthClientSecret, virtualAgentName: $virtualAgentName, region: $region}' > var.json
else
    # Fallback to manual JSON creation (basic escaping)
    oauth_client_id_escaped=$(echo "$oauth_client_id" | sed 's/"/\\"/g')
    oauth_client_secret_escaped=$(echo "$oauth_client_secret" | sed 's/"/\\"/g')
    virtual_agent_name_escaped=$(echo "$virtual_agent_name" | sed 's/"/\\"/g')
    region_escaped=$(echo "$region" | sed 's/"/\\"/g')
    cat > var.json <<EOF
{
    "oauthClientId": "$oauth_client_id_escaped",
    "oauthClientSecret": "$oauth_client_secret_escaped",
    "virtualAgentName": "$virtual_agent_name_escaped",
    "region": "$region_escaped"
}
EOF
fi

echo ""
echo "Configuration saved to var.json"
echo ""

# Build the zip
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap . && zip -j aws-lambda-connect-pstn-transfer.zip bootstrap

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
