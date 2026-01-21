#!/bin/bash

# Build script for TypeScript Lambda function
# Produces a zip file with the compiled JavaScript and node_modules for AWS Lambda

set -e

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Change to project root
cd "$PROJECT_ROOT" || exit 1

LAMBDA_DIR="lambdas/pstn-transfer-ts"
OUTPUT_ZIP="aws-lambda-connect-pstn-transfer-ts.zip"

echo "=== Building TypeScript Lambda Function ==="
echo ""

# Change to lambda directory
cd "$LAMBDA_DIR" || exit 1

# Install dependencies if node_modules doesn't exist
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm ci
fi

# Build TypeScript
echo "Compiling TypeScript..."
npm run build

if [ ! -d "dist" ]; then
    echo "Error: Build failed - dist directory not found"
    exit 1
fi

# Create deployment package
echo "Creating deployment package..."
npm run package

# The package script creates the zip in the parent directory (lambdas/)
# Check if it exists there
ZIP_IN_LAMBDAS="$PROJECT_ROOT/lambdas/$OUTPUT_ZIP"
if [ -f "$ZIP_IN_LAMBDAS" ]; then
    # Move it to project root to match Go build script behavior
    mv "$ZIP_IN_LAMBDAS" "$PROJECT_ROOT/$OUTPUT_ZIP"
elif [ -f "$PROJECT_ROOT/$OUTPUT_ZIP" ]; then
    # Already in project root (shouldn't happen, but handle it)
    echo "Zip file already in project root"
else
    echo "Error: Failed to create deployment package - $OUTPUT_ZIP not found"
    exit 1
fi

# Return to project root for final output
cd "$PROJECT_ROOT" || exit 1

echo ""
echo "Build successful: $OUTPUT_ZIP"
echo "Package size: $(du -h "$OUTPUT_ZIP" | cut -f1)"
