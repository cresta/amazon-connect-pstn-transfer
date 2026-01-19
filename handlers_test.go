package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/suite"
)

type HandlersTestSuite struct {
	suite.Suite
}

func TestHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}

func (s *HandlersTestSuite) TestGetPSTNTransferData() {
	tests := []struct {
		name             string
		apiKey           string
		oauthToken       string
		domain           string
		virtualAgentName string
		details          *events.ConnectDetails
		mockResponse     func(w http.ResponseWriter)
		mockStatusCode   int
		wantErr          bool
	}{
		{
			name:             "successful request",
			apiKey:           "test-api-key",
			oauthToken:       "",
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
			name:             "error response from server",
			apiKey:           "test-api-key",
			oauthToken:       "",
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

				var payload map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&payload)
				s.NoError(err)

				// Verify payload structure
				s.Equal(tt.details.ContactData.ContactID, payload["callId"])

				ccaasMetadata, ok := payload["ccaasMetadata"].(map[string]interface{})
				s.True(ok, "expected ccaasMetadata in payload")

				parameters, ok := ccaasMetadata["parameters"].(map[string]interface{})
				s.True(ok, "expected parameters in ccaasMetadata")

				// Verify filtered keys are not present
				_, ok = parameters["apiKey"]
				s.False(ok, "apiKey should be filtered out")
				_, ok = parameters["region"]
				s.False(ok, "region should be filtered out")

				w.WriteHeader(tt.mockStatusCode)
				tt.mockResponse(w)
			}))
			defer server.Close()

			logger := NewLogger()
			handlers := &Handlers{
				logger: logger,
				apiClient: &APIClient{
					logger: logger,
					client: &http.Client{},
				},
			}

			// Override the domain to use the test server
			domain := server.URL
			ctx := context.Background()

			got, err := handlers.GetPSTNTransferData(ctx, tt.apiKey, tt.oauthToken, domain, tt.virtualAgentName, tt.details)

			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.NotNil(got)
		})
	}
}

func (s *HandlersTestSuite) TestGetHandoffData() {
	tests := []struct {
		name           string
		apiKey         string
		oauthToken     string
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
			name:       "successful request",
			apiKey:     "test-api-key",
			oauthToken: "",
			domain:     "https://api.example.com",
			customer:   "test-customer",
			profile:    "test-profile",
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
			name:       "error response from server",
			apiKey:     "test-api-key",
			oauthToken: "",
			domain:     "https://api.example.com",
			customer:   "test-customer",
			profile:    "test-profile",
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

				var payload map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&payload)
				s.NoError(err)

				s.Equal(tt.eventData.ContactID, payload["correlationId"])

				w.WriteHeader(tt.mockStatusCode)
				tt.mockResponse(w)
			}))
			defer server.Close()

			logger := NewLogger()
			handlers := &Handlers{
				logger: logger,
				apiClient: &APIClient{
					logger: logger,
					client: &http.Client{},
				},
			}

			domain := server.URL
			ctx := context.Background()

			got, err := handlers.GetHandoffData(ctx, tt.apiKey, tt.oauthToken, domain, tt.customer, tt.profile, tt.eventData)

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
