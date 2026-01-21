package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"
)

var (
	// httpMaxRetries is the maximum number of retries for HTTP requests.
	// Can be configured via HTTP_MAX_RETRIES environment variable (default: 3).
	httpMaxRetries = GetIntFromEnv("HTTP_MAX_RETRIES", 3)
	// httpRetryBaseDelay is the base delay for exponential backoff.
	// Can be configured via HTTP_RETRY_BASE_DELAY environment variable (default: 100ms).
	httpRetryBaseDelay = GetDurationFromEnv("HTTP_RETRY_BASE_DELAY", 100*time.Millisecond)
	// httpClientTimeout is the timeout for HTTP client requests.
	// Can be configured via HTTP_CLIENT_TIMEOUT environment variable (default: 10s).
	httpClientTimeout = GetDurationFromEnv("HTTP_CLIENT_TIMEOUT", 10*time.Second)
)

// HTTPClient defines the interface for making HTTP requests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// retryHTTPClient wraps an HTTP client with retry logic.
type retryHTTPClient struct {
	client     *http.Client
	logger     *Logger
	maxRetries int
	baseDelay  time.Duration
	authConfig *AuthConfig
}

// Do executes the HTTP request with retry logic.
func (c *retryHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Check if context is already cancelled before starting
	if req.Context().Err() != nil {
		return nil, fmt.Errorf("context cancelled: %v", req.Context().Err())
	}

	// Read request body into memory so we can recreate it for retries
	var bodyBytes []byte
	var err error
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		_ = req.Body.Close() // Ignore error
		if err != nil {
			return nil, fmt.Errorf("error reading request body: %v", err)
		}
	}

	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Check if context is cancelled before retrying
			if req.Context().Err() != nil {
				return nil, fmt.Errorf("context cancelled: %v", req.Context().Err())
			}

			delay := ExponentialBackoff(attempt-1, c.baseDelay)
			if c.logger != nil {
				c.logger.Debugf("Retrying request to %s (attempt %d/%d) after %v", req.URL.String(), attempt+1, c.maxRetries+1, delay)
			}

			select {
			case <-req.Context().Done():
				return nil, fmt.Errorf("context cancelled: %v", req.Context().Err())
			case <-time.After(delay):
				// Continue with retry
			}
		}

		// Recreate request with body for each attempt
		retryReq := req.Clone(req.Context())
		if bodyBytes != nil {
			retryReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		// Add authentication header if configured
		if c.authConfig != nil {
			authHeader, err := c.getAuthHeader(retryReq.Context())
			if err != nil {
				return nil, fmt.Errorf("error getting auth header: %v", err)
			}
			if retryReq.Header.Get("Authorization") == "" && authHeader != "" {
				retryReq.Header.Set("Authorization", authHeader)
			}
		}

		resp, err := c.client.Do(retryReq)
		if err != nil {
			lastErr = err
			if !IsRetryableError(err, 0) {
				return nil, fmt.Errorf("error making HTTP request: %v", err)
			}
			continue
		}

		// Check if status code is retryable
		if !IsRetryableError(nil, resp.StatusCode) {
			// Non-retryable, return immediately
			return resp, nil
		}

		// Retryable status code - read body and close for retry
		if resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		lastResp = resp
		lastErr = fmt.Errorf("request returned retryable status: %d", resp.StatusCode)
	}

	// All retries exhausted
	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %v", c.maxRetries+1, lastErr)
	}
	if lastResp != nil {
		return nil, fmt.Errorf("request failed after %d attempts with status: %d", c.maxRetries+1, lastResp.StatusCode)
	}
	return nil, fmt.Errorf("request failed after %d attempts", c.maxRetries+1)
}

// ClientOption is a function that configures a retryHTTPClient.
type ClientOption func(*retryHTTPClient)

// WithLogger returns a ClientOption that sets the logger for the client.
func WithLogger(logger *Logger) ClientOption {
	return func(client *retryHTTPClient) {
		client.logger = logger
	}
}

// WithAuth returns a ClientOption that adds authentication to the client.
func WithAuth(authConfig *AuthConfig) ClientOption {
	return func(client *retryHTTPClient) {
		client.authConfig = authConfig
	}
}

// NewRetryHTTPClient creates a new HTTP client with retry logic.
// Retry configuration is read from environment variables:
// - HTTP_MAX_RETRIES: maximum number of retries (default: 3)
// - HTTP_RETRY_BASE_DELAY: base delay for exponential backoff (default: 100ms)
// - HTTP_CLIENT_TIMEOUT: HTTP client timeout (default: 10s)
// Options can be provided to configure the client:
//
//	NewRetryHTTPClient(WithLogger(logger), WithAuth(authConfig))
func NewRetryHTTPClient(opts ...ClientOption) HTTPClient {
	retryClient := &retryHTTPClient{
		client:     &http.Client{Timeout: httpClientTimeout},
		logger:     NewLogger(), // Default logger
		maxRetries: httpMaxRetries,
		baseDelay:  httpRetryBaseDelay,
	}

	// Apply options in order
	for _, opt := range opts {
		opt(retryClient)
	}
	return retryClient
}

// IsRetryableError determines if an error or status code should trigger a retry.
func IsRetryableError(err error, statusCode int) bool {
	if err != nil {
		return true // Network errors are retryable
	}
	// Retry on 5xx server errors, but not on 4xx client errors
	return statusCode >= 500 && statusCode < 600
}

// ExponentialBackoff calculates the delay for the given attempt with jitter.
func ExponentialBackoff(attempt int, baseDelay time.Duration) time.Duration {
	delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
	// Add jitter: random value between 0 and 25% of delay
	jitter := time.Duration(rand.Float64() * 0.25 * float64(delay))
	return delay + jitter
}

// AuthConfig holds authentication configuration for API requests.
type AuthConfig struct {
	// APIKey is used for API key authentication (deprecated)
	APIKey string
	// OAuth credentials for OAuth 2 authentication
	Region            string
	OAuthClientID     string
	OAuthClientSecret string
	// TokenFetcher is used to fetch OAuth tokens
	TokenFetcher OAuth2TokenFetcher
}

// getAuthHeader returns the appropriate Authorization header based on AuthConfig.
// Returns an error if authConfig is nil or if authentication is not properly configured.
// Otherwise returns the Authorization header value (e.g., "Bearer <token>" for OAuth 2 or "ApiKey <key>" for API key).
func (c *retryHTTPClient) getAuthHeader(ctx context.Context) (string, error) {
	if c.authConfig == nil {
		return "", fmt.Errorf("authConfig is required")
	}

	// OAuth 2 authentication takes precedence
	if c.authConfig.OAuthClientID != "" && c.authConfig.OAuthClientSecret != "" {
		if c.authConfig.TokenFetcher == nil {
			return "", fmt.Errorf("tokenFetcher is required for OAuth authentication")
		}
		if c.authConfig.Region == "" {
			return "", fmt.Errorf("region is required for OAuth authentication")
		}

		token, err := c.authConfig.TokenFetcher.GetToken(ctx, c.authConfig.Region, c.authConfig.OAuthClientID, c.authConfig.OAuthClientSecret)
		if err != nil {
			return "", fmt.Errorf("error fetching OAuth token: %v", err)
		}
		return fmt.Sprintf("Bearer %s", token), nil
	}

	// Fall back to API key authentication (deprecated)
	if c.authConfig.APIKey != "" {
		return fmt.Sprintf("ApiKey %s", c.authConfig.APIKey), nil
	}

	return "", fmt.Errorf("no authentication configured")
}
