# PSTN Transfer Hub Lambda

The Cresta-hosted, multi-tenant variant of the PSTN Transfer Lambda.

## Why this is a separate directory from `pstn-transfer-go`

`../pstn-transfer-go` is deployed once per customer, into that customer's own AWS account, via CloudFormation or manual setup — see its own README.
This directory holds a different deployment: one Lambda, deployed once by Cresta into a Cresta-owned account, invoked cross-account by every customer's Amazon Connect flow.
The request/response behavior is shared with `pstn-transfer-go` via the [`internal/pstntransfercore`](../../internal/pstntransfercore) package, but the deployment model is different enough that the entry points are kept separate, so each one's assumptions about *who* is calling and *what* credentials they present stay independently editable.

## How this differs from `pstn-transfer-go`

This directory is only `main.go` (plus its test) — everything else (HTTP client, OAuth, Secrets Manager, business-logic handlers, validation/parsing utilities, types) is imported from `internal/pstntransfercore`, shared verbatim with `pstn-transfer-go`. The only behavioral difference is in this file:

- **No deprecated `apiKey` auth path.** `pstn-transfer-go`'s `main.go` accepts a raw `apiKey` Parameter as a fallback, forwarded verbatim as an `Authorization: ApiKey ...` header. Because this Lambda's deployment is fully Cresta-controlled, it never needs to support that mechanism — only OAuth 2, normally via Secrets Manager (`oauthSecretArn`). This `main.go` never populates `AuthConfig.APIKey`, so that path in the shared package is simply never exercised here. A stray `apiKey` Parameter is still scrubbed from the event before logging, in case a flow config still sets one — it's just never read as a credential.

Everything else — `region`/`apiDomain` resolution, `virtualAgentName` parsing, OAuth token caching, retry logic, the two actions (`get_pstn_transfer_data`, `get_handoff_data`) — comes from the shared package and is identical to `pstn-transfer-go`.

## Development

Same as `pstn-transfer-go`:

```bash
go build ./...
go test ./...
```

## Deployment

Not yet wired up. This Lambda is meant to be deployed once, by Cresta, via Terraform (IAM role + resource policy) and Flux/ACK (the `Function` resource) — the same pattern as `cresta-api-ack-lambda-function` (`go-servers/voice-integration/amazon-connect-lambda`) — rather than the CloudFormation/manual-console flow `pstn-transfer-go`'s README documents for customer self-deploys. That infra is a separate, later piece of work.
