package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
)

// Handlers contains handler functions for different API actions.
type Handlers struct {
	logger    *Logger
	apiClient *APIClient
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(logger *Logger) *Handlers {
	return &Handlers{
		logger:    logger,
		apiClient: NewAPIClient(logger),
	}
}

// GetPSTNTransferData retrieves PSTN transfer data for a given contact.
func (h *Handlers) GetPSTNTransferData(ctx context.Context, apiKey, oauthToken, domain, virtualAgentName string, details *events.ConnectDetails) (*events.ConnectResponse, error) {
	url := fmt.Sprintf("%s/v1/%s:generatePSTNTransferData", domain, virtualAgentName)

	// Filter out apiDomain, region, action, apiKey, oauthClientId, oauthClientSecret, and virtualAgentName from parameters
	filteredKeys := []string{"apiDomain", "region", "action", "apiKey", "oauthClientId", "oauthClientSecret", "virtualAgentName"}
	filteredParameters := CopyMap(details.Parameters, filteredKeys)

	eventDataJSON, err := json.Marshal(details.ContactData)
	if err != nil {
		return nil, fmt.Errorf("error marshalling ContactData: %v", err)
	}
	var eventDataMap map[string]any
	if err := json.Unmarshal(eventDataJSON, &eventDataMap); err != nil {
		return nil, fmt.Errorf("error unmarshalling ContactData: %v", err)
	}

	// Merge ContactData with parameters as a sub-field of ccaasMetadata
	ccaasMetadata := make(map[string]any)
	for k, v := range eventDataMap {
		ccaasMetadata[k] = v
	}
	ccaasMetadata["parameters"] = filteredParameters

	payload := map[string]any{
		"callId":             details.ContactData.ContactID,
		"ccaasMetadata":      ccaasMetadata,
		"supportedDtmfChars": "0123456789*",
	}

	h.logger.Debugf("Making request to %s with payload: %+v", url, payload)

	body, err := h.apiClient.MakeRequest(ctx, "POST", url, apiKey, oauthToken, payload)
	if err != nil {
		return nil, err
	}

	var result *events.ConnectResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing response JSON: %v", err)
	}

	h.logger.Debugf("Received response: %+v", result)
	return result, nil
}

// GetHandoffData retrieves handoff data for a given contact.
func (h *Handlers) GetHandoffData(ctx context.Context, apiKey, oauthToken, domain, customer, profile string, eventData *events.ConnectContactData) (*events.ConnectResponse, error) {
	url := fmt.Sprintf("%s/v1/customers/%s/profiles/%s/handoffs:fetchAIAgentHandoff", domain, customer, profile)
	payload := map[string]any{
		"correlationId": eventData.ContactID,
	}

	h.logger.Debugf("Making request to %s with payload: %+v", url, payload)

	body, err := h.apiClient.MakeRequest(ctx, "POST", url, apiKey, oauthToken, payload)
	if err != nil {
		return nil, err
	}

	var result *FetchAIAgentHandoffResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshalling response body: %v", err)
	}
	h.logger.Debugf("Received response: %+v", result)

	return &events.ConnectResponse{
		"handoff_conversation":              result.Handoff.Conversation,
		"handoff_conversationCorrelationId": result.Handoff.ConversationCorrelationID,
		"handoff_summary":                   result.Handoff.Summary,
		"handoff_transferTarget":            result.Handoff.TransferTarget,
	}, nil
}
