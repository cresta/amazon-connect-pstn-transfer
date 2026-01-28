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

// TokenCache stores OAuth 2 access tokens with expiration, keyed by clientID.
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

// cacheKey generates a cache key from clientID.
func cacheKey(clientID string) string {
	return fmt.Sprintf("pstn-transfer:tokencache:%s", clientID)
}

// GetCachedToken returns a valid cached token if available, otherwise returns empty string.
func (tc *TokenCache) GetCachedToken(clientID string) string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	key := cacheKey(clientID)
	entry, ok := tc.cache[key]
	if ok && entry.token != "" && time.Now().Before(entry.expiresAt) {
		return entry.token
	}
	return ""
}

// SetToken caches a token with expiration time.
func (tc *TokenCache) SetToken(clientID, token string, expiresIn time.Duration) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	key := cacheKey(clientID)
	// Apply safety buffer but ensure at least some positive cache time
	safetyBuffer := 5 * time.Minute
	if expiresIn <= safetyBuffer {
		safetyBuffer = expiresIn / 2
	}
	tc.cache[key] = cacheEntry{
		token:     token,
		expiresAt: time.Now().Add(expiresIn - safetyBuffer),
	}
}

// ClearToken clears the cached token for a specific clientID (useful for testing).
func (tc *TokenCache) ClearToken(clientID string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	key := cacheKey(clientID)
	delete(tc.cache, key)
}

// OAuth2TokenFetcher defines the interface for fetching OAuth 2 tokens.
type OAuth2TokenFetcher interface {
	GetToken(ctx context.Context, authDomain, clientID, clientSecret string) (string, error)
}

// DefaultOAuth2TokenFetcher implements OAuth2TokenFetcher using HTTP client.
type DefaultOAuth2TokenFetcher struct {
	logger *Logger
	client HTTPClient
}

// NewOAuth2TokenFetcher creates a new OAuth2TokenFetcher with default configuration.
// Uses a retry-enabled HTTP client.
func NewOAuth2TokenFetcher() *DefaultOAuth2TokenFetcher {
	logger := NewLogger()
	return &DefaultOAuth2TokenFetcher{
		logger: logger,
		client: NewRetryHTTPClient(WithLogger(logger)),
	}
}

// GetToken fetches an OAuth 2 access token using client credentials flow.
func (f *DefaultOAuth2TokenFetcher) GetToken(ctx context.Context, authDomain, clientID, clientSecret string) (string, error) {
	// Check cache first (use clientID as cache key)
	if cachedToken := tokenCache.GetCachedToken(clientID); cachedToken != "" {
		return cachedToken, nil
	}

	// Build token URL from authDomain (domain only, append path)
	tokenURL := authDomain + "/v1/oauth/regionalToken"

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

	// Cache the token (use clientID as cache key)
	if tokenResponse.ExpiresIn > 0 {
		tokenCache.SetToken(clientID, tokenResponse.AccessToken, time.Duration(tokenResponse.ExpiresIn)*time.Second)
	} else {
		if f.logger != nil {
			f.logger.Errorf("token response has invalid expires_in (value: %d), token will not be cached", tokenResponse.ExpiresIn)
		}
		// Return error since we cannot cache the token and it may expire immediately
		return "", fmt.Errorf("invalid token response: expires_in is %d (must be > 0)", tokenResponse.ExpiresIn)
	}

	return tokenResponse.AccessToken, nil
}
