package main

import (
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/suite"
)

type UtilsTestSuite struct {
	suite.Suite
}

func TestUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(UtilsTestSuite))
}

func (s *UtilsTestSuite) TestGetFromEventParameterOrEnv() {
	tests := []struct {
		name         string
		event        events.ConnectEvent
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name: "value from event parameters",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					Parameters: map[string]string{
						"testKey": "event-value",
					},
				},
			},
			key:          "testKey",
			defaultValue: "default-value",
			want:         "event-value",
		},
		{
			name: "value from environment variable",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					Parameters: map[string]string{},
				},
			},
			key:          "testKey",
			defaultValue: "default-value",
			envValue:     "env-value",
			want:         "env-value",
		},
		{
			name: "default value when not in event or env",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					Parameters: map[string]string{},
				},
			},
			key:          "testKey",
			defaultValue: "default-value",
			want:         "default-value",
		},
		{
			name: "event parameter takes precedence over env",
			event: events.ConnectEvent{
				Details: events.ConnectDetails{
					Parameters: map[string]string{
						"testKey": "event-value",
					},
				},
			},
			key:          "testKey",
			defaultValue: "default-value",
			envValue:     "env-value",
			want:         "event-value",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Set environment variable if needed
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := GetFromEventParameterOrEnv(tt.event, tt.key, tt.defaultValue)
			s.Equal(tt.want, got)
		})
	}
}

func (s *UtilsTestSuite) TestCopyMap() {
	tests := []struct {
		name         string
		original     map[string]string
		filteredKeys []string
		want         map[string]interface{}
	}{
		{
			name:         "filter single key",
			original:     map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
			filteredKeys: []string{"key2"},
			want:         map[string]interface{}{"key1": "value1", "key3": "value3"},
		},
		{
			name:         "filter multiple keys",
			original:     map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
			filteredKeys: []string{"key1", "key3"},
			want:         map[string]interface{}{"key2": "value2"},
		},
		{
			name:         "filter all keys",
			original:     map[string]string{"key1": "value1", "key2": "value2"},
			filteredKeys: []string{"key1", "key2"},
			want:         map[string]interface{}{},
		},
		{
			name:         "filter non-existent keys",
			original:     map[string]string{"key1": "value1", "key2": "value2"},
			filteredKeys: []string{"key3", "key4"},
			want:         map[string]interface{}{"key1": "value1", "key2": "value2"},
		},
		{
			name:         "empty map",
			original:     map[string]string{},
			filteredKeys: []string{"key1"},
			want:         map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := CopyMap(tt.original, tt.filteredKeys)
			s.Equal(len(tt.want), len(got))
			for k, v := range tt.want {
				s.Equal(v, got[k])
			}
		})
	}
}

func (s *UtilsTestSuite) TestParseVirtualAgentName() {
	tests := []struct {
		name             string
		virtualAgentName string
		wantCustomer     string
		wantProfile      string
		wantVirtualAgentID string
		wantErr          bool
	}{
		{
			name:             "valid virtual agent name",
			virtualAgentName: "customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
			wantCustomer:     "test-customer",
			wantProfile:      "test-profile",
			wantVirtualAgentID: "test-agent",
			wantErr:          false,
		},
		{
			name:             "invalid format - too few parts",
			virtualAgentName: "customers/test-customer/profiles/test-profile",
			wantErr:          true,
		},
		{
			name:             "invalid format - too many parts",
			virtualAgentName: "customers/test-customer/profiles/test-profile/virtualAgents/test-agent/extra",
			wantErr:          true,
		},
		{
			name:             "empty string",
			virtualAgentName: "",
			wantErr:          true,
		},
		{
			name:             "invalid format - wrong structure",
			virtualAgentName: "invalid-format",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			gotCustomer, gotProfile, gotVirtualAgentID, err := ParseVirtualAgentName(tt.virtualAgentName)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(tt.wantCustomer, gotCustomer)
			s.Equal(tt.wantProfile, gotProfile)
			s.Equal(tt.wantVirtualAgentID, gotVirtualAgentID)
		})
	}
}

func (s *UtilsTestSuite) TestBuildAPIDomainFromRegion() {
	tests := []struct {
		name     string
		region   string
		want     string
	}{
		{
			name:   "region with -prod suffix",
			region: "us-west-2-prod",
			want:   "https://api.us-west-2-prod.cresta.ai",
		},
		{
			name:   "region with -staging suffix",
			region: "us-east-1-staging",
			want:   "https://api.us-east-1-staging.cresta.ai",
		},
		{
			name:   "region with custom suffix",
			region: "eu-west-1-dev",
			want:   "https://api.eu-west-1-dev.cresta.ai",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := BuildAPIDomainFromRegion(tt.region)
			s.Equal(tt.want, got)
		})
	}
}

func (s *UtilsTestSuite) TestExtractRegionFromDomain() {
	tests := []struct {
		name    string
		domain  string
		want    string
		wantErr bool
	}{
		{
			name:    "valid domain with -prod suffix",
			domain:  "https://api.us-west-2-prod.cresta.ai",
			want:    "us-west-2-prod",
			wantErr: false,
		},
		{
			name:    "valid domain with -staging suffix",
			domain:  "https://api.us-east-1-staging.cresta.ai",
			want:    "us-east-1-staging",
			wantErr: false,
		},
		{
			name:    "valid domain without protocol",
			domain:  "api.eu-west-1-prod.cresta.ai",
			want:    "eu-west-1-prod",
			wantErr: false,
		},
		{
			name:    "invalid domain format",
			domain:  "https://invalid-domain.com",
			want:    "",
			wantErr: true,
		},
		{
			name:    "domain without cresta.ai/cresta.com",
			domain:  "https://api.us-west-2-prod.example.com",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			domain:  "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := ExtractRegionFromDomain(tt.domain)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(tt.want, got)
		})
	}
}
