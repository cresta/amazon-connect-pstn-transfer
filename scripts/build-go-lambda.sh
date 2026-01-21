#!/bin/bash

# Build script for Go Lambda function
# Produces a zip file with the bootstrap executable for AWS Lambda

set -e

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Change to project root
cd "$PROJECT_ROOT" || exit 1

LAMBDA_DIR="lambdas/pstn-transfer-go"
OUTPUT_ZIP="aws-lambda-connect-pstn-transfer-go.zip"
BUILD_DIR=$(mktemp -d)
trap "rm -rf $BUILD_DIR" EXIT

echo "=== Building Go Lambda Function ==="
echo ""

# Read version from VERSION file
VERSION=$(cat "$PROJECT_ROOT/VERSION" | tr -d '[:space:]')
if [ -z "$VERSION" ]; then
    echo "Error: VERSION file is empty or not found"
    exit 1
fi

# Build the Lambda function
# Using provided.al2023 runtime with bootstrap handler
# Inject version via ldflags
echo "Building for Linux ARM64 with version $VERSION..."
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -ldflags "-X main.Version=$VERSION" -o "$BUILD_DIR/bootstrap" "./$LAMBDA_DIR"

if [ ! -f "$BUILD_DIR/bootstrap" ]; then
    echo "Error: Build failed - bootstrap executable not found"
    exit 1
fi

# Create zip file with bootstrap executable
echo "Creating deployment package..."
cd "$BUILD_DIR" || exit 1
zip -j "$PROJECT_ROOT/$OUTPUT_ZIP" bootstrap

cd "$PROJECT_ROOT" || exit 1

if [ ! -f "$OUTPUT_ZIP" ]; then
    echo "Error: Failed to create deployment package"
    exit 1
fi

echo "Build successful: $OUTPUT_ZIP"
echo "Package size: $(du -h "$OUTPUT_ZIP" | cut -f1)"
