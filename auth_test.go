package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type AuthTestSuite struct {
	suite.Suite
}

func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}

func (s *AuthTestSuite) TestOAuth2TokenFetcher_GetToken() {
	tests := []struct {
		name           string
		apiDomain      string
		region         string
		clientID       string
		clientSecret   string
		mockResponse   func(w http.ResponseWriter, statusCode int)
		mockStatusCode int
		wantErr        bool
		wantToken      string
		wantRegion     string
	}{
		{
			name:         "successful token fetch with region parameter",
			apiDomain:    "https://api.us-west-2-prod.cresta.ai",
			region:       "us-east-1-prod",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "test-access-token",
					"token_type":   "Bearer",
					"expires_in":   3600,
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantToken:      "test-access-token",
			wantRegion:     "us-east-1-prod",
		},
		{
			name:         "successful token fetch with provided region",
			apiDomain:    "https://api.us-west-2-prod.cresta.ai",
			region:       "us-west-2-prod",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "test-access-token",
					"token_type":   "Bearer",
					"expires_in":   3600,
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantToken:      "test-access-token",
			wantRegion:     "us-west-2-prod",
		},
		{
			name:         "error response from server",
			apiDomain:    "https://api.us-west-2-prod.cresta.ai",
			region:       "",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "invalid_client",
				})
			},
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
			wantToken:      "",
		},
		{
			name:         "successful token fetch with region provided",
			apiDomain:    "https://invalid-domain.com",
			region:       "us-west-2-prod",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "test-access-token",
					"token_type":   "Bearer",
					"expires_in":   3600,
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantToken:      "test-access-token",
			wantRegion:     "us-west-2-prod",
		},
		{
			name:         "invalid domain format but region provided",
			apiDomain:    "https://invalid-domain.com",
			region:       "us-west-2-prod",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token": "test-access-token",
					"token_type":   "Bearer",
					"expires_in":   3600,
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantToken:      "test-access-token",
			wantRegion:     "us-west-2-prod",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Clear cache before each test
			tokenCache.ClearToken(tt.region, tt.clientID)

			var server *httptest.Server
			if tt.mockResponse != nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					s.Equal("POST", r.Method)
					s.Equal("/v1/oauth/regionalToken", r.URL.Path)
					s.Equal("application/json", r.Header.Get("Content-Type"))
					// Verify Basic Auth is used
					username, password, ok := r.BasicAuth()
					s.True(ok, "expected Basic Auth to be used")
					s.Equal(tt.clientID, username)
					s.Equal(tt.clientSecret, password)
					tt.mockResponse(w, tt.mockStatusCode)
				}))
				defer server.Close()
			}

			var usedRegion string
			fetcher := &DefaultOAuth2TokenFetcher{
				client: http.DefaultClient,
				tokenURL: func(region string) string {
					usedRegion = region
					if server != nil {
						return server.URL + "/v1/oauth/regionalToken"
					}
					return "https://auth." + region + ".cresta.ai/v1/oauth/regionalToken"
				},
			}

			ctx := context.Background()
			got, err := fetcher.GetToken(ctx, tt.region, tt.clientID, tt.clientSecret)

			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(tt.wantToken, got)
			if tt.wantRegion != "" {
				s.Equal(tt.wantRegion, usedRegion)
			}
		})
	}
}

func (s *AuthTestSuite) TestOAuth2TokenFetcher_GetToken_Cache() {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "cached-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	fetcher := &DefaultOAuth2TokenFetcher{
		client: http.DefaultClient,
		tokenURL: func(region string) string {
			return server.URL + "/v1/oauth/regionalToken"
		},
	}

	ctx := context.Background()
	region := "us-west-2-prod"
	clientID := "client-id"
	clientSecret := "client-secret"

	// Clear cache before test
	tokenCache.ClearToken(region, clientID)

	s.Run("first call hits server", func() {
		token1, err := fetcher.GetToken(ctx, region, clientID, clientSecret)
		s.NoError(err)
		s.Equal(1, callCount)
		s.NotEmpty(token1)
	})

	s.Run("second call uses cache", func() {
		token2, err := fetcher.GetToken(ctx, region, clientID, clientSecret)
		s.NoError(err)
		s.Equal(1, callCount, "should use cache, not hit server again")
		s.NotEmpty(token2)
	})

	s.Run("different clientID hits server again", func() {
		callCount = 0
		tokenCache.ClearToken(region, "different-client-id")
		_, err := fetcher.GetToken(ctx, region, "different-client-id", clientSecret)
		s.NoError(err)
		s.Equal(1, callCount)
	})

	s.Run("different region hits server again", func() {
		callCount = 0
		tokenCache.ClearToken("us-east-1-prod", clientID)
		token4, err := fetcher.GetToken(ctx, "us-east-1-prod", clientID, clientSecret)
		s.NoError(err)
		s.Equal(1, callCount)
		s.NotEmpty(token4)
	})

	s.Run("same region and clientID uses cache", func() {
		callCount = 0
		token5, err := fetcher.GetToken(ctx, "us-east-1-prod", clientID, clientSecret)
		s.NoError(err)
		s.Equal(0, callCount, "should use cache")
		s.NotEmpty(token5)
	})
}
