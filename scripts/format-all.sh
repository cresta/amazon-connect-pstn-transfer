#!/bin/bash

# Format script that formats code for all implementations
# Runs Go formatting (gofmt) and TypeScript formatting (prettier)

# Note: We don't use set -e here because we want to format all code
# even if one formatter fails, then report the overall status

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Change to project root
cd "$PROJECT_ROOT" || exit 1

echo "=== Formatting All Code ==="
echo ""

# Track if any formatters failed
FORMAT_FAILED=0

# Run Go formatting
echo "--- Formatting Go Code ---"
if [ -d "lambdas/pstn-transfer-go" ]; then
    if command -v gofmt &> /dev/null; then
        if gofmt -w ./lambdas/pstn-transfer-go; then
            echo "✓ Go code formatted"
        else
            echo "✗ Go formatting failed"
            FORMAT_FAILED=1
        fi
    else
        echo "⚠ gofmt not found, skipping Go formatting"
    fi
else
    echo "⚠ Go implementation directory not found, skipping Go formatting"
fi
echo ""

# Run TypeScript formatting
echo "--- Formatting TypeScript Code ---"
if [ -d "lambdas/pstn-transfer-ts" ]; then
    if command -v npm &> /dev/null; then
        cd lambdas/pstn-transfer-ts
        if npm run format; then
            echo "✓ TypeScript code formatted"
        else
            echo "✗ TypeScript formatting failed"
            FORMAT_FAILED=1
        fi
        cd "$PROJECT_ROOT"
    else
        echo "⚠ npm not found, skipping TypeScript formatting"
    fi
else
    echo "⚠ TypeScript implementation directory not found, skipping TypeScript formatting"
fi
echo ""

# Summary
echo "=== Format Summary ==="
if [ $FORMAT_FAILED -eq 0 ]; then
    echo "✓ All code formatted successfully!"
    exit 0
else
    echo "✗ Some formatting failed"
    exit 1
fi
