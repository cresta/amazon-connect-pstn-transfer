#!/bin/bash

# Build script for Python Lambda function
# Creates a deployment-ready zip file

set -e

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
LAMBDA_DIR="$PROJECT_ROOT/lambdas/pstn-transfer-py"
OUTPUT_ZIP="aws-lambda-connect-pstn-transfer-py.zip"

echo "=== Building Python Lambda ==="

# Run the package script
chmod +x "$LAMBDA_DIR/package.sh"
"$LAMBDA_DIR/package.sh"

# The package script creates the zip in the project root
ZIP_IN_LAMBDAS="$PROJECT_ROOT/$OUTPUT_ZIP"
if [ -f "$ZIP_IN_LAMBDAS" ]; then
    echo ""
    echo "Build successful: $OUTPUT_ZIP"
    echo "Package size: $(du -h "$ZIP_IN_LAMBDAS" | cut -f1)"
else
    echo "Error: Failed to create deployment package - $OUTPUT_ZIP not found"
    exit 1
fi

echo "=== Python Lambda build completed ==="
