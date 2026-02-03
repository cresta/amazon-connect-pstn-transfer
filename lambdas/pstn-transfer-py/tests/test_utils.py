"""
Tests for utility functions matching Go and TypeScript test structure
"""

import pytest

from src.types import ConnectEvent
from src.utils import (
    build_api_domain_from_region,
    copy_map,
    extract_region_from_domain,
    get_auth_region,
    get_duration_from_env,
    get_from_event_parameter_or_env,
    get_int_from_env,
    parse_virtual_agent_name,
    validate_domain,
    validate_path_segment,
)


class TestGetFromEventParameterOrEnv:
    """Tests for get_from_event_parameter_or_env"""

    def test_returns_event_parameter(self):
        """Should return value from event parameters"""
        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test"},
                "Parameters": {"key": "event-value"},
            }
        )
        result = get_from_event_parameter_or_env(event, "key", "default")
        assert result == "event-value"

    def test_returns_env_var_when_no_parameter(self, monkeypatch):
        """Should return environment variable when not in parameters"""
        monkeypatch.setenv("key", "env-value")
        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test"},
                "Parameters": {},
            }
        )
        result = get_from_event_parameter_or_env(event, "key", "default")
        assert result == "env-value"

    def test_returns_default_when_neither_present(self, monkeypatch):
        """Should return default when key not in parameters or env"""
        monkeypatch.delenv("key", raising=False)
        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test"},
                "Parameters": {},
            }
        )
        result = get_from_event_parameter_or_env(event, "key", "default")
        assert result == "default"

    def test_event_parameter_takes_precedence(self, monkeypatch):
        """Event parameter should take precedence over environment variable"""
        monkeypatch.setenv("key", "env-value")
        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test"},
                "Parameters": {"key": "event-value"},
            }
        )
        result = get_from_event_parameter_or_env(event, "key", "default")
        assert result == "event-value"


class TestCopyMap:
    """Tests for copy_map"""

    def test_copies_unfiltered_keys(self):
        """Should copy keys that are not filtered"""
        original = {"keep1": "value1", "keep2": "value2", "filter": "filtered"}
        filtered_keys = {"filter"}
        result = copy_map(original, filtered_keys)
        assert result == {"keep1": "value1", "keep2": "value2"}

    def test_returns_empty_dict_when_all_filtered(self):
        """Should return empty dict when all keys are filtered"""
        original = {"a": "1", "b": "2"}
        filtered_keys = {"a", "b"}
        result = copy_map(original, filtered_keys)
        assert result == {}


class TestParseVirtualAgentName:
    """Tests for parse_virtual_agent_name"""

    def test_parses_valid_name(self):
        """Should parse valid virtual agent name"""
        customer, profile, agent_id = parse_virtual_agent_name(
            "customers/test-customer/profiles/test-profile/virtualAgents/agent-123"
        )
        assert customer == "test-customer"
        assert profile == "test-profile"
        assert agent_id == "agent-123"

    def test_raises_on_invalid_format(self):
        """Should raise error on invalid format"""
        with pytest.raises(ValueError, match="invalid virtual agent name"):
            parse_virtual_agent_name("invalid-format")

    def test_raises_on_missing_parts(self):
        """Should raise error when parts are missing"""
        with pytest.raises(ValueError, match="invalid virtual agent name"):
            parse_virtual_agent_name("customers/test-customer/profiles")


class TestBuildAPIDomainFromRegion:
    """Tests for build_api_domain_from_region"""

    def test_builds_prod_domain(self):
        """Should build .cresta.com domain for prod regions"""
        result = build_api_domain_from_region("us-west-2-prod")
        assert result == "https://api.us-west-2-prod.cresta.com"

    def test_builds_staging_domain(self):
        """Should build .cresta.ai domain for non-prod regions"""
        result = build_api_domain_from_region("us-west-2-staging")
        assert result == "https://api.us-west-2-staging.cresta.ai"

    def test_raises_on_invalid_region(self):
        """Should raise error on invalid region characters"""
        with pytest.raises(ValueError, match="invalid region"):
            build_api_domain_from_region("invalid_region!")


class TestExtractRegionFromDomain:
    """Tests for extract_region_from_domain"""

    def test_extracts_region_from_api_domain(self):
        """Should extract region from API domain"""
        result = extract_region_from_domain("https://api.us-west-2-prod.cresta.com")
        assert result == "us-west-2-prod"

    def test_extracts_region_with_hyphen(self):
        """Should extract region with hyphen separator"""
        result = extract_region_from_domain("https://api-us-west-2-prod.cresta.ai")
        assert result == "us-west-2-prod"

    def test_raises_on_non_matching_domain(self):
        """Should raise error on non-matching domain"""
        with pytest.raises(ValueError, match="could not extract region"):
            extract_region_from_domain("https://example.com")


class TestGetAuthRegion:
    """Tests for get_auth_region"""

    def test_maps_voice_prod(self):
        """Should map voice-prod to us-west-2-prod"""
        result = get_auth_region("voice-prod")
        assert result == "us-west-2-prod"

    def test_maps_chat_prod(self):
        """Should map chat-prod to us-west-2-prod"""
        result = get_auth_region("chat-prod")
        assert result == "us-west-2-prod"

    def test_returns_region_as_is(self):
        """Should return unmapped region as-is"""
        result = get_auth_region("eu-west-1-prod")
        assert result == "eu-west-1-prod"


class TestGetIntFromEnv:
    """Tests for get_int_from_env"""

    def test_returns_env_value(self, monkeypatch):
        """Should return integer from environment"""
        monkeypatch.setenv("TEST_INT", "42")
        result = get_int_from_env("TEST_INT", 0)
        assert result == 42

    def test_returns_default_on_missing(self, monkeypatch):
        """Should return default when env var missing"""
        monkeypatch.delenv("TEST_INT", raising=False)
        result = get_int_from_env("TEST_INT", 99)
        assert result == 99

    def test_returns_default_on_invalid(self, monkeypatch):
        """Should return default on invalid integer"""
        monkeypatch.setenv("TEST_INT", "not-a-number")
        result = get_int_from_env("TEST_INT", 99)
        assert result == 99


class TestGetDurationFromEnv:
    """Tests for get_duration_from_env"""

    def test_parses_milliseconds(self, monkeypatch):
        """Should parse milliseconds duration"""
        monkeypatch.setenv("TEST_DURATION", "100ms")
        result = get_duration_from_env("TEST_DURATION", 0)
        assert result == 100

    def test_parses_seconds(self, monkeypatch):
        """Should parse seconds duration"""
        monkeypatch.setenv("TEST_DURATION", "2s")
        result = get_duration_from_env("TEST_DURATION", 0)
        assert result == 2000

    def test_parses_minutes(self, monkeypatch):
        """Should parse minutes duration"""
        monkeypatch.setenv("TEST_DURATION", "1m")
        result = get_duration_from_env("TEST_DURATION", 0)
        assert result == 60000

    def test_returns_default_on_missing(self, monkeypatch):
        """Should return default when env var missing"""
        monkeypatch.delenv("TEST_DURATION", raising=False)
        result = get_duration_from_env("TEST_DURATION", 500)
        assert result == 500


class TestValidateDomain:
    """Tests for validate_domain"""

    def test_accepts_valid_https_domain(self):
        """Should accept valid HTTPS domain"""
        validate_domain("https://api.example.com")  # Should not raise

    def test_accepts_localhost_http(self):
        """Should accept HTTP localhost for testing"""
        validate_domain("http://localhost:8080")  # Should not raise

    def test_rejects_empty_domain(self):
        """Should reject empty domain"""
        with pytest.raises(ValueError, match="cannot be empty"):
            validate_domain("")

    def test_rejects_http_non_localhost(self):
        """Should reject HTTP for non-localhost"""
        with pytest.raises(ValueError, match="must use HTTPS"):
            validate_domain("http://api.example.com")

    def test_rejects_domain_with_path(self):
        """Should reject domain with path components"""
        with pytest.raises(ValueError, match="cannot contain path"):
            validate_domain("https://api.example.com/path")


class TestValidatePathSegment:
    """Tests for validate_path_segment"""

    def test_accepts_valid_segment(self):
        """Should accept valid path segment"""
        validate_path_segment("valid-segment", "test")  # Should not raise

    def test_rejects_empty_segment(self):
        """Should reject empty segment"""
        with pytest.raises(ValueError, match="cannot be empty"):
            validate_path_segment("", "test")

    def test_rejects_path_traversal(self):
        """Should reject path traversal attempts"""
        with pytest.raises(ValueError, match="path traversal"):
            validate_path_segment("../parent", "test")

    def test_rejects_slash(self):
        """Should reject slash in segment"""
        with pytest.raises(ValueError, match="path traversal"):
            validate_path_segment("a/b", "test")

    def test_rejects_encoded_traversal(self):
        """Should reject URL-encoded path traversal"""
        with pytest.raises(ValueError, match="path traversal"):
            validate_path_segment("%2e%2e", "test")

    def test_rejects_null_byte(self):
        """Should reject null byte"""
        with pytest.raises(ValueError, match="null byte"):
            validate_path_segment("test\x00", "test")
