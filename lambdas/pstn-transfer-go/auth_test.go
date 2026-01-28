package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
		authDomain     string
		clientID       string
		clientSecret   string
		mockResponse   func(w http.ResponseWriter, statusCode int)
		mockStatusCode int
		wantErr        bool
		wantToken      string
	}{
		{
			name:         "successful token fetch with authDomain",
			authDomain:   "https://auth.us-east-1-prod.cresta.ai",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(map[string]any{
					"access_token": "test-access-token",
					"token_type":   "Bearer",
					"expires_in":   3600,
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantToken:      "test-access-token",
		},
		{
			name:         "successful token fetch with different authDomain",
			authDomain:   "https://auth.us-west-2-prod.cresta.ai",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(map[string]any{
					"access_token": "test-access-token",
					"token_type":   "Bearer",
					"expires_in":   3600,
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantToken:      "test-access-token",
		},
		{
			name:         "error response from server",
			authDomain:   "https://auth.us-west-2-prod.cresta.ai",
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
			name:         "missing access_token in response",
			authDomain:   "https://auth.us-west-2-prod.cresta.ai",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(map[string]any{
					"token_type": "Bearer",
					"expires_in": 3600,
					// access_token is missing
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        true,
			wantToken:      "",
		},
		{
			name:         "empty access_token in response",
			authDomain:   "https://auth.us-west-2-prod.cresta.ai",
			clientID:     "test-client-id",
			clientSecret: "test-client-secret",
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(map[string]any{
					"access_token": "",
					"token_type":   "Bearer",
					"expires_in":   3600,
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        true,
			wantToken:      "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Clear cache before each test
			tokenCache.ClearToken(tt.clientID)

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

			fetcher := NewOAuth2TokenFetcher()
			fetcher.client = http.DefaultClient
			// Override authDomain to use test server if available
			authDomain := tt.authDomain
			if server != nil {
				authDomain = server.URL
			}

			ctx := context.Background()
			got, err := fetcher.GetToken(ctx, authDomain, tt.clientID, tt.clientSecret)

			if tt.wantErr {
				s.Error(err)
				if tt.name == "missing access_token in response" || tt.name == "empty access_token in response" {
					s.Contains(err.Error(), "missing access_token")
				}
				return
			}
			s.NoError(err)
			s.Equal(tt.wantToken, got)
		})
	}
}

func (s *AuthTestSuite) TestOAuth2TokenFetcher_GetToken_ContextCancellation() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	fetcher := NewOAuth2TokenFetcher()
	fetcher.client = http.DefaultClient

	authDomain := server.URL
	clientID := "test-client-id"
	clientSecret := "test-client-secret"

	// Clear cache before test
	tokenCache.ClearToken(clientID)

	s.Run("context cancelled before request", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := fetcher.GetToken(ctx, authDomain, clientID, clientSecret)
		s.Error(err)
		s.Contains(err.Error(), "context")
	})

	s.Run("context cancelled during request", func() {
		// Clear cache to force a new request
		tokenCache.ClearToken(clientID)

		// Create a server that delays its response
		slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Delay response to allow cancellation
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		}))
		defer slowServer.Close()

		slowFetcher := NewOAuth2TokenFetcher()
		slowFetcher.client = http.DefaultClient

		slowAuthDomain := slowServer.URL

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel context in a goroutine after a short delay
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		_, err := slowFetcher.GetToken(ctx, slowAuthDomain, clientID, clientSecret)
		s.Error(err)
		// The error should be related to context cancellation
		s.NotNil(ctx.Err(), "expected context to be cancelled")
		s.Equal(context.Canceled, ctx.Err())
	})
}

func (s *AuthTestSuite) TestOAuth2TokenFetcher_GetToken_Cache() {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "cached-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	fetcher := NewOAuth2TokenFetcher()
	fetcher.client = http.DefaultClient

	ctx := context.Background()
	authDomain := server.URL
	clientID := "client-id"
	clientSecret := "client-secret"

	// Clear cache before test
	tokenCache.ClearToken(clientID)

	s.Run("first call hits server", func() {
		token1, err := fetcher.GetToken(ctx, authDomain, clientID, clientSecret)
		s.NoError(err)
		s.Equal(1, callCount)
		s.NotEmpty(token1)
	})

	s.Run("second call uses cache", func() {
		token2, err := fetcher.GetToken(ctx, authDomain, clientID, clientSecret)
		s.NoError(err)
		s.Equal(1, callCount, "should use cache, not hit server again")
		s.NotEmpty(token2)
	})

	s.Run("different clientID hits server again", func() {
		callCount = 0
		tokenCache.ClearToken("different-client-id")
		_, err := fetcher.GetToken(ctx, authDomain, "different-client-id", clientSecret)
		s.NoError(err)
		s.Equal(1, callCount)
	})

	s.Run("same clientID uses cache regardless of authDomain", func() {
		callCount = 0
		differentAuthDomain := server.URL + "/different"
		// Same clientID, different authDomain - should use cached token
		token4, err := fetcher.GetToken(ctx, differentAuthDomain, clientID, clientSecret)
		s.NoError(err)
		s.Equal(0, callCount, "should use cache for same clientID")
		s.NotEmpty(token4)
	})

	s.Run("same clientID uses cache", func() {
		callCount = 0
		token5, err := fetcher.GetToken(ctx, authDomain, clientID, clientSecret)
		s.NoError(err)
		s.Equal(0, callCount, "should use cache")
		s.NotEmpty(token5)
	})
}
