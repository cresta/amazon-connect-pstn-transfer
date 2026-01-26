package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// OAuthCredentials represents OAuth credentials retrieved from Secrets Manager
type OAuthCredentials struct {
	OAuthClientID     string
	OAuthClientSecret string
}

// GetOAuthCredentialsFromSecretsManager fetches OAuth credentials from AWS Secrets Manager
// The secret should be a JSON object with oauthClientId and oauthClientSecret fields
func GetOAuthCredentialsFromSecretsManager(ctx context.Context, logger *Logger, secretArn string) (*OAuthCredentials, error) {
	// Extract region from secret ARN
	region, err := extractRegionFromSecretArn(secretArn)
	if err != nil {
		return nil, fmt.Errorf("failed to extract region from secret ARN: %w", err)
	}

	// Load AWS config with the region
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		logger.Errorf("Failed to load AWS config: %v", err)
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Secrets Manager client
	client := secretsmanager.NewFromConfig(cfg)

	// Get secret value
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretArn),
	}

	result, err := client.GetSecretValue(ctx, input)
	if err != nil {
		logger.Errorf("Failed to retrieve secret from Secrets Manager: %v", err)
		return nil, fmt.Errorf("failed to retrieve secret from Secrets Manager: %w", err)
	}

	if result.SecretString == nil {
		return nil, fmt.Errorf("secret value is empty or not a string")
	}

	// Parse JSON secret value
	var secretValue map[string]any
	if err := json.Unmarshal([]byte(*result.SecretString), &secretValue); err != nil {
		logger.Errorf("Failed to parse secret JSON: %v", err)
		return nil, fmt.Errorf("failed to parse secret JSON: %w", err)
	}

	// Extract oauthClientId and oauthClientSecret
	oauthClientID, ok1 := secretValue["oauthClientId"].(string)
	oauthClientSecret, ok2 := secretValue["oauthClientSecret"].(string)

	if !ok1 || !ok2 || oauthClientID == "" || oauthClientSecret == "" {
		return nil, fmt.Errorf("secret must contain oauthClientId and oauthClientSecret as non-empty strings")
	}

	logger.Debugf("Successfully retrieved OAuth credentials from Secrets Manager")

	return &OAuthCredentials{
		OAuthClientID:     oauthClientID,
		OAuthClientSecret: oauthClientSecret,
	}, nil
}

// extractRegionFromSecretArn extracts AWS region from Secrets Manager ARN
// Format: arn:aws:secretsmanager:REGION:ACCOUNT:secret:NAME
func extractRegionFromSecretArn(arn string) (string, error) {
	parts := strings.Split(arn, ":")
	if len(parts) < 4 || parts[0] != "arn" || parts[1] != "aws" || parts[2] != "secretsmanager" {
		return "", fmt.Errorf("invalid Secrets Manager ARN format: %s", arn)
	}
	return parts[3], nil
}
