# PSTN Transfer TypeScript Lambda

TypeScript implementation of the Amazon Connect PSTN Transfer Lambda function, matching the Go implementation exactly.

## Overview

This TypeScript implementation provides the same functionality as the Go implementation:
- `get_pstn_transfer_data`: Generates PSTN transfer data (phone number and DTMF sequence)
- `get_handoff_data`: Fetches AI agent handoff data

## Development

### Prerequisites

- Node.js 24+ 
- npm or yarn

### Setup

```bash
cd lambdas/pstn-transfer-ts
npm install
```

### Build

```bash
npm run build
```

This compiles TypeScript to JavaScript in the `dist/` directory.

### Test

```bash
npm test
```

### Package for Lambda

```bash
npm run package
```

This creates a zip file at `../aws-lambda-connect-pstn-transfer-ts.zip` ready for Lambda deployment.

## Structure

- `src/handler.ts` - Main Lambda handler entry point
- `src/handlers.ts` - Business logic handlers for each action
- `src/client.ts` - API client for making HTTP requests
- `src/httpclient.ts` - HTTP client with retry logic
- `src/auth.ts` - OAuth 2 authentication with token caching
- `src/logger.ts` - Logging utility
- `src/utils.ts` - Utility functions (validation, parsing, etc.)
- `src/models.ts` - TypeScript type definitions

## Testing

The test suite matches the Go implementation's test structure and validates:
- Successful requests for both actions
- Error handling
- Parameter filtering
- Authentication (API key and OAuth 2)
- Response transformation

Tests use Jest and can be run with `npm test`.

**Note**: The shared integration tests that validate both Go and TypeScript implementations are located in `shared/testdata/` and should be run separately. See the [Shared Tests README](../../shared/testdata/README.md) for details.

## Deployment

This TypeScript Lambda can be deployed alongside or instead of the Go implementation. Both implementations are functionally equivalent and validated using the shared integration tests in `shared/testdata/`.
