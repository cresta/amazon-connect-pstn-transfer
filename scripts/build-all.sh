#!/bin/bash

# Build script that builds both Go and TypeScript Lambda functions
# Runs build-go-lambda.sh and build-typescript-lambda.sh

set -e

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo "=== Building All Lambda Functions ==="
echo ""

"$SCRIPT_DIR/build-go-lambda.sh"
echo ""

"$SCRIPT_DIR/build-typescript-lambda.sh"
echo ""

echo "=== All builds completed successfully ==="
