package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// GetFromEventParameterOrEnv retrieves a value from event parameters, environment variables, or returns a default.
func GetFromEventParameterOrEnv(event events.ConnectEvent, key, defaultValue string) string {
	if value, ok := event.Details.Parameters[key]; ok {
		return value
	}
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// CopyMap copies a map excluding specified keys and converts values to interface{}.
func CopyMap(original map[string]string, filteredKeys []string) map[string]any {
	copy := make(map[string]any)
	for k, v := range original {
		if !slices.Contains(filteredKeys, k) {
			copy[k] = v
		}
	}
	return copy
}

// ParseVirtualAgentName parses a virtual agent name into its components.
// Format: customers/{customer}/profiles/{profile}/virtualAgents/{virtualAgentID}
func ParseVirtualAgentName(virtualAgentName string) (customer string, profile string, virtualAgentID string, err error) {
	parts := strings.Split(virtualAgentName, "/")
	if len(parts) != 6 {
		return "", "", "", fmt.Errorf("invalid virtual agent name: %s", virtualAgentName)
	}
	return parts[1], parts[3], parts[5], nil
}

// BuildAPIDomainFromRegion builds an API domain URL from a region.
// Region should include the suffix (e.g., "us-west-2-prod" or "us-west-2-staging").
// e.g., "us-west-2-prod" -> "https://api.us-west-2-prod.cresta.ai"
func BuildAPIDomainFromRegion(region string) string {
	return fmt.Sprintf("https://api.%s.cresta.ai", region)
}

// ExtractRegionFromDomain extracts the AWS region (including suffix) from the API domain.
// e.g., "us-west-2-prod" from "https://api.us-west-2-prod.cresta.ai"
func ExtractRegionFromDomain(apiDomain string) (string, error) {
	re := regexp.MustCompile(`api\.([a-z0-9-]+)\.cresta\.(ai|com)`)
	matches := re.FindStringSubmatch(apiDomain)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract region from domain: %s", apiDomain)
	}
	return matches[1], nil
}
