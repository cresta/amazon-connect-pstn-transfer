#!/bin/bash

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
