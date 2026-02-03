# AWS Connect PSTN Transfer

[![CloudFormation Template](https://img.shields.io/badge/View-CloudFormation%20Template-blue?logo=amazon-aws)](https://github.com/cresta/amazon-connect-pstn-transfer/blob/main/infra/cloudformation/template.yaml)

This repo contains the required AWS resources for doing a transfer using PSTN only

- **Lambda Function** (Go, TypeScript, and Python implementations available)
- **AWS Connect Flow**

## Table Of Contents

- [AWS Connect PSTN Transfer](#aws-connect-pstn-transfer)
  - [Table Of Contents](#table-of-contents)
  - [Lambda Function Overview](#lambda-function-overview)
    - [Configuration](#configuration)
      - [Authentication](#authentication)
    - [Usage](#usage)
      - [Supported Actions](#supported-actions)
    - [Handoff Response Format](#handoff-response-format)
    - [API Specification](#api-specification)
  - [Lambda Function Implementations](#lambda-function-implementations)
    - [Global Build Scripts](#global-build-scripts)
    - [Git Hooks](#git-hooks)
    - [Running All Tests](#running-all-tests)
    - [Linting All](#linting-all)
    - [Formatting All](#formatting-all)
    - [Shared Integration Tests](#shared-integration-tests)
    - [Version Management](#version-management)
  - [Example Connect Flow](#example-connect-flow)
    - [VS Code Configuration](#vs-code-configuration)

## Lambda Function Overview

This AWS Lambda function processes Amazon Connect events and interacts with a virtual agent API to handle PSTN transfers and handoff data.
It provides two main functionalities:

- Returning a phone number and DTMF sequence
- Fetching handoff data

### Configuration

The function accepts configuration in two ways:

- **Environment variables**: Set in the Lambda function configuration (recommended for sensitive values like credentials)
- **Amazon Connect parameters**: Passed in the `Parameters` section of the Lambda invocation from your Connect flow

> **Note**: Parameters passed from Amazon Connect take precedence over environment variables. It is recommended to set `action` via Amazon Connect parameter and the rest via environment variables for better security and flexibility.

- **action**: The action to perform, either `get_pstn_transfer_data` or `get_handoff_data`. Required.
- **region**: AWS region with suffix (e.g., `voice-prod`, `us-west-2-prod` or `us-west-2-staging`). Required if `apiDomain` is not provided.
- **apiDomain**: API domain without protocol prefix (e.g., `api-customer-profile.cresta.com`). Optional. The `https://` prefix will be automatically added. If provided, the region will be extracted from the domain. Use this parameter when you need to use a customer-specific endpoint instead of the generic `api.cresta.com` domain.
- **authDomain**: OAuth authentication domain without protocol prefix (e.g., `auth.us-west-2-prod.cresta.ai`). Optional. The `https://` prefix will be automatically added, and `/v1/oauth/regionalToken` will be appended. Use this when your custom `apiDomain` requires a different auth endpoint than what would be automatically mapped.

**Validation Rules for `apiDomain` and `authDomain`:**

- **OAuth with `apiDomain`**: If you're using OAuth authentication (`oauthClientId`/`oauthClientSecret` or `oauthSecretArn`) and provide `apiDomain`, then `authDomain` **must** also be provided.
- **API key with `apiDomain`**: If you're using API key authentication (`apiKey`) and provide `apiDomain`, `authDomain` is **not required** (API key doesn't use OAuth endpoints).
- **OAuth without `apiDomain`**: If you're using OAuth authentication but don't provide `apiDomain`, `authDomain` is **not required** (the system will automatically build the auth domain from the `region` parameter).

- **virtualAgentName**: The resourcename of the virtual agent the call is transferred to. Format: `customers/{customer}/profiles/{profile}/virtualAgents/{virtualAgentID}`. Required.

#### Examples

**Example 1: Using standard region with OAuth**

Set the following environment variables in your Lambda function configuration:

- `region` = `us-west-2-prod`
- `oauthSecretArn` = `arn:aws:secretsmanager:us-west-2:123456789012:secret:my-oauth-credentials`

- **API Domain**: `https://api.us-west-2-prod.cresta.com` (built from region)
- **Auth Domain**: `https://auth.us-west-2-prod.cresta.ai` (built from region)
- **Note**: `authDomain` not required - automatically built from `region`

**Example 2: Using voice-prod region with OAuth**

Set the following environment variables in your Lambda function configuration:

- `region` = `voice-prod`
- `oauthSecretArn` = `arn:aws:secretsmanager:us-west-2:123456789012:secret:my-oauth-credentials`

- **API Domain**: `https://api.voice-prod.cresta.com` (built from region)
- **Auth Domain**: `https://auth.us-west-2-prod.cresta.ai` (mapped from voice-prod)
- **Note**: `authDomain` not required - automatically mapped from `region` (voice-prod maps to us-west-2-prod auth endpoint)

**Example 3: Using custom apiDomain with OAuth (requires authDomain)**

Set the following environment variables in your Lambda function configuration:

- `apiDomain` = `api-customer-profile.cresta.com`
- `authDomain` = `auth.us-west-2-prod.cresta.ai`
- `oauthSecretArn` = `arn:aws:secretsmanager:us-west-2:123456789012:secret:my-oauth-credentials`

- **API Domain**: `https://api-customer-profile.cresta.com`
- **Auth Domain**: `https://auth.us-west-2-prod.cresta.ai`
- **Note**: When using OAuth with `apiDomain`, `authDomain` **must** be provided

**Example 4: Using custom apiDomain with API key (authDomain not required)**

Set the following environment variables in your Lambda function configuration:

- `apiDomain` = `api-customer-profile.cresta.com`
- `apiKey` = `your-api-key`

- **API Domain**: `https://api-customer-profile.cresta.com`
- **Auth Domain**: Not used (API key authentication doesn't require OAuth endpoints)
- **Note**: When using API key with `apiDomain`, `authDomain` is **not required**

> **Note**: Any additional parameters passed from Amazon Connect (beyond the required ones listed above) will be included in the `ccaasMetadata` sent to the backend API. This allows you to pass custom metadata from your Connect flow to the backend.

#### Authentication

The function uses OAuth 2 Client Credentials Flow for authentication. You can provide credentials in one of two ways:

**Option 1: AWS Secrets Manager** ✅ **RECOMMENDED for production**

- **oauthSecretArn**: ARN of the AWS Secrets Manager secret containing OAuth credentials
  - The secret must be a JSON object with `oauthClientId` and `oauthClientSecret` fields
  - Example secret value:
    ```json
    {
      "oauthClientId": "your-client-id",
      "oauthClientSecret": "your-client-secret"
    }
    ```
  - The Lambda execution role must have `secretsmanager:GetSecretValue` permission for the secret
  - If provided, takes precedence over `oauthClientId`/`oauthClientSecret` environment variables

**Option 2: Environment Variables**

- **oauthClientId**: OAuth 2 client ID
- **oauthClientSecret**: OAuth 2 client secret

> **Note**: If both `oauthSecretArn` and `oauthClientId`/`oauthClientSecret` are provided, `oauthSecretArn` takes precedence.

### Usage

The Lambda function expects an Amazon Connect event with the following structure:

```json
{
  "Details": {
    "ContactData": {
      "ContactId": "..."
      // Other contact data from Amazon Connect
    },
    "Parameters": {
      // Recommended: Only pass action and flow-specific values as parameters
      // Prefer setting oauthClientId, oauthClientSecret, region, and virtualAgentName as environment variables
      "action": "get_pstn_transfer_data",
      // Optional: Any additional parameters will be included in ccaasMetadata
      "customParameter": "some_custom_value_that_will_be_passed_as_metadata"
    }
  }
}
```

> **Note**:
>
> - **For production deployments**, it's recommended to use AWS Secrets Manager (`oauthSecretArn`) for storing OAuth credentials instead of environment variables for better security and credential rotation support.
> - For other configuration, set `region` and `virtualAgentName` as environment variables in the Lambda function configuration.
> - Only pass `action` and flow-specific values (like `customParameter` above) as parameters from Amazon Connect.
> - Any additional parameters beyond the required ones will be included in the `ccaasMetadata` sent to the backend API.

#### Supported Actions

1. `get_pstn_transfer_data`
   - Generates PSTN transfer data for a given contact
   - Requires valid virtual agent name and contact ID

2. `get_handoff_data`
   - Fetches the latest handoff data for BOT conversations
   - Uses contact ID as correlation ID

### Handoff Response Format

All responses are flattened to a map of string key-value pairs, making them compatible with Amazon Connect's response handling. Nested JSON structures are flattened using underscore notation.

e.g.

```json
{
  "handoff_conversation": "customers/cresta/profiles/walter-dev/conversations/51ca9fc2-49ff-48f7-89ef-f3dbebf39239",
  "handoff_conversationCorrelationId": "ee4d8126-134e-4e74-8250-71c7bbf446c5",
  "handoff_summary": "Conversation is too short to generate a summary.",
  "handoff_transferTarget": "pstn:PSTN3"
}
```

### API Specification

An OpenAPI 3.0.0 specification for the used endpoints: `fetchAIAgentHandoff` and `generatePSTNTransferData` is available at [`shared/docs/api-spec.yaml`](./shared/docs/api-spec.yaml). This specification documents the request/response schemas, authentication methods, and error responses for the underlying API endpoints that this Lambda function interacts with.
Make sure to change the domain to the region-specific domain (e.g., `https://api.us-west-2-prod.cresta.com`) before trying it out.

## Lambda Function Implementations

This repository contains multiple implementations of the Lambda function, all functionally equivalent:

### Go Implementation

- **Location**: [`lambdas/pstn-transfer-go/`](./lambdas/pstn-transfer-go/)
- **README**: [Go Implementation README](./lambdas/pstn-transfer-go/README.md)

### TypeScript Implementation

- **Location**: [`lambdas/pstn-transfer-ts/`](./lambdas/pstn-transfer-ts/)
- **README**: [TypeScript Implementation README](./lambdas/pstn-transfer-ts/README.md)

### Python Implementation

- **Location**: [`lambdas/pstn-transfer-py/`](./lambdas/pstn-transfer-py/)
- **README**: [Python Implementation README](./lambdas/pstn-transfer-py/README.md)

All three implementations provide identical functionality and can be used interchangeably. Choose the implementation that best fits your team's expertise and infrastructure requirements.

For implementation-specific details, development setup, and deployment instructions, please refer to the respective README files linked above.

### Global Build Scripts

Usage:

```bash
# Build all Lambda functions
./scripts/build-all.sh
```

### Git Hooks

To install pre-commit hooks that run format and lint checks before each commit:

```bash
./scripts/install-hooks.sh
```

### Running All Tests

To run all tests (Go, TypeScript, Python, and shared integration tests) in one command:

```bash
./scripts/test-all.sh
```

### Linting All

To run all linters (Go, TypeScript, and Python) in one command:

```bash
./scripts/lint-all.sh
```

### Formatting All

To format all code (Go, TypeScript, and Python) in one command:

```bash
./scripts/format-all.sh
```

### Shared Integration Tests

- **Location**: [`shared/testdata/`](./shared/testdata/)
- **README**: [Shared Tests README](./shared/testdata/README.md)

The shared integration tests validate that all three implementations (Go, TypeScript, and Python) behave identically.
To run the shared tests:

```bash
cd shared/testdata
npm install  # First time only
npm test
```

### Version Management

The project uses a shared `VERSION` file at the project root for version management across all implementations. This version is:

- **Injected at build time** into all three implementations
- **Included in `ccaasMetadata`** sent to the backend API for logging and tracking
- **Single source of truth** - update the `VERSION` file to change the version for all implementations

The version is automatically read from the `VERSION` file during the build process:

- **Go**: Injected via `-ldflags` during compilation
- **TypeScript**: Injected via esbuild `--define` flag during bundling
- **Python**: Injected via string replacement during packaging

To update the version, simply edit the `VERSION` file at the project root.

## Example Connect Flow

The following flow is defined in [./shared/docs/VA_PSTN_Transfer.json](./shared/docs/VA_PSTN_Transfer.json)

![flow](./shared/docs/aws-connect-flow.png)

1. Call comes into Amazon Connect
2. Amazon Connect calls a lambda function to fetch DTMF sequence and phoneNumber to transfer to
   > - action: `get_pstn_transfer_data`
   > - Response validation is set to JSON
3. It stores the returned values as attributes on the Current Contact
   > ![flow](./shared/docs/aws-connect-phonenumber-dtmf.png)
4. It says the DTMF sequence (for debugging purposes)
5. Amazon Connect transfers the given phone number and enters the DTMF sequence
   > ![flow](./shared/docs/aws-connect-transfer.png)
6. Upon closure of that call, Amazon Connect continues the flow and calls the lambda function to fetch the Handoff which includes the transfer target.

   > ![flow](./shared/docs/aws-connect-action.png)
   >
   > - action: `get_handoff_data`
   > - Response validation is set to JSON

   Note: This will make all Handoff response properties (`handoff_transferTarget`, `handoff_summary`, `handoff_conversation` and `handoff_conversationCorrelationId`) available in the 'External' Namespace. Only `handoff_transferTarget` is used in this example flow.

7. The transfer target is returned as an attribute
   > `handoff_transferTarget`
   > ![flow](./shared/docs/aws-connect-target.png)
8. The target is spoken out loud (for debugging purposes)

> **When importing the flow, make sure to change the reference to the lambda function with your own**

### VS Code Configuration

The project includes VS Code configurations for optimal development:

1. **Required Extensions**:
   - **dfarley1.file-picker**: Required for the `event` task to select test event files. VS Code should prompt you to install this when opening the workspace. Otherwise you can install vsix in the `.vscode` folder.

2. **Recommended Extensions**:
   - Install the recommended Go extensions for VS Code

3. **Debugging**:
   The project includes launch configurations for debugging your Lambda function locally.

4. **Tasks**:
   Predefined tasks are available for building and testing the application. The `event` task requires the `dfarley1.file-picker` extension to select test event files.
