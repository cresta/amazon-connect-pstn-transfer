#!/bin/bash

# Build script that builds Go, TypeScript, and Python Lambda functions
# Runs build-go-lambda.sh, build-typescript-lambda.sh, and build-python-lambda.sh

set -e

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo "=== Building All Lambda Functions ==="
echo ""

"$SCRIPT_DIR/build-go-lambda.sh"
echo ""

"$SCRIPT_DIR/build-typescript-lambda.sh"
echo ""

"$SCRIPT_DIR/build-python-lambda.sh"
echo ""

echo "=== All builds completed successfully ==="
