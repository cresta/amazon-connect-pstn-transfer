package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
)

// FilteredKeys is a map of keys to filter out from parameters
var FilteredKeys = map[string]bool{
	"apiDomain":          true,
	"region":             true,
	"action":             true,
	"apiKey":             true,
	"oauthClientId":      true,
	"oauthClientSecret":  true,
	"virtualAgentName":   true,
	"supportedDtmfChars": true,
}

// Handlers contains handler functions for different API actions.
type Handlers struct {
	logger             *Logger
	apiClient          *CrestaAPIClient
	domain             string
	customerID         string
	profileID          string
	virtualAgentID     string
	supportedDtmfChars string
	event              events.ConnectEvent
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(logger *Logger, authConfig *AuthConfig, domain, customerID, profileID, virtualAgentID, supportedDtmfChars string, event events.ConnectEvent) *Handlers {
	apiClient, err := NewCrestaAPIClient(logger, authConfig)
	if err != nil {
		// This should never happen as authConfig is validated before calling NewHandlers
		panic(fmt.Sprintf("failed to create APIClient: %v", err))
	}
	return &Handlers{
		logger:             logger,
		apiClient:          apiClient,
		domain:             domain,
		customerID:         customerID,
		profileID:          profileID,
		virtualAgentID:     virtualAgentID,
		supportedDtmfChars: supportedDtmfChars,
		event:              event,
	}
}

// GetPSTNTransferData retrieves PSTN transfer data for a given contact.
func (h *Handlers) GetPSTNTransferData(ctx context.Context) (*events.ConnectResponse, error) {
	virtualAgentName := fmt.Sprintf("customers/%s/profiles/%s/virtualAgents/%s", h.customerID, h.profileID, h.virtualAgentID)
	url := fmt.Sprintf("%s/v1/%s:generatePSTNTransferData", h.domain, virtualAgentName)

	filteredParameters := CopyMap(h.event.Details.Parameters, FilteredKeys)

	eventDataJSON, err := json.Marshal(h.event.Details.ContactData)
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
		"callId":             h.event.Details.ContactData.ContactID,
		"ccaasMetadata":      ccaasMetadata,
		"supportedDtmfChars": h.supportedDtmfChars,
	}

	h.logger.Debugf("Making request to %s with payload: %+v", url, payload)

	body, err := h.apiClient.MakeRequest(ctx, "POST", url, payload)
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
func (h *Handlers) GetHandoffData(ctx context.Context) (*events.ConnectResponse, error) {
	url := fmt.Sprintf("%s/v1/customers/%s/profiles/%s/handoffs:fetchAIAgentHandoff", h.domain, h.customerID, h.profileID)
	payload := map[string]any{
		"correlationId": h.event.Details.ContactData.ContactID,
	}

	h.logger.Debugf("Making request to %s with payload: %+v", url, payload)

	body, err := h.apiClient.MakeRequest(ctx, "POST", url, payload)
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
