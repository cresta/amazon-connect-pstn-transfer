package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// HandlerService contains dependencies for the Lambda handler.
type HandlerService struct {
	logger       *Logger
	tokenFetcher OAuth2TokenFetcher
}

// NewHandlerService creates a new HandlerService with default dependencies.
func NewHandlerService() *HandlerService {
	logger := NewLogger()
	return &HandlerService{
		logger:       logger,
		tokenFetcher: NewOAuth2TokenFetcher(),
	}
}

func handler(ctx context.Context, event events.ConnectEvent) (events.ConnectResponse, error) {
	return NewHandlerService().Handle(ctx, event)
}

// Handle processes the Lambda event and returns a response.
func (s *HandlerService) Handle(ctx context.Context, event events.ConnectEvent) (events.ConnectResponse, error) {
	s.logger.Debugf("Received event: %+v", event)

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

	// Validate domain to prevent injection attacks
	if err := ValidateDomain(domain); err != nil {
		return nil, fmt.Errorf("invalid domain: %v", err)
	}

	action := GetFromEventParameterOrEnv(event, "action", "")
	if action == "" {
		return nil, fmt.Errorf("action is required")
	}

	apiKey := GetFromEventParameterOrEnv(event, "apiKey", "") // Deprecated: use oauthClientId/oauthClientSecret instead
	oauthClientID := GetFromEventParameterOrEnv(event, "oauthClientId", "")
	oauthClientSecret := GetFromEventParameterOrEnv(event, "oauthClientSecret", "")

	virtualAgentName := GetFromEventParameterOrEnv(event, "virtualAgentName", "")
	if virtualAgentName == "" {
		return nil, fmt.Errorf("virtualAgentName is required")
	}

	customer, profile, virtualAgentID, err := ParseVirtualAgentName(virtualAgentName)
	if err != nil {
		s.logger.Errorf("Error parsing virtual agent name: %v", err)
		return nil, err
	}

	// Validate path segments to prevent injection attacks
	if err := ValidatePathSegment(customer, "customer"); err != nil {
		return nil, err
	}
	if err := ValidatePathSegment(profile, "profile"); err != nil {
		return nil, err
	}
	if err := ValidatePathSegment(virtualAgentID, "virtualAgentID"); err != nil {
		return nil, err
	}

	// Either API key (deprecated) or OAuth 2 credentials must be provided
	var authConfig *AuthConfig
	if oauthClientID != "" && oauthClientSecret != "" {
		// Use OAuth 2 authentication
		s.logger.Infof("Using OAuth 2 authentication")
		authConfig = &AuthConfig{
			Region:            region,
			OAuthClientID:     oauthClientID,
			OAuthClientSecret: oauthClientSecret,
			TokenFetcher:      s.tokenFetcher,
		}
	} else if apiKey != "" {
		// Use API key authentication (deprecated)
		s.logger.Warnf("Using API key authentication (deprecated)")
		authConfig = &AuthConfig{
			APIKey: apiKey,
		}
	} else {
		return nil, fmt.Errorf("either apiKey (deprecated) or oauthClientId/oauthClientSecret must be provided")
	}

	// Get supportedDtmfChars from environment variable only, default to "0123456789*"
	supportedDtmfChars := os.Getenv("supportedDtmfChars")
	if supportedDtmfChars == "" {
		supportedDtmfChars = "0123456789*"
	}

	// Create handlers with authConfig, domain, parsed components, and event
	handlers := NewHandlers(s.logger, authConfig, domain, customer, profile, virtualAgentID, supportedDtmfChars, event)

	s.logger.Infof("Domain: %s, Region: %s, Action: %s, Virtual Agent Name: %s", domain, region, action, virtualAgentName)

	switch action {
	case "get_pstn_transfer_data":
		result, err = handlers.GetPSTNTransferData(ctx)
	case "get_handoff_data":
		result, err = handlers.GetHandoffData(ctx)
	default:
		return nil, fmt.Errorf("invalid action: %s", action)
	}

	if err != nil {
		return nil, err
	}

	return *result, nil
}

func main() {
	// Support test mode: if --test flag is passed, read from stdin and write to stdout
	if len(os.Args) > 1 && os.Args[1] == "--test" {
		var event events.ConnectEvent
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(&event); err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding event: %v\n", err)
			os.Exit(1)
		}

		result, err := handler(context.Background(), event)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		encoder := json.NewEncoder(os.Stdout)
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding response: %v\n", err)
			os.Exit(1)
		}
	} else {
		lambda.Start(handler)
	}
}
