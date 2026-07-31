package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	core "github.com/cresta/aws-connect-lambda/internal/pstntransfercore"
)

// Version is set at build time via ldflags
var Version string

// HandlerService contains dependencies for the Lambda handler.
type HandlerService struct {
	logger       *core.Logger
	tokenFetcher core.OAuth2TokenFetcher
}

// NewHandlerService creates a new HandlerService with default dependencies.
func NewHandlerService() *HandlerService {
	logger := core.NewLogger()
	return &HandlerService{
		logger:       logger,
		tokenFetcher: core.NewOAuth2TokenFetcher(),
	}
}

func handler(ctx context.Context, event events.ConnectEvent) (events.ConnectResponse, error) {
	return NewHandlerService().Handle(ctx, event)
}

// Handle processes the Lambda event and returns a response.
//
// Unlike pstn-transfer-go, this Lambda does not accept the deprecated apiKey credential —
// see README.md for why. It still defensively strips a stray apiKey parameter (if a flow
// happens to pass one) before logging, alongside the OAuth credentials, so nothing sensitive
// reaches the debug log either way.
func (s *HandlerService) Handle(ctx context.Context, event events.ConnectEvent) (events.ConnectResponse, error) {
	// Strip secret-bearing parameters before any logging of the event, mirroring ack-lambda's
	// GetAndRemoveAPIKey()-before-first-log pattern — fixes the bug class that leaked ack-lambda's
	// API key into prod logs for ~2 months in 2023. Values are captured here and reused below;
	// they are deliberately not re-read from event.Details.Parameters later in this function.
	_ = core.GetAndRemoveFromEventParameterOrEnv(event, "apiKey", "") // not a supported credential here; scrubbed defensively only
	oauthClientID := core.GetAndRemoveFromEventParameterOrEnv(event, "oauthClientId", "")
	oauthClientSecret := core.GetAndRemoveFromEventParameterOrEnv(event, "oauthClientSecret", "")

	s.logger.Debugf("Received event: %+v", event)

	var result *events.ConnectResponse
	var err error

	// Extract region first - from region parameter or apiDomain
	regionParam := core.GetFromEventParameterOrEnv(event, "region", "")
	apiDomainParam := core.GetFromEventParameterOrEnv(event, "apiDomain", "")
	authDomainParam := core.GetFromEventParameterOrEnv(event, "authDomain", "")

	oauthSecretArn := core.GetFromEventParameterOrEnv(event, "oauthSecretArn", "")

	// Validate that apiDomain and authDomain are used together
	willUseOAuth := oauthSecretArn != "" || (oauthClientID != "" && oauthClientSecret != "")
	if willUseOAuth {
		if (apiDomainParam != "" && authDomainParam == "") || (apiDomainParam == "" && authDomainParam != "") {
			return nil, fmt.Errorf("apiDomain and authDomain must be provided together")
		}
	}

	var region string
	if regionParam != "" {
		region = regionParam
	} else if apiDomainParam != "" {
		// Try to extract region from apiDomain, but don't fail if it doesn't match the pattern
		extractedRegion, err := core.ExtractRegionFromDomain(apiDomainParam)
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
		// Add https:// prefix if not present
		if !strings.HasPrefix(apiDomainParam, "http://") && !strings.HasPrefix(apiDomainParam, "https://") {
			domain = "https://" + apiDomainParam
		} else {
			domain = apiDomainParam
		}
	} else {
		domain = core.BuildAPIDomainFromRegion(region)
	}

	// Validate domain to prevent injection attacks
	if err := core.ValidateDomain(domain); err != nil {
		return nil, fmt.Errorf("invalid domain: %v", err)
	}

	// Process authDomain if provided
	var authDomain string
	if authDomainParam != "" {
		// Add https:// prefix if not present
		if !strings.HasPrefix(authDomainParam, "http://") && !strings.HasPrefix(authDomainParam, "https://") {
			authDomain = "https://" + authDomainParam
		} else {
			authDomain = authDomainParam
		}
		// Validate authDomain to prevent injection attacks
		if err := core.ValidateDomain(authDomain); err != nil {
			return nil, fmt.Errorf("invalid authDomain: %v", err)
		}
	}

	action := core.GetFromEventParameterOrEnv(event, "action", "")
	if action == "" {
		return nil, fmt.Errorf("action is required")
	}

	virtualAgentName := core.GetFromEventParameterOrEnv(event, "virtualAgentName", "")
	if virtualAgentName == "" {
		return nil, fmt.Errorf("virtualAgentName is required")
	}

	customer, profile, virtualAgentID, err := core.ParseVirtualAgentName(virtualAgentName)
	if err != nil {
		s.logger.Errorf("Error parsing virtual agent name: %v", err)
		return nil, err
	}

	// Validate path segments to prevent injection attacks
	if err := core.ValidatePathSegment(customer, "customer"); err != nil {
		return nil, err
	}
	if err := core.ValidatePathSegment(profile, "profile"); err != nil {
		return nil, err
	}
	if err := core.ValidatePathSegment(virtualAgentID, "virtualAgentID"); err != nil {
		return nil, err
	}

	// OAuth 2 credentials must be provided. Priority: Secrets Manager > Environment/Parameters.
	var authConfig *core.AuthConfig
	resolvedOAuthClientID := oauthClientID
	resolvedOAuthClientSecret := oauthClientSecret

	// Try to fetch from Secrets Manager if ARN is provided
	if oauthSecretArn != "" {
		s.logger.Infof("Fetching OAuth credentials from Secrets Manager: %s", oauthSecretArn)
		credentials, err := core.GetOAuthCredentialsFromSecretsManager(ctx, s.logger, oauthSecretArn)
		if err != nil {
			s.logger.Errorf("Failed to retrieve credentials from Secrets Manager: %v", err)
			return nil, fmt.Errorf("failed to retrieve OAuth credentials from Secrets Manager: %w", err)
		}
		resolvedOAuthClientID = credentials.OAuthClientID
		resolvedOAuthClientSecret = credentials.OAuthClientSecret
		s.logger.Infof("Successfully retrieved OAuth credentials from Secrets Manager")
	}

	if resolvedOAuthClientID == "" || resolvedOAuthClientSecret == "" {
		return nil, fmt.Errorf("either oauthClientId/oauthClientSecret or oauthSecretArn must be provided")
	}

	// Determine auth domain: use provided authDomain, or build from region
	var finalAuthDomain string
	if authDomain != "" {
		finalAuthDomain = authDomain
	} else {
		authRegion := core.GetAuthRegion(region)
		finalAuthDomain = fmt.Sprintf("https://auth.%s.cresta.ai", authRegion)
	}
	s.logger.Infof("Using OAuth 2 authentication")
	authConfig = &core.AuthConfig{
		AuthDomain:        finalAuthDomain,
		OAuthClientID:     resolvedOAuthClientID,
		OAuthClientSecret: resolvedOAuthClientSecret,
		TokenFetcher:      s.tokenFetcher,
	}

	// Get supportedDtmfChars from environment variable only, default to "0123456789*"
	supportedDtmfChars := os.Getenv("supportedDtmfChars")
	if supportedDtmfChars == "" {
		supportedDtmfChars = "0123456789*"
	}

	// Create handlers with authConfig, domain, parsed components, and event
	handlers := core.NewHandlers(s.logger, authConfig, domain, customer, profile, virtualAgentID, supportedDtmfChars, Version, event)

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
