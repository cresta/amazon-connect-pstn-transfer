package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) TestAPIClient_MakeRequest() {
	tests := []struct {
		name           string
		method         string
		url            string
		authConfig     *AuthConfig
		payload        any
		mockResponse   func(w http.ResponseWriter)
		mockStatusCode int
		wantErr        bool
		wantAuthHeader string
	}{
		{
			name:   "successful request with API key",
			method: "POST",
			url:    "/test",
			authConfig: &AuthConfig{
				APIKey: "test-api-key",
			},
			payload: map[string]string{"key": "value"},
			mockResponse: func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"result": "success"})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantAuthHeader: "ApiKey test-api-key",
		},
		{
			name:   "successful request with OAuth token",
			method: "POST",
			url:    "/test",
			authConfig: &AuthConfig{
				APIKey: "test-api-key", // For simplicity in test, we'll use API key
			},
			payload: map[string]string{"key": "value"},
			mockResponse: func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"result": "success"})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantAuthHeader: "ApiKey test-api-key",
		},
		{
			name:           "no authentication provided",
			method:         "POST",
			url:            "/test",
			authConfig:     nil,
			payload:        map[string]string{"key": "value"},
			mockStatusCode: http.StatusOK,
			wantErr:        true,
		},
		{
			name:   "error response from server",
			method: "POST",
			url:    "/test",
			authConfig: &AuthConfig{
				APIKey: "test-api-key",
			},
			payload: map[string]string{"key": "value"},
			mockResponse: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "bad request"})
			},
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
			wantAuthHeader: "ApiKey test-api-key",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s.Equal(tt.method, r.Method)
				s.Equal(tt.url, r.URL.Path)
				if tt.wantAuthHeader != "" {
					s.Equal(tt.wantAuthHeader, r.Header.Get("Authorization"))
				}
				s.Equal("application/json", r.Header.Get("Content-Type"))
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockResponse != nil {
					tt.mockResponse(w)
				}
			}))
			defer server.Close()

			logger := NewLogger()
			client, err := NewCrestaAPIClient(logger, tt.authConfig)
			if tt.wantErr && tt.authConfig == nil {
				// Expect error when authConfig is nil
				s.Error(err)
				return
			}
			s.NoError(err)
			ctx := context.Background()
			url := server.URL + tt.url

			got, err := client.MakeRequest(ctx, tt.method, url, tt.payload)

			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.NotNil(got)
		})
	}
}

func (s *ClientTestSuite) TestAPIClient_MakeRequest_JSONMarshalling() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"result": "success"})
	}))
	defer server.Close()

	logger := NewLogger()
	authConfig := &AuthConfig{
		APIKey: "test-key",
	}
	client, err := NewCrestaAPIClient(logger, authConfig)
	s.NoError(err)
	ctx := context.Background()

	// Test with complex payload
	payload := map[string]any{
		"callId": "test-call-id",
		"metadata": map[string]any{
			"key1": "value1",
			"key2": 123,
		},
	}

	got, err := client.MakeRequest(ctx, "POST", server.URL+"/test", payload)
	s.NoError(err)
	s.NotNil(got)

	var result map[string]any
	err = json.Unmarshal(got, &result)
	s.NoError(err)
	s.Equal("success", result["result"])
}
