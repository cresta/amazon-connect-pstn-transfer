#!/bin/bash

# Test script that runs all tests for the project
# Runs Go tests, TypeScript tests, Python tests, and shared integration tests

# Note: We don't use set -e here because we want to run all tests
# even if one fails, then report the overall status

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Change to project root
cd "$PROJECT_ROOT" || exit 1

echo "=== Running All Tests ==="
echo ""

# Track if any tests failed
TESTS_FAILED=0

# Run Go tests
echo "--- Running Go Tests ---"
if [ -d "lambdas/pstn-transfer-go" ]; then
    if command -v go &> /dev/null; then
        if go test -v ./lambdas/pstn-transfer-go/...; then
            echo "✓ Go tests passed"
        else
            echo "✗ Go tests failed"
            TESTS_FAILED=1
        fi
    else
        echo "⚠ Go not found, skipping Go tests"
    fi
else
    echo "⚠ Go implementation directory not found, skipping Go tests"
fi
echo ""

# Run TypeScript tests
echo "--- Running TypeScript Tests ---"
if [ -d "lambdas/pstn-transfer-ts" ]; then
    if command -v npm &> /dev/null; then
        if ! cd lambdas/pstn-transfer-ts; then
            echo "✗ Failed to enter lambdas/pstn-transfer-ts directory"
            TESTS_FAILED=1
        else
            if npm test; then
                echo "✓ TypeScript tests passed"
            else
                echo "✗ TypeScript tests failed"
                TESTS_FAILED=1
            fi
            if ! cd "$PROJECT_ROOT"; then
                echo "✗ Failed to return to PROJECT_ROOT"
                exit 1
            fi
        fi
    else
        echo "⚠ npm not found, skipping TypeScript tests"
    fi
else
    echo "⚠ TypeScript implementation directory not found, skipping TypeScript tests"
fi
echo ""

# Run Python tests
echo "--- Running Python Tests ---"
if [ -d "lambdas/pstn-transfer-py" ]; then
    if command -v python3 &> /dev/null || command -v python &> /dev/null; then
        if ! cd lambdas/pstn-transfer-py; then
            echo "✗ Failed to enter lambdas/pstn-transfer-py directory"
            TESTS_FAILED=1
        else
            # Try pytest first, fall back to python -m pytest
            if command -v pytest &> /dev/null; then
                if pytest; then
                    echo "✓ Python tests passed"
                else
                    echo "✗ Python tests failed"
                    TESTS_FAILED=1
                fi
            elif python3 -m pytest --version &> /dev/null; then
                if python3 -m pytest; then
                    echo "✓ Python tests passed"
                else
                    echo "✗ Python tests failed"
                    TESTS_FAILED=1
                fi
            else
                echo "⚠ pytest not found, skipping Python tests"
            fi
            if ! cd "$PROJECT_ROOT"; then
                echo "✗ Failed to return to PROJECT_ROOT"
                exit 1
            fi
        fi
    else
        echo "⚠ Python not found, skipping Python tests"
    fi
else
    echo "⚠ Python implementation directory not found, skipping Python tests"
fi
echo ""

# Run shared integration tests
echo "--- Running Shared Integration Tests ---"
if [ -d "shared/testdata" ]; then
    if command -v npm &> /dev/null; then
        if ! cd shared/testdata; then
            echo "✗ Failed to enter shared/testdata directory"
            TESTS_FAILED=1
        else
            if npm test; then
                echo "✓ Shared integration tests passed"
            else
                echo "✗ Shared integration tests failed"
                TESTS_FAILED=1
            fi
            if ! cd "$PROJECT_ROOT"; then
                echo "✗ Failed to return to PROJECT_ROOT"
                exit 1
            fi
        fi
    else
        echo "⚠ npm not found, skipping shared integration tests"
    fi
else
    echo "⚠ Shared testdata directory not found, skipping shared integration tests"
fi
echo ""

# Summary
echo "=== Test Summary ==="
if [ $TESTS_FAILED -eq 0 ]; then
    echo "✓ All tests passed!"
    exit 0
else
    echo "✗ Some tests failed"
    exit 1
fi
