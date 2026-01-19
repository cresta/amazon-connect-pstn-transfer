package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTPClient defines the interface for making HTTP requests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// APIClient handles HTTP requests to the Cresta API.
type APIClient struct {
	logger *Logger
	client HTTPClient
}

// NewAPIClient creates a new API client with the default HTTP client.
func NewAPIClient(logger *Logger) *APIClient {
	return &APIClient{
		logger: logger,
		client: http.DefaultClient,
	}
}

// MakeRequest makes an HTTP request with the given authentication.
func (c *APIClient) MakeRequest(ctx context.Context, method, url string, apiKey string, oauthToken string, payload any) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload: %v", err)
	}
	c.logger.Debugf("Sending request to %s with payload: %s", url, string(jsonData))

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Use OAuth 2 token if provided, otherwise fall back to API key
	if oauthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oauthToken))
	} else if apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", apiKey))
	} else {
		return nil, fmt.Errorf("either apiKey or oauthToken must be provided")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// MakeHTTPRequest is a convenience function that uses the default API client.
func MakeHTTPRequest(ctx context.Context, method, url string, apiKey string, oauthToken string, payload any) ([]byte, error) {
	logger := NewLogger()
	client := NewAPIClient(logger)
	return client.MakeRequest(ctx, method, url, apiKey, oauthToken, payload)
}
