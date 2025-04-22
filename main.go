package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// FetchAIAgentHandoff API response.
type FetchAIAgentHandoffResponse struct {
	Handoff Handoff `json:"handoff"`
}

type Handoff struct {
	Conversation              string `json:"conversation"`
	ConversationCorrelationID string `json:"conversationCorrelationId"`
	Summary                   string `json:"summary"`
	TransferTarget            string `json:"transferTarget"`
	// NOTE: We don't need the metadataByTaxonomy field.
}

func makeHTTPRequest(ctx context.Context, method, url string, apiKey string, payload any) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload: %v", err)
	}
	fmt.Printf("Sending request to %s with payload: %s\n", url, string(jsonData))

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func getPSTNTransferData(ctx context.Context, apiKey, domain, virtualAgentName string, eventData *events.ConnectContactData) (*events.ConnectResponse, error) {
	url := fmt.Sprintf("%s/v1/%s:generatePSTNTransferData", domain, virtualAgentName)
	payload := map[string]any{
		"callId":             eventData.ContactID,
		"ccaasMetadata":      eventData,
		"supportedDtmfChars": "0123456789*",
	}

	fmt.Printf("Making request to %s with payload: %+v\n", url, payload)

	body, err := makeHTTPRequest(ctx, "POST", url, apiKey, payload)
	if err != nil {
		return nil, err
	}

	var result *events.ConnectResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parsing response JSON: %v", err)
	}

	fmt.Printf("Received response: %+v\n", result)
	return result, nil
}

func parseVirtualAgentName(virtualAgentName string) (customer string, profile string, virtualAgentID string, err error) {
	// virtualAgentFormat is the format of the virtual agent ID
	// customers/{customer}/profiles/{profile}/virtualAgents/{virtualAgentID}
	parts := strings.Split(virtualAgentName, "/")
	if len(parts) != 6 {
		return "", "", "", fmt.Errorf("invalid virtual agent name: %s", virtualAgentName)
	}
	return parts[1], parts[3], parts[5], nil
}

func getHandoffData(ctx context.Context, apiKey, domain, customer, profile string, eventData *events.ConnectContactData) (*events.ConnectResponse, error) {
	url := fmt.Sprintf("%s/v1/customers/%s/profiles/%s/handoffs:fetchAIAgentHandoff", domain, customer, profile)
	payload := map[string]any{
		"correlationId": eventData.ContactID,
	}

	fmt.Printf("Making request to %s with payload: %+v\n", url, payload)

	body, err := makeHTTPRequest(ctx, "POST", url, apiKey, payload)
	if err != nil {
		return nil, err
	}

	var result *events.FetchAIAgentHandoffResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error marshalling response body: %v", err)
	}
	fmt.Printf("Received response: %+v\n", result)

	return &events.ConnectResponse{
		"handoff_conversation":              result.Handoff.Conversation,
		"handoff_conversationCorrelationId": result.Handoff.ConversationCorrelationID,
		"handoff_summary":                   result.Handoff.Summary,
		"handoff_transferTarget":            result.Handoff.TransferTarget,
	}, nil
}

func handler(ctx context.Context, event events.ConnectEvent) (events.ConnectResponse, error) {
	fmt.Printf("Received event: %+v\n", event)

	var result *events.ConnectResponse
	var err error

	domain := getFromEventParameterOrEnv(event, "apiDomain", "https://api.us-west-2-prod.cresta.com")
	action := getFromEventParameterOrEnv(event, "action", "")
	apiKey := getFromEventParameterOrEnv(event, "apiKey", "")
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	virtualAgentName := getFromEventParameterOrEnv(event, "virtualAgentName", "")
	if virtualAgentName == "" {
		return nil, fmt.Errorf("virtualAgentName is required")
	}

	fmt.Printf("Domain: %s, Action: %s, Virtual Agent Name: %s\n", domain, action, virtualAgentName)
	customer, profile, _, err := parseVirtualAgentName(virtualAgentName)
	if err != nil {
		fmt.Printf("Error parsing virtual agent name: %v\n", err)
		return nil, err
	}

	switch action {
	case "get_pstn_transfer_data":
		result, err = getPSTNTransferData(ctx, apiKey, domain, virtualAgentName, &event.Details.ContactData)
	case "get_handoff_data":
		result, err = getHandoffData(ctx, apiKey, domain, customer, profile, &event.Details.ContactData)
	default:
		return nil, fmt.Errorf("invalid action: %s", action)
	}

	if err != nil {
		return nil, err
	}

	return *result, nil
}

func getFromEventParameterOrEnv(event events.ConnectEvent, key, defaultValue string) string {
	if value, ok := event.Details.Parameters[key]; ok {
		return value
	}
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	lambda.Start(handler)
}
