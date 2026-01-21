package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CrestaAPIClient handles HTTP requests to the Cresta API.
type CrestaAPIClient struct {
	logger *Logger
	client HTTPClient
}

func NewCrestaAPIClient(logger *Logger, authConfig *AuthConfig) (*CrestaAPIClient, error) {
	if authConfig == nil {
		// For CrestaAPIClient, auth is required
		return nil, fmt.Errorf("authConfig is required for CrestaAPIClient")
	}
	client := NewRetryHTTPClient(WithLogger(logger), WithAuth(authConfig))
	return &CrestaAPIClient{
		logger: logger,
		client: client,
	}, nil
}

func (c *CrestaAPIClient) MakeRequest(ctx context.Context, method, url string, payload any) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload: %v", err)
	}
	c.logger.Debugf("Sending request to %s with payload: %s", url, string(jsonData))

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}
