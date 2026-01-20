package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type HTTPClientTestSuite struct {
	suite.Suite
}

func TestHTTPClientTestSuite(t *testing.T) {
	suite.Run(t, new(HTTPClientTestSuite))
}

func (s *HTTPClientTestSuite) TestNewHTTPClient() {
	// Given: default timeout configuration
	// When: creating a new HTTP client
	client := &http.Client{Timeout: httpClientTimeout}

	// Then: client should have the configured timeout
	s.NotNil(client)
	s.Equal(httpClientTimeout, client.Timeout)
}

func (s *HTTPClientTestSuite) TestNewRetryHTTPClient() {
	// Given: a logger
	logger := NewLogger()

	// When: creating a new retry HTTP client
	client := NewRetryHTTPClient(WithLogger(logger))

	// Then: client should be created successfully
	s.NotNil(client)

	// And: client should implement HTTPClient interface
	var _ HTTPClient = client
}

func (s *HTTPClientTestSuite) TestRetryHTTPClient_Do_SuccessOnFirstAttempt() {
	// Given: a retry HTTP client and a server that returns success immediately
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	logger := NewLogger()
	client := NewRetryHTTPClient(WithLogger(logger))

	req, _ := http.NewRequest("GET", server.URL, nil)

	// When: making a request
	resp, err := client.Do(req)

	// Then: request should succeed on first attempt
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	s.Equal("success", string(body))
}

func (s *HTTPClientTestSuite) TestRetryHTTPClient_Do_RetriesOnNetworkError() {
	// Given: a retry HTTP client with maxRetries=2 and a server that fails then succeeds
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			// Simulate network error by closing connection
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	logger := NewLogger()
	client := &retryHTTPClient{
		client:     &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
		maxRetries: 2,
		baseDelay:  10 * time.Millisecond,
	}

	req, _ := http.NewRequest("GET", server.URL, nil)

	// When: making a request that fails initially
	resp, err := client.Do(req)

	// Then: request should retry and eventually succeed
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(2, attempts) // Should have retried once
}

func (s *HTTPClientTestSuite) TestRetryHTTPClient_Do_RetriesOn5xxStatus() {
	// Given: a retry HTTP client and a server that returns 500 then 200
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	logger := NewLogger()
	client := &retryHTTPClient{
		client:     &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
		maxRetries: 2,
		baseDelay:  10 * time.Millisecond,
	}

	req, _ := http.NewRequest("GET", server.URL, nil)

	// When: making a request that gets 500 error initially
	resp, err := client.Do(req)

	// Then: request should retry and eventually succeed
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(2, attempts) // Should have retried once
}

func (s *HTTPClientTestSuite) TestRetryHTTPClient_Do_DoesNotRetryOn4xxStatus() {
	// Given: a retry HTTP client and a server that returns 400
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	logger := NewLogger()
	client := NewRetryHTTPClient(WithLogger(logger))

	req, _ := http.NewRequest("GET", server.URL, nil)

	// When: making a request that gets 400 error
	resp, err := client.Do(req)

	// Then: request should not retry and return immediately
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(http.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	s.Equal("bad request", string(body))
}

func (s *HTTPClientTestSuite) TestRetryHTTPClient_Do_ExhaustsRetries() {
	// Given: a retry HTTP client with maxRetries=2 and a server that always returns 500
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	logger := NewLogger()
	client := &retryHTTPClient{
		client:     &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
		maxRetries: 2,
		baseDelay:  10 * time.Millisecond,
	}

	req, _ := http.NewRequest("GET", server.URL, nil)

	// When: making a request that always fails
	resp, err := client.Do(req)

	// Then: request should exhaust retries and return error
	s.Error(err)
	s.Nil(resp)
	s.Contains(err.Error(), "request failed after 3 attempts")
	s.Equal(3, attempts) // Initial attempt + 2 retries
}

func (s *HTTPClientTestSuite) TestRetryHTTPClient_Do_HandlesContextCancellation() {
	// Given: a retry HTTP client and a cancelled context
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := NewLogger()
	client := NewRetryHTTPClient(WithLogger(logger))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)

	// When: making a request with cancelled context
	resp, err := client.Do(req)

	// Then: request should return context cancelled error
	s.Error(err)
	s.Nil(resp)
	s.Contains(err.Error(), "context cancelled")
}

func (s *HTTPClientTestSuite) TestRetryHTTPClient_Do_PreservesRequestBody() {
	// Given: a retry HTTP client and a server that reads request body
	requestBody := "test body"
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		body, _ := io.ReadAll(r.Body)
		if attempts == 1 {
			// First attempt fails
			w.WriteHeader(http.StatusInternalServerError)
		} else if string(body) == requestBody {
			// Second attempt succeeds if body is preserved
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	logger := NewLogger()
	client := &retryHTTPClient{
		client:     &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
		maxRetries: 1,
		baseDelay:  10 * time.Millisecond,
	}

	req, _ := http.NewRequest("POST", server.URL, bytes.NewReader([]byte(requestBody)))

	// When: making a request that retries
	resp, err := client.Do(req)

	// Then: request body should be preserved across retries
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal(2, attempts)
}

func (s *HTTPClientTestSuite) TestIsRetryableError() {
	tests := []struct {
		name       string
		err        error
		statusCode int
		want       bool
	}{
		{
			name:       "Given: network error, When: checking retryability, Then: should be retryable",
			err:        io.ErrUnexpectedEOF,
			statusCode: 0,
			want:       true,
		},
		{
			name:       "Given: 500 status code, When: checking retryability, Then: should be retryable",
			err:        nil,
			statusCode: 500,
			want:       true,
		},
		{
			name:       "Given: 503 status code, When: checking retryability, Then: should be retryable",
			err:        nil,
			statusCode: 503,
			want:       true,
		},
		{
			name:       "Given: 400 status code, When: checking retryability, Then: should not be retryable",
			err:        nil,
			statusCode: 400,
			want:       false,
		},
		{
			name:       "Given: 404 status code, When: checking retryability, Then: should not be retryable",
			err:        nil,
			statusCode: 404,
			want:       false,
		},
		{
			name:       "Given: 200 status code, When: checking retryability, Then: should not be retryable",
			err:        nil,
			statusCode: 200,
			want:       false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := IsRetryableError(tt.err, tt.statusCode)
			s.Equal(tt.want, got)
		})
	}
}

func (s *HTTPClientTestSuite) TestExponentialBackoff() {
	// Given: base delay of 100ms
	baseDelay := 100 * time.Millisecond

	tests := []struct {
		name    string
		attempt int
		wantMin time.Duration
		wantMax time.Duration
	}{
		{
			name:    "Given: attempt 0, When: calculating backoff, Then: should return base delay with jitter",
			attempt: 0,
			wantMin: baseDelay,
			wantMax: baseDelay + time.Duration(float64(baseDelay)*0.25),
		},
		{
			name:    "Given: attempt 1, When: calculating backoff, Then: should return 2x delay with jitter",
			attempt: 1,
			wantMin: 2 * baseDelay,
			wantMax: 2*baseDelay + time.Duration(float64(2*baseDelay)*0.25),
		},
		{
			name:    "Given: attempt 2, When: calculating backoff, Then: should return 4x delay with jitter",
			attempt: 2,
			wantMin: 4 * baseDelay,
			wantMax: 4*baseDelay + time.Duration(float64(4*baseDelay)*0.25),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// When: calculating exponential backoff
			got := ExponentialBackoff(tt.attempt, baseDelay)

			// Then: delay should be within expected range
			s.GreaterOrEqual(got, tt.wantMin)
			s.LessOrEqual(got, tt.wantMax)
		})
	}
}
