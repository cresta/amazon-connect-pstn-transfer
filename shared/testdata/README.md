# Shared Integration Tests

This directory contains shared integration tests that validate both the Go and TypeScript implementations of the PSTN Transfer Lambda function.

## Overview

The tests build both implementations as executables and run them against a mock HTTP server to ensure consistent behavior across both languages. Each test scenario is defined as a JSON file in the `scenarios/` directory.

## Prerequisites

- Node.js (v24 or higher)
- Go (v1.21 or higher)
- npm dependencies installed in this directory
- Go dependencies installed in `lambdas/pstn-transfer-go/`
- TypeScript dependencies installed in `lambdas/pstn-transfer-ts/`

## Running Tests

### From this directory

```bash
cd shared/testdata
npm install  # First time only
npm test
```

**Note**: These tests are separate from the lambda-specific unit tests. Run the TypeScript lambda tests with `cd lambdas/pstn-transfer-ts && npm test` and the Go lambda tests with `cd lambdas/pstn-transfer-go && go test ./...`.

## Test Structure

- **`scenarios/`**: Contains test scenario JSON files organized by category
  - `handler-scenarios/`: Scenarios that test the full handler execution
- **`mock-server/`**: Mock HTTP server implementation for simulating API responses
- **`test-runner/`**: Utilities for building and executing Go and TypeScript handlers
- **`scenarios.test.ts`**: Main test file that loads and executes all scenarios

## Adding New Scenarios

1. Create a new JSON file in `scenarios/handler-scenarios/` (or appropriate subdirectory)
2. Follow the existing scenario format:
   ```json
   {
     "name": "scenario_name",
     "description": "What this scenario tests",
     "mock": {
       "path": "/api/endpoint",
       "method": "POST",
       "responses": [
         {
           "status": 200,
           "body": { "key": "value" }
         }
       ]
     },
     "test": {
       "type": "handler",
       "action": "getPSTNTransferData"
     },
     "expectations": {
       "status": 200,
       "body": { "key": "value" }
     }
   }
   ```
3. Run tests to verify the new scenario works for both implementations

## Scenario Types

- **Handler scenarios**: Test the full Lambda handler execution, including retry logic, authentication, and error handling

## Mock Server

The mock server simulates API responses and supports:
- Multiple response attempts (for testing retry logic)
- Custom status codes and response bodies
- Custom headers (e.g., `Retry-After` for 429 responses)

## How It Works

1. Tests load all scenario JSON files from `scenarios/` subdirectories
2. For each scenario:
   - Builds the Go binary and TypeScript handler
   - Starts a mock HTTP server
   - Registers the scenario's mock responses
   - Executes both implementations with the same event
   - Compares outputs to ensure consistency
3. Validates that both implementations:
   - Return the same results for success cases
   - Fail in the same way for error cases
   - Handle retries consistently
