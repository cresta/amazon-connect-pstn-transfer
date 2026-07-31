// Package pstntransfercore holds the PSTN Transfer Lambda logic shared between
// lambdas/pstn-transfer-go (deployed per-customer, in the customer's own AWS account) and
// lambdas/pstn-transfer-hub-go (deployed once by Cresta, invoked cross-account). Each
// lambda's own main.go is the only per-deployment code: it resolves auth/region from
// Parameters or env vars and calls into this package, which is otherwise identical for both.
//
//   - handlers.go - business logic for the two actions (get_pstn_transfer_data, get_handoff_data)
//   - client.go / httpclient.go - HTTP client with retry logic and auth injection
//   - auth.go - OAuth 2 client-credentials flow with token caching
//   - secretsmanager.go - fetches OAuth credentials from AWS Secrets Manager
//   - utils.go - validation/parsing helpers (region, domain, virtualAgentName, event Parameters)
//   - logger.go - logging utility
//   - models.go - API response type definitions
//
// AuthConfig (in httpclient.go) still supports the deprecated API-key path via its APIKey
// field, since pstn-transfer-go's main.go still needs it. pstn-transfer-hub-go's main.go
// simply never populates that field, so the path is unreachable there without needing a
// second copy of this package.
package pstntransfercore
