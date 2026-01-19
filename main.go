package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// HandlerService contains dependencies for the Lambda handler.
type HandlerService struct {
	handlers     *Handlers
	tokenFetcher OAuth2TokenFetcher
}

// NewHandlerService creates a new HandlerService with default dependencies.
func NewHandlerService() *HandlerService {
	return &HandlerService{
		handlers:     NewHandlers(),
		tokenFetcher: NewOAuth2TokenFetcher(),
	}
}

func handler(ctx context.Context, event events.ConnectEvent) (events.ConnectResponse, error) {
	return NewHandlerService().Handle(ctx, event)
}

// Handle processes the Lambda event and returns a response.
func (s *HandlerService) Handle(ctx context.Context, event events.ConnectEvent) (events.ConnectResponse, error) {
	fmt.Printf("Received event: %+v\n", event)

	var result *events.ConnectResponse
	var err error

	// Extract region first - from region parameter or apiDomain (deprecated)
	regionParam := GetFromEventParameterOrEnv(event, "region", "")
	apiDomainParam := GetFromEventParameterOrEnv(event, "apiDomain", "") // Deprecated: use region instead

	var region string
	if regionParam != "" {
		region = regionParam
	} else if apiDomainParam != "" {
		// Try to extract region from apiDomain, but don't fail if it doesn't match the pattern
		extractedRegion, err := ExtractRegionFromDomain(apiDomainParam)
		if err != nil {
			return nil, fmt.Errorf("could not extract region from apiDomain: %v", err)
		}
		region = extractedRegion
	}

	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	// Calculate apiDomain from region if not provided, otherwise use provided apiDomain
	var domain string
	if apiDomainParam != "" {
		domain = apiDomainParam
	} else {
		domain = BuildAPIDomainFromRegion(region)
	}

	action := GetFromEventParameterOrEnv(event, "action", "")
	apiKey := GetFromEventParameterOrEnv(event, "apiKey", "") // Deprecated: use oauthClientId/oauthClientSecret instead
	oauthClientID := GetFromEventParameterOrEnv(event, "oauthClientId", "")
	oauthClientSecret := GetFromEventParameterOrEnv(event, "oauthClientSecret", "")

	// Either API key (deprecated) or OAuth 2 credentials must be provided
	var oauthToken string
	if oauthClientID != "" && oauthClientSecret != "" {
		// Use OAuth 2 authentication - pass region directly
		oauthToken, err = s.tokenFetcher.GetToken(ctx, region, oauthClientID, oauthClientSecret)
		if err != nil {
			return nil, fmt.Errorf("error getting OAuth 2 token: %v", err)
		}
		fmt.Printf("Using OAuth 2 authentication\n")
	} else if apiKey != "" {
		// Use API key authentication (deprecated)
		fmt.Printf("Using API key authentication (deprecated)\n")
	} else {
		return nil, fmt.Errorf("either apiKey (deprecated) or oauthClientId/oauthClientSecret must be provided")
	}

	virtualAgentName := GetFromEventParameterOrEnv(event, "virtualAgentName", "")
	if virtualAgentName == "" {
		return nil, fmt.Errorf("virtualAgentName is required")
	}

	fmt.Printf("Domain: %s, Region: %s, Action: %s, Virtual Agent Name: %s\n", domain, region, action, virtualAgentName)
	customer, profile, _, err := ParseVirtualAgentName(virtualAgentName)
	if err != nil {
		fmt.Printf("Error parsing virtual agent name: %v\n", err)
		return nil, err
	}

	switch action {
	case "get_pstn_transfer_data":
		result, err = s.handlers.GetPSTNTransferData(ctx, apiKey, oauthToken, domain, virtualAgentName, &event.Details)
	case "get_handoff_data":
		result, err = s.handlers.GetHandoffData(ctx, apiKey, oauthToken, domain, customer, profile, &event.Details.ContactData)
	default:
		return nil, fmt.Errorf("invalid action: %s", action)
	}

	if err != nil {
		return nil, err
	}

	return *result, nil
}

func main() {
	lambda.Start(handler)
}
