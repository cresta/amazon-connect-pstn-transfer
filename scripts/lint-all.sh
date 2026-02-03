#!/bin/bash

# Lint script that runs linting for all implementations
# Runs Go linting (gofmt, go vet), TypeScript linting (eslint), and Python linting (ruff)

# Note: We don't use set -e here because we want to run all linters
# even if one fails, then report the overall status

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Change to project root
cd "$PROJECT_ROOT" || exit 1

echo "=== Running All Linters ==="
echo ""

# Track if any linters failed
LINT_FAILED=0

# Run Go linting
echo "--- Running Go Linters ---"
if [ -d "lambdas/pstn-transfer-go" ]; then
    if command -v go &> /dev/null; then
        # Check gofmt
        if command -v gofmt &> /dev/null; then
            UNFORMATTED=$(gofmt -l ./lambdas/pstn-transfer-go)
            if [ -z "$UNFORMATTED" ]; then
                echo "✓ gofmt: all files formatted"
            else
                echo "✗ gofmt: files need formatting:"
                echo "$UNFORMATTED"
                LINT_FAILED=1
            fi
        else
            echo "⚠ gofmt not found, skipping"
        fi
        
        # Run go vet
        if go vet ./lambdas/pstn-transfer-go/...; then
            echo "✓ go vet: no issues found"
        else
            echo "✗ go vet: issues found"
            LINT_FAILED=1
        fi
    else
        echo "⚠ Go not found, skipping Go linting"
    fi
else
    echo "⚠ Go implementation directory not found, skipping Go linting"
fi
echo ""

# Run TypeScript linting
echo "--- Running TypeScript Linter ---"
if [ -d "lambdas/pstn-transfer-ts" ]; then
    if command -v npm &> /dev/null; then
        if ! cd lambdas/pstn-transfer-ts; then
            echo "✗ Failed to enter lambdas/pstn-transfer-ts directory"
            LINT_FAILED=1
        else
            if npm run lint; then
                echo "✓ TypeScript linting passed"
            else
                echo "✗ TypeScript linting failed"
                LINT_FAILED=1
            fi
            if ! cd "$PROJECT_ROOT"; then
                echo "✗ Failed to return to PROJECT_ROOT"
                exit 1
            fi
        fi
    else
        echo "⚠ npm not found, skipping TypeScript linting"
    fi
else
    echo "⚠ TypeScript implementation directory not found, skipping TypeScript linting"
fi
echo ""

# Run Python linting
echo "--- Running Python Linter ---"
if [ -d "lambdas/pstn-transfer-py" ]; then
    if command -v ruff &> /dev/null || python3 -m ruff --version &> /dev/null 2>&1; then
        if ! cd lambdas/pstn-transfer-py; then
            echo "✗ Failed to enter lambdas/pstn-transfer-py directory"
            LINT_FAILED=1
        else
            # Try ruff directly, fall back to python -m ruff
            if command -v ruff &> /dev/null; then
                if ruff check --fix src tests; then
                    echo "✓ Python linting passed"
                else
                    echo "✗ Python linting failed"
                    LINT_FAILED=1
                fi
            elif python3 -m ruff check --fix src tests; then
                echo "✓ Python linting passed"
            else
                echo "✗ Python linting failed"
                LINT_FAILED=1
            fi
            if ! cd "$PROJECT_ROOT"; then
                echo "✗ Failed to return to PROJECT_ROOT"
                exit 1
            fi
        fi
    else
        echo "⚠ ruff not found, skipping Python linting"
    fi
else
    echo "⚠ Python implementation directory not found, skipping Python linting"
fi
echo ""

# Summary
echo "=== Lint Summary ==="
if [ $LINT_FAILED -eq 0 ]; then
    echo "✓ All linters passed!"
    exit 0
else
    echo "✗ Some linters failed"
    exit 1
fi
