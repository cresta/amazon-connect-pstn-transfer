package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/suite"
)

// readVersionFromFile reads the VERSION file from the project root
func readVersionFromFile() string {
	// Get the current test file directory
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(filename)
	// Navigate to project root: lambdas/pstn-transfer-go -> lambdas -> project root
	projectRoot := filepath.Join(testDir, "..", "..")
	versionPath := filepath.Join(projectRoot, "VERSION")
	versionBytes, err := os.ReadFile(versionPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(versionBytes))
}

type HandlersTestSuite struct {
	suite.Suite
	originalVersion string
}

func (s *HandlersTestSuite) SetupTest() {
	// Set Version to match VERSION file for tests (since ldflags aren't used in test builds)
	s.originalVersion = Version
	Version = readVersionFromFile()
	if Version == "" {
		Version = "unknown"
	}
}

func (s *HandlersTestSuite) TearDownTest() {
	// Restore original version
	Version = s.originalVersion
}

func TestHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}

func (s *HandlersTestSuite) TestGetPSTNTransferData() {
	tests := []struct {
		name             string
		authConfig       *AuthConfig
		domain           string
		virtualAgentName string
		details          *events.ConnectDetails
		mockResponse     func(w http.ResponseWriter)
		mockStatusCode   int
		wantErr          bool
	}{
		{
			name: "successful request",
			authConfig: &AuthConfig{
				APIKey: "test-api-key",
			},
			domain:           "https://api.example.com",
			virtualAgentName: "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
			details: &events.ConnectDetails{
				ContactData: events.ConnectContactData{
					ContactID: "test-contact-id",
				},
				Parameters: map[string]string{
					"customParam": "customValue",
					"apiKey":      "should-be-filtered",
					"region":      "should-be-filtered",
				},
			},
			mockResponse: func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(events.ConnectResponse{
					"phoneNumber":  "+1234567890",
					"dtmfSequence": "1234",
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "error response from server",
			authConfig: &AuthConfig{
				APIKey: "test-api-key",
			},
			domain:           "https://api.example.com",
			virtualAgentName: "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
			details: &events.ConnectDetails{
				ContactData: events.ConnectContactData{
					ContactID: "test-contact-id",
				},
				Parameters: map[string]string{},
			},
			mockResponse: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "bad request"})
			},
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/v1/" + tt.virtualAgentName + ":generatePSTNTransferData"
				s.Equal(expectedPath, r.URL.Path)

				var payload map[string]any
				err := json.NewDecoder(r.Body).Decode(&payload)
				s.NoError(err)

				// Verify payload structure
				s.Equal(tt.details.ContactData.ContactID, payload["callId"])

				ccaasMetadata, ok := payload["ccaasMetadata"].(map[string]any)
				s.True(ok, "expected ccaasMetadata in payload")

				parameters, ok := ccaasMetadata["parameters"].(map[string]any)
				s.True(ok, "expected parameters in ccaasMetadata")

				// Verify filtered keys are not present
				_, ok = parameters["apiKey"]
				s.False(ok, "apiKey should be filtered out")
				_, ok = parameters["region"]
				s.False(ok, "region should be filtered out")

				// Verify version is present in ccaasMetadata and matches VERSION file
				version, ok := ccaasMetadata["version"].(string)
				s.True(ok, "expected version in ccaasMetadata")
				s.NotEmpty(version, "version should not be empty")
				expectedVersion := readVersionFromFile()
				s.Equal(expectedVersion, version, "version should match VERSION file")

				w.WriteHeader(tt.mockStatusCode)
				tt.mockResponse(w)
			}))
			defer server.Close()

			logger := NewLogger()
			// Override the domain to use the test server
			domain := server.URL
			// Parse virtualAgentName to get components
			customer, profile, virtualAgentID, err := ParseVirtualAgentName(tt.virtualAgentName)
			s.NoError(err)
			// Create event with details
			event := events.ConnectEvent{
				Details: *tt.details,
			}

			supportedDtmfChars := "0123456789*"
			handlers := NewHandlers(logger, tt.authConfig, domain, customer, profile, virtualAgentID, supportedDtmfChars, event)
			// Override the apiClient to use the test server's http client with auth middleware
			// Create a retry client with auth, but override the underlying http.Client for testing
			testClient := NewRetryHTTPClient(WithLogger(logger), WithAuth(tt.authConfig))
			handlers.apiClient = &CrestaAPIClient{
				logger: logger,
				client: testClient,
			}

			ctx := context.Background()

			got, err := handlers.GetPSTNTransferData(ctx)

			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.NotNil(got)
		})
	}
}

// TestGetHandoffData tests the external API interface towards Amazon Connect.
// This test verifies that the response format matches the expected structure that
// Amazon Connect expects, including field names like "handoff_summary",
// "handoff_conversation", "handoff_conversationCorrelationId", and
// "handoff_transferTarget".
//
// IMPORTANT: These external field names are part of a public API contract with
// Amazon Connect. They cannot be changed without breaking existing customer
// Connect instances that may already depend on these exact field names.
func (s *HandlersTestSuite) TestGetHandoffData() {
	tests := []struct {
		name           string
		authConfig     *AuthConfig
		domain         string
		customer       string
		profile        string
		eventData      *events.ConnectContactData
		mockResponse   func(w http.ResponseWriter)
		mockStatusCode int
		wantErr        bool
		wantResponse   events.ConnectResponse
	}{
		{
			name: "successful request",
			authConfig: &AuthConfig{
				APIKey: "test-api-key",
			},
			domain:   "https://api.example.com",
			customer: "test-customer",
			profile:  "test-profile",
			eventData: &events.ConnectContactData{
				ContactID: "test-contact-id",
			},
			mockResponse: func(w http.ResponseWriter) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(FetchAIAgentHandoffResponse{
					Handoff: Handoff{
						Conversation:              "conversation-id",
						ConversationCorrelationID: "correlation-id",
						Summary:                   "test summary",
						TransferTarget:            "pstn:PSTN1",
					},
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			wantResponse: events.ConnectResponse{
				"handoff_conversation":              "conversation-id",
				"handoff_conversationCorrelationId": "correlation-id",
				"handoff_summary":                   "test summary",
				"handoff_transferTarget":            "pstn:PSTN1",
			},
		},
		{
			name: "error response from server",
			authConfig: &AuthConfig{
				APIKey: "test-api-key",
			},
			domain:   "https://api.example.com",
			customer: "test-customer",
			profile:  "test-profile",
			eventData: &events.ConnectContactData{
				ContactID: "test-contact-id",
			},
			mockResponse: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			},
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/v1/customers/" + tt.customer + "/profiles/" + tt.profile + "/handoffs:fetchAIAgentHandoff"
				s.Equal(expectedPath, r.URL.Path)

				var payload map[string]any
				err := json.NewDecoder(r.Body).Decode(&payload)
				s.NoError(err)

				s.Equal(tt.eventData.ContactID, payload["correlationId"])

				w.WriteHeader(tt.mockStatusCode)
				tt.mockResponse(w)
			}))
			defer server.Close()

			logger := NewLogger()
			domain := server.URL
			// Create event with ContactData
			event := events.ConnectEvent{
				Details: events.ConnectDetails{
					ContactData: *tt.eventData,
				},
			}
			// GetHandoffData doesn't use virtualAgentID, but we need to pass it to NewHandlers
			supportedDtmfChars := "0123456789*"
			handlers := NewHandlers(logger, tt.authConfig, domain, tt.customer, tt.profile, "", supportedDtmfChars, event)
			// Override the apiClient to use the test server's http client with auth middleware
			// Create a retry client with auth, but override the underlying http.Client for testing
			testClient := NewRetryHTTPClient(WithLogger(logger), WithAuth(tt.authConfig))
			handlers.apiClient = &CrestaAPIClient{
				logger: logger,
				client: testClient,
			}

			ctx := context.Background()

			got, err := handlers.GetHandoffData(ctx)

			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.NotNil(got)
			for k, v := range tt.wantResponse {
				s.Equal(v, (*got)[k])
			}
		})
	}
}
