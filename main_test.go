package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}

type mockTokenFetcher struct {
	token string
	err   error
}

func (m *mockTokenFetcher) GetToken(ctx context.Context, region, clientID, clientSecret string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.token, nil
}

func (s *MainTestSuite) TestHandlerService_Handle() {
	tests := []struct {
		name           string
		event          events.ConnectEvent
		mockToken      string
		tokenErr       error
		mockResponse   func(w http.ResponseWriter, statusCode int)
		mockStatusCode int
		wantErr        bool
	}{
		{
			name: "successful get_pstn_transfer_data with API key",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					ContactData: events.ConnectContactData{
						ContactID: "test-contact-id",
					},
					Parameters: map[string]string{
						"action":           "get_pstn_transfer_data",
						"apiKey":           "test-api-key",
						"virtualAgentName": "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
						"customParam":      "customValue",
					},
				},
			},
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(events.ConnectResponse{
					"phoneNumber":  "+1234567890",
					"dtmfSequence": "1234",
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful get_pstn_transfer_data with OAuth 2",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					ContactData: events.ConnectContactData{
						ContactID: "test-contact-id",
					},
					Parameters: map[string]string{
						"action":            "get_pstn_transfer_data",
						"oauthClientId":     "test-client-id",
						"oauthClientSecret": "test-client-secret",
						"virtualAgentName":  "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
					},
				},
			},
			mockToken: "test-oauth-token",
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(events.ConnectResponse{
					"phoneNumber":  "+1234567890",
					"dtmfSequence": "1234",
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful get_pstn_transfer_data with OAuth 2 and region parameter",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					ContactData: events.ConnectContactData{
						ContactID: "test-contact-id",
					},
					Parameters: map[string]string{
						"action":            "get_pstn_transfer_data",
						"oauthClientId":     "test-client-id",
						"oauthClientSecret": "test-client-secret",
						"region":            "us-east-1-prod",
						"virtualAgentName":  "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
					},
				},
			},
			mockToken: "test-oauth-token",
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(events.ConnectResponse{
					"phoneNumber":  "+1234567890",
					"dtmfSequence": "1234",
				})
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful get_handoff_data",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					ContactData: events.ConnectContactData{
						ContactID: "test-contact-id",
					},
					Parameters: map[string]string{
						"action":           "get_handoff_data",
						"apiKey":           "test-api-key",
						"virtualAgentName": "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
					},
				},
			},
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
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
		},
		{
			name: "missing virtualAgentName",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					Parameters: map[string]string{
						"action": "get_pstn_transfer_data",
						"apiKey": "test-api-key",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing authentication",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					Parameters: map[string]string{
						"action":           "get_pstn_transfer_data",
						"virtualAgentName": "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid action",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					Parameters: map[string]string{
						"action":           "invalid_action",
						"apiKey":           "test-api-key",
						"virtualAgentName": "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid virtual agent name format",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					Parameters: map[string]string{
						"action":           "get_pstn_transfer_data",
						"apiKey":           "test-api-key",
						"virtualAgentName": "invalid-format",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var server *httptest.Server
			if tt.mockResponse != nil {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					tt.mockResponse(w, tt.mockStatusCode)
				}))
				defer server.Close()
			}

			logger := NewLogger()
			service := &HandlerService{
				logger: logger,
				tokenFetcher: &mockTokenFetcher{
					token: tt.mockToken,
					err:   tt.tokenErr,
				},
			}

			// Override API domain to use test server if available
			if server != nil {
				tt.event.Details.Parameters["apiDomain"] = server.URL
				// Provide region parameter when using test server to avoid extraction from domain
				if _, hasRegion := tt.event.Details.Parameters["region"]; !hasRegion {
					tt.event.Details.Parameters["region"] = "us-west-2-prod"
				}
			} else {
				tt.event.Details.Parameters["apiDomain"] = "https://api.us-west-2-prod.cresta.ai"
			}

			ctx := context.Background()
			got, err := service.Handle(ctx, tt.event)

			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.NotNil(got)
		})
	}
}

func (s *MainTestSuite) TestHandlerService_Handle_EnvironmentVariables() {
	// Set environment variables
	os.Setenv("apiKey", "env-api-key")
	os.Setenv("virtualAgentName", "customers/env-customer/profiles/env-profile/virtualAgents/env-agent")
	defer os.Unsetenv("apiKey")
	defer os.Unsetenv("virtualAgentName")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(events.ConnectResponse{
			"phoneNumber":  "+1234567890",
			"dtmfSequence": "1234",
		})
	}))
	defer server.Close()

	event := events.ConnectEvent{
		Details: events.ConnectDetails{
			ContactData: events.ConnectContactData{
				ContactID: "test-contact-id",
			},
			Parameters: map[string]string{
				"action":    "get_pstn_transfer_data",
				"apiDomain": server.URL,
				"region":    "us-west-2-prod",
			},
		},
	}

	service := NewHandlerService()
	ctx := context.Background()
	got, err := service.Handle(ctx, event)

	s.NoError(err)
	s.NotNil(got)
}
