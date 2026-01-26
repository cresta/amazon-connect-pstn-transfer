package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// Helper function to check if error message contains any of the given substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

type SecretsManagerTestSuite struct {
	suite.Suite
	logger *Logger
}

func TestSecretsManagerTestSuite(t *testing.T) {
	suite.Run(t, new(SecretsManagerTestSuite))
}

func (s *SecretsManagerTestSuite) SetupTest() {
	s.logger = NewLogger()
}

func (s *SecretsManagerTestSuite) TestExtractRegionFromSecretArn() {
	tests := []struct {
		name    string
		arn     string
		want    string
		wantErr bool
	}{
		{
			name:    "valid ARN with us-west-2",
			arn:     "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret",
			want:    "us-west-2",
			wantErr: false,
		},
		{
			name:    "valid ARN with eu-west-1",
			arn:     "arn:aws:secretsmanager:eu-west-1:123456789012:secret:test-secret",
			want:    "eu-west-1",
			wantErr: false,
		},
		{
			name:    "invalid ARN - wrong service",
			arn:     "arn:aws:s3:us-west-2:123456789012:bucket:test-bucket",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid ARN - not an ARN",
			arn:     "not-an-arn",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid ARN - missing parts",
			arn:     "arn:aws:secretsmanager",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid ARN - wrong partition",
			arn:     "arn:aws-cn:secretsmanager:us-west-2:123456789012:secret:test-secret",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			region, err := extractRegionFromSecretArn(tt.arn)
			if tt.wantErr {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(tt.want, region)
			}
		})
	}
}

func (s *SecretsManagerTestSuite) TestGetOAuthCredentialsFromSecretsManager_InvalidARN() {
	secretArn := "invalid-arn"

	_, err := GetOAuthCredentialsFromSecretsManager(context.Background(), s.logger, secretArn)
	s.Error(err)
	s.Contains(err.Error(), "failed to extract region from secret ARN")
	s.Contains(err.Error(), "invalid Secrets Manager ARN format")
}

func (s *SecretsManagerTestSuite) TestGetOAuthCredentialsFromSecretsManager_ValidARNButNoCredentials() {
	secretArn := "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret"

	// This will fail at AWS config loading stage since we don't have AWS credentials in test environment
	// But we can verify it gets past ARN parsing
	_, err := GetOAuthCredentialsFromSecretsManager(context.Background(), s.logger, secretArn)
	// Should fail at config loading or Secrets Manager call, not ARN parsing
	s.Error(err)
	s.NotContains(err.Error(), "invalid Secrets Manager ARN format")
	// Should contain error about AWS config or Secrets Manager access
	s.True(
		containsAny(err.Error(), []string{
			"failed to load AWS config",
			"Failed to load AWS config",
			"failed to retrieve secret",
			"Failed to retrieve secret",
			"no such host",
			"credentials",
		}),
		"Error should be about AWS config or Secrets Manager access, got: %s", err.Error(),
	)
}

// Test helper function to validate secret JSON structure
func (s *SecretsManagerTestSuite) TestValidateSecretJSON() {
	tests := []struct {
		name        string
		secretJSON  string
		wantErr     bool
		description string
	}{
		{
			name:        "valid secret JSON",
			secretJSON:  `{"oauthClientId":"test-id","oauthClientSecret":"test-secret"}`,
			wantErr:     false,
			description: "should parse valid JSON with both fields",
		},
		{
			name:        "missing oauthClientId",
			secretJSON:  `{"oauthClientSecret":"test-secret"}`,
			wantErr:     true,
			description: "should fail when oauthClientId is missing",
		},
		{
			name:        "missing oauthClientSecret",
			secretJSON:  `{"oauthClientId":"test-id"}`,
			wantErr:     true,
			description: "should fail when oauthClientSecret is missing",
		},
		{
			name:        "empty oauthClientId",
			secretJSON:  `{"oauthClientId":"","oauthClientSecret":"test-secret"}`,
			wantErr:     true,
			description: "should fail when oauthClientId is empty",
		},
		{
			name:        "empty oauthClientSecret",
			secretJSON:  `{"oauthClientId":"test-id","oauthClientSecret":""}`,
			wantErr:     true,
			description: "should fail when oauthClientSecret is empty",
		},
		{
			name:        "invalid JSON",
			secretJSON:  `invalid json{`,
			wantErr:     true,
			description: "should fail on invalid JSON",
		},
		{
			name:        "non-string oauthClientId",
			secretJSON:  `{"oauthClientId":123,"oauthClientSecret":"test-secret"}`,
			wantErr:     true,
			description: "should fail when oauthClientId is not a string",
		},
		{
			name:        "non-string oauthClientSecret",
			secretJSON:  `{"oauthClientId":"test-id","oauthClientSecret":456}`,
			wantErr:     true,
			description: "should fail when oauthClientSecret is not a string",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var secretValue map[string]any
			err := json.Unmarshal([]byte(tt.secretJSON), &secretValue)
			if err != nil {
				if !tt.wantErr {
					s.Failf("Unexpected JSON parse error", "error: %v", err)
				}
				return
			}

			oauthClientID, ok1 := secretValue["oauthClientId"].(string)
			oauthClientSecret, ok2 := secretValue["oauthClientSecret"].(string)

			if !ok1 || !ok2 || oauthClientID == "" || oauthClientSecret == "" {
				if !tt.wantErr {
					s.Failf("Validation should have passed", tt.description)
				}
			} else {
				if tt.wantErr {
					s.Failf("Validation should have failed", tt.description)
				}
			}
		})
	}
}
