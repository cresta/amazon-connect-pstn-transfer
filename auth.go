package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// TokenCache stores OAuth 2 access tokens with expiration, keyed by region and clientID.
type TokenCache struct {
	cache map[string]cacheEntry
	mu    sync.RWMutex
}

type cacheEntry struct {
	token     string
	expiresAt time.Time // Token Expiration Time + 5 minute buffer
}

var tokenCache = &TokenCache{
	cache: make(map[string]cacheEntry),
}

// cacheKey generates a cache key from region and clientID.
func cacheKey(region, clientID string) string {
	return fmt.Sprintf("pstn-transfer:tokencache:%s:%s", region, clientID)
}

// GetCachedToken returns a valid cached token if available, otherwise returns empty string.
func (tc *TokenCache) GetCachedToken(region, clientID string) string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	key := cacheKey(region, clientID)
	entry, ok := tc.cache[key]
	if ok && entry.token != "" && time.Now().Before(entry.expiresAt) {
		return entry.token
	}
	return ""
}

// SetToken caches a token with expiration time.
func (tc *TokenCache) SetToken(region, clientID, token string, expiresIn time.Duration) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	key := cacheKey(region, clientID)
	tc.cache[key] = cacheEntry{
		token:     token,
		expiresAt: time.Now().Add(expiresIn - 5*time.Minute), // Subtract 5 minute buffer for safety
	}
}

// ClearToken clears the cached token for a specific region and clientID (useful for testing).
func (tc *TokenCache) ClearToken(region, clientID string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	key := cacheKey(region, clientID)
	delete(tc.cache, key)
}

// OAuth2TokenFetcher defines the interface for fetching OAuth 2 tokens.
type OAuth2TokenFetcher interface {
	GetToken(ctx context.Context, region, clientID, clientSecret string) (string, error)
}

// DefaultOAuth2TokenFetcher implements OAuth2TokenFetcher using HTTP client.
type DefaultOAuth2TokenFetcher struct {
	client   HTTPClient
	tokenURL func(region string) string
}

// NewOAuth2TokenFetcher creates a new OAuth2TokenFetcher with default configuration.
// Region should include the suffix (e.g., "us-west-2-prod" or "us-west-2-staging").
// Uses a retry-enabled HTTP client.
func NewOAuth2TokenFetcher() *DefaultOAuth2TokenFetcher {
	logger := NewLogger()
	return &DefaultOAuth2TokenFetcher{
		client: NewRetryHTTPClient(WithLogger(logger)),
		tokenURL: func(region string) string {
			return fmt.Sprintf("https://auth.%s.cresta.ai/v1/oauth/regionalToken", region)
		},
	}
}

// GetToken fetches an OAuth 2 access token using client credentials flow.
func (f *DefaultOAuth2TokenFetcher) GetToken(ctx context.Context, region, clientID, clientSecret string) (string, error) {
	// Check cache first
	if cachedToken := tokenCache.GetCachedToken(region, clientID); cachedToken != "" {
		return cachedToken, nil
	}

	// Construct token endpoint URL using the same region
	tokenURL := f.tokenURL(region)

	// Prepare JSON payload
	payload := map[string]string{
		"grant_type": "client_credentials",
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling payload: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating token request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading token response: %v", err)
	}

	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", fmt.Errorf("error parsing token response: %v", err)
	}

	if tokenResponse.AccessToken == "" {
		return "", fmt.Errorf("missing access_token in token response")
	}

	// Cache the token
	if tokenResponse.ExpiresIn > 0 {
		tokenCache.SetToken(region, clientID, tokenResponse.AccessToken, time.Duration(tokenResponse.ExpiresIn)*time.Second)
	}

	return tokenResponse.AccessToken, nil
}
