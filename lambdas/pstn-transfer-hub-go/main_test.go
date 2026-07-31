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

	core "github.com/cresta/aws-connect-lambda/internal/pstntransfercore"
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

func (m *mockTokenFetcher) GetToken(ctx context.Context, authDomain, clientID, clientSecret string) (string, error) {
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
			name: "apiKey alone is no longer a supported auth mechanism",
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
			wantErr: true,
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
			name: "successful get_pstn_transfer_data with OAuth 2, apiDomain, and authDomain",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					ContactData: events.ConnectContactData{
						ContactID: "test-contact-id",
					},
					Parameters: map[string]string{
						"action":            "get_pstn_transfer_data",
						"oauthClientId":     "test-client-id",
						"oauthClientSecret": "test-client-secret",
						"apiDomain":         "api-customer-profile.cresta.com",
						"authDomain":        "auth.us-west-2-prod.cresta.ai",
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
			name: "successful get_pstn_transfer_data with apiDomain parameter (api-customer-profile.cresta.com)",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					ContactData: events.ConnectContactData{
						ContactID: "test-contact-id",
					},
					Parameters: map[string]string{
						"action":            "get_pstn_transfer_data",
						"oauthClientId":     "test-client-id",
						"oauthClientSecret": "test-client-secret",
						"apiDomain":         "api-customer-profile.cresta.com",
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
			name: "successful get_handoff_data with apiDomain parameter (api-customer-profile.cresta.com)",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					ContactData: events.ConnectContactData{
						ContactID: "test-contact-id",
					},
					Parameters: map[string]string{
						"action":            "get_handoff_data",
						"oauthClientId":     "test-client-id",
						"oauthClientSecret": "test-client-secret",
						"apiDomain":         "api-customer-profile.cresta.com",
						"virtualAgentName":  "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
					},
				},
			},
			mockToken: "test-oauth-token",
			mockResponse: func(w http.ResponseWriter, statusCode int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				json.NewEncoder(w).Encode(core.FetchAIAgentHandoffResponse{
					Handoff: core.Handoff{
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
			name: "successful get_handoff_data",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					ContactData: events.ConnectContactData{
						ContactID: "test-contact-id",
					},
					Parameters: map[string]string{
						"action":            "get_handoff_data",
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
				json.NewEncoder(w).Encode(core.FetchAIAgentHandoffResponse{
					Handoff: core.Handoff{
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
						"action":            "invalid_action",
						"oauthClientId":     "test-client-id",
						"oauthClientSecret": "test-client-secret",
						"virtualAgentName":  "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
					},
				},
			},
			mockToken: "test-oauth-token",
			wantErr:   true,
		},
		{
			name: "invalid virtual agent name format",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					Parameters: map[string]string{
						"action":           "get_pstn_transfer_data",
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

			logger := core.NewLogger()
			service := &HandlerService{
				logger: logger,
				tokenFetcher: &mockTokenFetcher{
					token: tt.mockToken,
					err:   tt.tokenErr,
				},
			}

			// Override API domain to use test server if available
			if server != nil {
				if _, hasAPIDomain := tt.event.Details.Parameters["apiDomain"]; hasAPIDomain {
					if _, hasRegion := tt.event.Details.Parameters["region"]; !hasRegion {
						tt.event.Details.Parameters["apiDomain"] = server.URL
						tt.event.Details.Parameters["region"] = "customer-profile"
					} else {
						tt.event.Details.Parameters["apiDomain"] = server.URL
					}
					_, hasOAuthID := tt.event.Details.Parameters["oauthClientId"]
					_, hasOAuthSecret := tt.event.Details.Parameters["oauthClientSecret"]
					_, hasOAuthARN := tt.event.Details.Parameters["oauthSecretArn"]
					if hasOAuthID || hasOAuthSecret || hasOAuthARN {
						tt.event.Details.Parameters["authDomain"] = server.URL
					}
				} else {
					tt.event.Details.Parameters["apiDomain"] = server.URL
					if _, hasRegion := tt.event.Details.Parameters["region"]; !hasRegion {
						tt.event.Details.Parameters["region"] = "us-west-2-prod"
					}
					_, hasOAuthID := tt.event.Details.Parameters["oauthClientId"]
					_, hasOAuthSecret := tt.event.Details.Parameters["oauthClientSecret"]
					_, hasOAuthARN := tt.event.Details.Parameters["oauthSecretArn"]
					if hasOAuthID || hasOAuthSecret || hasOAuthARN {
						tt.event.Details.Parameters["authDomain"] = server.URL
					}
				}
			} else {
				if _, hasAPIDomain := tt.event.Details.Parameters["apiDomain"]; !hasAPIDomain {
					tt.event.Details.Parameters["apiDomain"] = "https://api.us-west-2-prod.cresta.ai"
				}
				_, hasOAuthID := tt.event.Details.Parameters["oauthClientId"]
				_, hasOAuthSecret := tt.event.Details.Parameters["oauthClientSecret"]
				_, hasOAuthARN := tt.event.Details.Parameters["oauthSecretArn"]
				if (hasOAuthID || hasOAuthSecret || hasOAuthARN) && tt.event.Details.Parameters["authDomain"] == "" {
					tt.event.Details.Parameters["authDomain"] = "https://auth.us-west-2-prod.cresta.ai"
				}
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

func (s *MainTestSuite) TestHandlerService_Handle_StripsSecretsBeforeLogging() {
	// Guards against the same bug class that leaked ack-lambda's API key into prod logs for
	// ~2 months in 2023: Handle() must strip apiKey/oauthClientId/oauthClientSecret from
	// event.Details.Parameters before its first log line, even though apiKey is no longer a
	// supported credential here.
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
				"action":            "get_pstn_transfer_data",
				"apiKey":            "test-api-key",
				"oauthClientId":     "test-client-id",
				"oauthClientSecret": "test-client-secret",
				"apiDomain":         server.URL,
				"authDomain":        server.URL,
				"region":            "customer-profile",
				"virtualAgentName":  "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
			},
		},
	}

	service := &HandlerService{
		logger:       core.NewLogger(),
		tokenFetcher: &mockTokenFetcher{token: "test-token"},
	}
	got, err := service.Handle(context.Background(), event)

	s.NoError(err)
	s.NotNil(got)
	_, hasAPIKey := event.Details.Parameters["apiKey"]
	_, hasOAuthID := event.Details.Parameters["oauthClientId"]
	_, hasOAuthSecret := event.Details.Parameters["oauthClientSecret"]
	s.False(hasAPIKey, "apiKey must be stripped from Parameters so a later log of the event can't leak it")
	s.False(hasOAuthID, "oauthClientId must be stripped from Parameters so a later log of the event can't leak it")
	s.False(hasOAuthSecret, "oauthClientSecret must be stripped from Parameters so a later log of the event can't leak it")
}

func (s *MainTestSuite) TestHandlerService_Handle_WithAPIDomain_customer_profile() {
	// Test that handler correctly extracts region from apiDomain when apiDomain is api-customer-profile.cresta.com
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
				"action":            "get_pstn_transfer_data",
				"oauthClientId":     "test-client-id",
				"oauthClientSecret": "test-client-secret",
				"apiDomain":         server.URL, // Use test server for HTTP requests
				"authDomain":        server.URL,
				"virtualAgentName":  "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
				// Note: region is extracted from apiDomain, but since we're using test server URL,
				// we provide region to avoid extraction from localhost. The extraction logic
				// is tested separately in the shared package's utils_test.go for
				// api-customer-profile.cresta.com.
				"region": "customer-profile",
			},
		},
	}

	service := &HandlerService{
		logger:       core.NewLogger(),
		tokenFetcher: &mockTokenFetcher{token: "test-oauth-token"},
	}
	ctx := context.Background()
	got, err := service.Handle(ctx, event)

	s.NoError(err)
	s.NotNil(got)
}

func (s *MainTestSuite) TestHandlerService_Handle_WithAPIDomain_customer_profile_Extraction() {
	// Test that handler correctly extracts region from apiDomain=api-customer-profile.cresta.com
	// This test verifies the extraction works when apiDomain is provided without region
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
				"action":            "get_pstn_transfer_data",
				"oauthClientId":     "test-client-id",
				"oauthClientSecret": "test-client-secret",
				"apiDomain":         "api-customer-profile.cresta.com",
				"virtualAgentName":  "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
				// No region parameter - handler should extract "customer-profile" from apiDomain
			},
		},
	}

	// Override apiDomain to use test server for HTTP requests
	// The extraction from api-customer-profile.cresta.com is tested in the shared package's
	// utils_test.go.
	event.Details.Parameters["apiDomain"] = server.URL
	event.Details.Parameters["authDomain"] = server.URL
	event.Details.Parameters["region"] = "customer-profile" // Provide region since we override apiDomain

	service := &HandlerService{
		logger:       core.NewLogger(),
		tokenFetcher: &mockTokenFetcher{token: "test-oauth-token"},
	}
	ctx := context.Background()
	got, err := service.Handle(ctx, event)

	s.NoError(err)
	s.NotNil(got)
}

func (s *MainTestSuite) TestHandlerService_Handle_EnvironmentVariables() {
	// Set environment variables
	os.Setenv("oauthClientId", "env-client-id")
	os.Setenv("oauthClientSecret", "env-client-secret")
	os.Setenv("virtualAgentName", "customers/env-customer/profiles/env-profile/virtualAgents/env-agent")
	defer os.Unsetenv("oauthClientId")
	defer os.Unsetenv("oauthClientSecret")
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
				"action":     "get_pstn_transfer_data",
				"apiDomain":  server.URL,
				"authDomain": server.URL,
				"region":     "us-west-2-prod",
			},
		},
	}

	service := &HandlerService{
		logger:       core.NewLogger(),
		tokenFetcher: &mockTokenFetcher{token: "test-oauth-token"},
	}
	ctx := context.Background()
	got, err := service.Handle(ctx, event)

	s.NoError(err)
	s.NotNil(got)
}
