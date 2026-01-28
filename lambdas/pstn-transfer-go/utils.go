package main

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

var (
	apiDomainRegex        = regexp.MustCompile(`api[-.]([a-z0-9-]+)\.cresta\.(ai|com)`)
	virtualAgentNameRegex = regexp.MustCompile(`^customers/([^/]+)/profiles/([^/]+)/virtualAgents/([^/]+)$`)

	regionToAuthRegion = map[string]string{
		"chat-prod":  "us-west-2-prod", // chat-prod uses us-west-2-prod auth endpoint
		"voice-prod": "us-west-2-prod", // voice-prod uses us-west-2-prod auth endpoint
	}
)

func GetFromEventParameterOrEnv(event events.ConnectEvent, key, defaultValue string) string {
	if value, ok := event.Details.Parameters[key]; ok {
		return value
	}
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func CopyMap(original map[string]string, filteredKeys map[string]bool) map[string]any {
	result := make(map[string]any)
	for k, v := range original {
		if !filteredKeys[k] {
			result[k] = v
		}
	}
	return result
}

// ParseVirtualAgentName parses a virtual agent name.
// Format: customers/{customer}/profiles/{profile}/virtualAgents/{virtualAgentID}
func ParseVirtualAgentName(virtualAgentName string) (customer string, profile string, virtualAgentID string, err error) {
	matches := virtualAgentNameRegex.FindStringSubmatch(virtualAgentName)
	if len(matches) != 4 {
		return "", "", "", fmt.Errorf("invalid virtual agent name: %s. Expected format: customers/{customer}/profiles/{profile}/virtualAgents/{virtualAgentID}", virtualAgentName)
	}
	return matches[1], matches[2], matches[3], nil
}

// BuildAPIDomainFromRegion builds an API domain URL from a region.
// e.g., "us-west-2-prod" -> "https://api.us-west-2-prod.cresta.com"
// e.g., "us-west-2-staging" -> "https://api.us-west-2-staging.cresta.ai"
func BuildAPIDomainFromRegion(region string) string {
	if strings.HasSuffix(region, "-prod") {
		return fmt.Sprintf("https://api.%s.cresta.com", region)
	}
	return fmt.Sprintf("https://api.%s.cresta.ai", region)
}

// ExtractRegionFromDomain extracts the AWS region from the API domain.
// e.g., "us-west-2-prod" from "https://api.us-west-2-prod.cresta.ai"
func ExtractRegionFromDomain(apiDomain string) (string, error) {
	matches := apiDomainRegex.FindStringSubmatch(apiDomain)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract region from domain: %s", apiDomain)
	}
	return matches[1], nil
}

// GetAuthRegion maps a region to its corresponding auth region.
// If no mapping exists, the region is returned as-is.
func GetAuthRegion(region string) string {
	if authRegion, ok := regionToAuthRegion[region]; ok {
		return authRegion
	}
	return region
}

// GetIntFromEnv retrieves an integer from environment variable or returns default.
func GetIntFromEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetDurationFromEnv retrieves a duration from environment variable or returns default.
// Accepts duration strings like "100ms", "2s", "1m", etc.
func GetDurationFromEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// ValidateDomain validates that a domain is a safe URL for API requests.
// Returns an error if the domain contains path traversal, fragments, or other unsafe components.
// Allows HTTP only for localhost/127.0.0.1 (for testing), otherwise requires HTTPS.
func ValidateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	parsedURL, err := url.Parse(domain)
	if err != nil {
		return fmt.Errorf("invalid domain URL: %v", err)
	}

	// Require HTTPS scheme for security, except for localhost (testing)
	isLocalhost := parsedURL.Hostname() == "localhost" || parsedURL.Hostname() == "127.0.0.1" || strings.HasPrefix(parsedURL.Hostname(), "127.")
	if parsedURL.Scheme != "https" && !(parsedURL.Scheme == "http" && isLocalhost) {
		return fmt.Errorf("domain must use HTTPS scheme, got: %s", parsedURL.Scheme)
	}

	// Reject domains with path, query, or fragment components
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		return fmt.Errorf("domain cannot contain path components: %s", parsedURL.Path)
	}
	if parsedURL.RawQuery != "" {
		return fmt.Errorf("domain cannot contain query parameters: %s", parsedURL.RawQuery)
	}
	if parsedURL.Fragment != "" {
		return fmt.Errorf("domain cannot contain fragment: %s", parsedURL.Fragment)
	}

	// Ensure host is present
	if parsedURL.Host == "" {
		return fmt.Errorf("domain must have a host")
	}

	// Check for path traversal attempts in host
	if strings.Contains(parsedURL.Host, "/") || strings.Contains(parsedURL.Host, "..") {
		return fmt.Errorf("domain host contains invalid characters")
	}

	return nil
}

// ValidatePathSegment validates that a path segment is safe (no path traversal or special characters).
func ValidatePathSegment(segment, name string) error {
	if segment == "" {
		return fmt.Errorf("%s cannot be empty", name)
	}

	// Reject path traversal attempts
	if strings.Contains(segment, "..") || strings.Contains(segment, "/") {
		return fmt.Errorf("%s contains invalid characters (path traversal detected): %s", name, segment)
	}

	// Reject URL-encoded path traversal
	if strings.Contains(segment, "%2e%2e") || strings.Contains(segment, "%2E%2E") {
		return fmt.Errorf("%s contains URL-encoded path traversal: %s", name, segment)
	}

	// Reject null bytes
	if strings.Contains(segment, "\x00") {
		return fmt.Errorf("%s contains null byte", name)
	}

	return nil
}
