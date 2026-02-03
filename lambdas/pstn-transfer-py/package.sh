#!/bin/bash

# Package script for Python Lambda function
# Creates a deployment-ready zip file

set -e

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Read version from VERSION file
VERSION=$(cat ../../VERSION | tr -d '[:space:]')

echo "=== Packaging Python Lambda (version: $VERSION) ==="

# Clean up any previous package
rm -rf package
rm -f ../aws-lambda-connect-pstn-transfer-py.zip

# Create package directory
mkdir -p package

# Install dependencies
echo "Installing dependencies..."
pip install -r requirements.txt -t package/ --quiet

# Copy source code
echo "Copying source code..."
cp -r src package/

# Inject version into the package
echo "Injecting version..."
cat > package/src/version.py << EOF
"""
Version information for the Lambda function
This is injected at build time
"""

VERSION: str = "${VERSION}"
EOF

# Create zip file
echo "Creating zip file..."
cd package
zip -r ../../aws-lambda-connect-pstn-transfer-py.zip . -x "*.pyc" -x "__pycache__/*" -x "*.egg-info/*" --quiet
cd ..

# Clean up
rm -rf package

echo "=== Package created: lambdas/aws-lambda-connect-pstn-transfer-py.zip ==="
