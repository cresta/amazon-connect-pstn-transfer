#!/bin/bash

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Change to project root directory
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

echo "=== AWS Lambda Deployment Configuration ==="
echo ""

# Check if var.json exists
if [ -f "var.json" ]; then
    echo "Found existing var.json file."
    echo ""
    
    # Extract values from var.json
    oauth_secret_arn=$(extract_json_value "oauthSecretArn" "var.json")
    oauth_client_id=$(extract_json_value "oauthClientId" "var.json")
    oauth_client_secret=$(extract_json_value "oauthClientSecret" "var.json")
    virtual_agent_name=$(extract_json_value "virtualAgentName" "var.json")
    region=$(extract_json_value "region" "var.json")
    api_domain=$(extract_json_value "apiDomain" "var.json")
    auth_domain=$(extract_json_value "authDomain" "var.json")
    
    # Set default if region is empty
    if [ -z "$region" ]; then
        region="us-west-2-prod"
    fi
    
    # Display current values
    echo "Current configuration:"
    if [ -n "$oauth_secret_arn" ]; then
        echo "  OAuth Secret ARN: $oauth_secret_arn"
    else
        echo "  OAuth Client ID: ${oauth_client_id:0:10}..." # Show only first 10 chars for security
        echo "  OAuth Client Secret: ${oauth_client_secret:0:10}..." # Show only first 10 chars for security
    fi
    echo "  Virtual Agent Name: $virtual_agent_name"
    echo "  Region: $region"
    if [ -n "$api_domain" ]; then
        echo "  API Domain: $api_domain"
    fi
    if [ -n "$auth_domain" ]; then
        echo "  Auth Domain: $auth_domain"
    fi
    echo ""
    
    # Ask for confirmation
    read -p "Use these values? (y/n): " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        echo ""
        # Ask if using Secrets Manager
        read -p "Use AWS Secrets Manager for OAuth credentials? (y/n, default: n): " use_secrets_manager
        if [[ "$use_secrets_manager" =~ ^[Yy]$ ]]; then
            read -p "Enter OAuth Secret ARN (required): " oauth_secret_arn
            if [ -z "$oauth_secret_arn" ]; then
                echo "Error: OAuth Secret ARN is required"
                exit 1
            fi
            oauth_client_id=""
            oauth_client_secret=""
        else
            oauth_secret_arn=""
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
        
        read -p "Enter API Domain (optional, e.g., api-customer-profile.cresta.com, must be used with authDomain): " api_domain
        
        read -p "Enter Auth Domain (optional, e.g., auth.us-west-2-prod.cresta.ai, must be used with apiDomain): " auth_domain
        
        # Validate that apiDomain and authDomain are used together
        if ([ -n "$api_domain" ] && [ -z "$auth_domain" ]) || ([ -z "$api_domain" ] && [ -n "$auth_domain" ]); then
            echo "Error: apiDomain and authDomain must be provided together"
            exit 1
        fi
    fi
else
    # Prompt for values if var.json doesn't exist
    echo "How would you like to provide OAuth credentials?"
    echo "1) AWS Secrets Manager (recommended for production)"
    echo "2) Environment variables"
    read -p "Enter choice [1 or 2] (default: 2): " auth_choice
    
    if [ -z "$auth_choice" ]; then
        auth_choice="2"
    fi
    
    if [ "$auth_choice" = "1" ]; then
        read -p "Enter OAuth Secret ARN (required): " oauth_secret_arn
        if [ -z "$oauth_secret_arn" ]; then
            echo "Error: OAuth Secret ARN is required"
            exit 1
        fi
        oauth_client_id=""
        oauth_client_secret=""
    else
        oauth_secret_arn=""
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
    
    read -p "Enter API Domain (optional, e.g., api-customer-profile.cresta.com, must be used with authDomain): " api_domain
    
    read -p "Enter Auth Domain (optional, e.g., auth.us-west-2-prod.cresta.ai, must be used with apiDomain): " auth_domain
    
    # Validate that apiDomain and authDomain are used together
    if ([ -n "$api_domain" ] && [ -z "$auth_domain" ]) || ([ -z "$api_domain" ] && [ -n "$auth_domain" ]); then
        echo "Error: apiDomain and authDomain must be provided together"
        exit 1
    fi
fi

# Validate required fields
if [ -z "$oauth_secret_arn" ] && [ -z "$oauth_client_id" ]; then
    echo "Error: Either OAuth Secret ARN or OAuth Client ID is required"
    exit 1
fi

if [ -z "$oauth_secret_arn" ] && [ -z "$oauth_client_secret" ]; then
    echo "Error: OAuth Client Secret is required when not using Secrets Manager"
    exit 1
fi

if [ -z "$virtual_agent_name" ]; then
    echo "Error: Virtual Agent Name is required"
    exit 1
fi

# Create or update var.json file
if command -v jq &> /dev/null; then
    jq -n \
        --arg oauthSecretArn "${oauth_secret_arn:-}" \
        --arg oauthClientId "${oauth_client_id:-}" \
        --arg oauthClientSecret "${oauth_client_secret:-}" \
        --arg virtualAgentName "$virtual_agent_name" \
        --arg region "$region" \
        --arg apiDomain "${api_domain:-}" \
        --arg authDomain "${auth_domain:-}" \
        '{oauthSecretArn: $oauthSecretArn, oauthClientId: $oauthClientId, oauthClientSecret: $oauthClientSecret, virtualAgentName: $virtualAgentName, region: $region, apiDomain: $apiDomain, authDomain: $authDomain}' > var.json
else
    # Fallback to manual JSON creation (basic escaping)
    oauth_secret_arn_escaped=$(echo "${oauth_secret_arn:-}" | sed 's/"/\\"/g')
    oauth_client_id_escaped=$(echo "${oauth_client_id:-}" | sed 's/"/\\"/g')
    oauth_client_secret_escaped=$(echo "${oauth_client_secret:-}" | sed 's/"/\\"/g')
    virtual_agent_name_escaped=$(echo "$virtual_agent_name" | sed 's/"/\\"/g')
    region_escaped=$(echo "$region" | sed 's/"/\\"/g')
    api_domain_escaped=$(echo "${api_domain:-}" | sed 's/"/\\"/g')
    auth_domain_escaped=$(echo "${auth_domain:-}" | sed 's/"/\\"/g')
    cat > var.json <<EOF
{
    "oauthSecretArn": "$oauth_secret_arn_escaped",
    "oauthClientId": "$oauth_client_id_escaped",
    "oauthClientSecret": "$oauth_client_secret_escaped",
    "virtualAgentName": "$virtual_agent_name_escaped",
    "region": "$region_escaped",
    "apiDomain": "$api_domain_escaped",
    "authDomain": "$auth_domain_escaped"
}
EOF
fi

echo ""
echo "Configuration saved to var.json"
echo ""

# Ask which implementation to deploy
echo "Which Lambda implementation would you like to deploy?"
echo "1) Go (provided.al2023 runtime, ARM64)"
echo "2) TypeScript (Node.js runtime, ARM64)"
echo "3) Python (Python 3.14 runtime, x86_64)"
read -p "Enter choice [1, 2, or 3] (default: 1): " implementation_choice

if [ -z "$implementation_choice" ]; then
    implementation_choice="1"
fi

case "$implementation_choice" in
    1)
        IMPLEMENTATION="go"
        RUNTIME="provided.al2023"
        HANDLER="bootstrap"
        ARCHITECTURE="arm64"
        ZIP_FILE="aws-lambda-connect-pstn-transfer-go.zip"
        BUILD_SCRIPT="build-go-lambda.sh"
        ;;
    2)
        IMPLEMENTATION="typescript"
        RUNTIME="nodejs24.x"
        HANDLER="handler.handler"
        ARCHITECTURE="arm64"
        ZIP_FILE="aws-lambda-connect-pstn-transfer-ts.zip"
        BUILD_SCRIPT="build-typescript-lambda.sh"
        ;;
    3)
        IMPLEMENTATION="python"
        RUNTIME="python3.14"
        HANDLER="src.handler.handler"
        ARCHITECTURE="x86_64"
        ZIP_FILE="aws-lambda-connect-pstn-transfer-py.zip"
        BUILD_SCRIPT="build-python-lambda.sh"
        ;;
    *)
        echo "Error: Invalid choice. Please enter 1, 2, or 3."
        exit 1
        ;;
esac

echo ""
echo "Deploying $IMPLEMENTATION implementation..."
echo "  Runtime: $RUNTIME"
echo "  Handler: $HANDLER"
echo "  Architecture: $ARCHITECTURE"
echo ""

# Build the zip using the appropriate build script
if ! "$SCRIPT_DIR/$BUILD_SCRIPT"; then
    echo "Error: Build script failed: $BUILD_SCRIPT (exit code: $?)"
    exit 1
fi

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

    # Add Secrets Manager permissions if using Secrets Manager
    if [ -n "$oauth_secret_arn" ]; then
        echo "Adding Secrets Manager permissions to IAM role..."
        aws iam put-role-policy \
            --role-name $role_name \
            --policy-name SecretsManagerAccess \
            --policy-document "{
                \"Version\": \"2012-10-17\",
                \"Statement\": [{
                    \"Effect\": \"Allow\",
                    \"Action\": [\"secretsmanager:GetSecretValue\"],
                    \"Resource\": \"$oauth_secret_arn\"
                }]
            }" \
            --no-cli-pager
    fi

    # Wait for role to propagate
    echo "Waiting for role to propagate..."
    sleep 10
else
    # Manage Secrets Manager permissions based on authentication method
    if [ -n "$oauth_secret_arn" ]; then
        echo "Ensuring Secrets Manager permissions on IAM role..."
        aws iam put-role-policy \
            --role-name $role_name \
            --policy-name SecretsManagerAccess \
            --policy-document "{
                \"Version\": \"2012-10-17\",
                \"Statement\": [{
                    \"Effect\": \"Allow\",
                    \"Action\": [\"secretsmanager:GetSecretValue\"],
                    \"Resource\": \"$oauth_secret_arn\"
                }]
            }" \
            --no-cli-pager
    else
        # Remove Secrets Manager policy if switching away from Secrets Manager
        echo "Removing Secrets Manager permissions from IAM role..."
        aws iam delete-role-policy \
            --role-name $role_name \
            --policy-name SecretsManagerAccess \
            --no-cli-pager 2>/dev/null || {
            # Policy might not exist, which is fine - suppress error
            echo "Note: SecretsManagerAccess policy not found (may have been already removed)"
        }
    fi
fi

# Check if the function already exists
already_exists=$(aws lambda get-function --function-name "$function_name" --query "Configuration.FunctionName" --output text 2>/dev/null)

if [ -z "$already_exists" ]; then
    role_arn=$(aws iam get-role --role-name $role_name --query "Role.Arn" --output text 2>/dev/null)
    # Try to create the function, if it already exists, update the code
    aws lambda create-function --function-name $function_name \
        --runtime "$RUNTIME" --handler "$HANDLER" \
        --zip-file "fileb://$ZIP_FILE" \
        --role $role_arn \
        --architectures "$ARCHITECTURE" \
        --no-cli-pager \
        --environment "{\"Variables\":$(cat var.json)}"
else
    # Update the function code
    aws lambda update-function-code --function-name $function_name \
        --zip-file "fileb://$ZIP_FILE" \
        --no-cli-pager

    # Wait for the code update to complete
    echo "Waiting for function code update to complete..."
    aws lambda wait function-updated --function-name $function_name

    # Update runtime and handler if switching implementations
    aws lambda update-function-configuration --function-name $function_name \
        --runtime "$RUNTIME" \
        --handler "$HANDLER" \
        --environment "{\"Variables\":$(cat var.json)}" \
        --no-cli-pager
fi

echo ""
echo "Deployment complete!"
echo "Function name: $function_name"
echo "Implementation: $IMPLEMENTATION"
echo "Runtime: $RUNTIME"
