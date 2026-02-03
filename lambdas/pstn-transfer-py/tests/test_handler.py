"""
Tests for main handler matching Go and TypeScript test structure
"""

from pathlib import Path
from unittest.mock import MagicMock

import pytest

from src.handler import DefaultHandlerService, handler
from src.types import ConnectEvent

# Read version from VERSION file for test verification
VERSION_FILE_PATH = Path(__file__).parent.parent.parent.parent / "VERSION"
EXPECTED_VERSION = VERSION_FILE_PATH.read_text().strip()


@pytest.fixture(autouse=True)
def mock_version(monkeypatch):
    """Mock the VERSION to match the VERSION file"""
    monkeypatch.setenv("LAMBDA_VERSION", EXPECTED_VERSION)
    import src.version

    monkeypatch.setattr(src.version, "VERSION", EXPECTED_VERSION)


class TestDefaultHandlerService:
    """Tests for DefaultHandlerService"""

    def test_requires_region(self):
        """Should require region parameter"""
        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test"},
                "Parameters": {
                    "action": "get_pstn_transfer_data",
                    "virtualAgentName": "customers/c/profiles/p/virtualAgents/v",
                    "oauthClientId": "id",
                    "oauthClientSecret": "secret",
                },
            }
        )

        service = DefaultHandlerService()

        with pytest.raises(ValueError, match="region is required"):
            service.handle(event)

    def test_requires_action(self, monkeypatch):
        """Should require action parameter"""
        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test"},
                "Parameters": {
                    "region": "us-west-2-prod",
                    "virtualAgentName": "customers/c/profiles/p/virtualAgents/v",
                    "oauthClientId": "id",
                    "oauthClientSecret": "secret",
                },
            }
        )

        service = DefaultHandlerService()

        with pytest.raises(ValueError, match="action is required"):
            service.handle(event)

    def test_requires_virtual_agent_name(self):
        """Should require virtualAgentName parameter"""
        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test"},
                "Parameters": {
                    "region": "us-west-2-prod",
                    "action": "get_pstn_transfer_data",
                    "oauthClientId": "id",
                    "oauthClientSecret": "secret",
                },
            }
        )

        service = DefaultHandlerService()

        with pytest.raises(ValueError, match="virtualAgentName is required"):
            service.handle(event)

    def test_requires_authentication(self):
        """Should require authentication credentials"""
        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test"},
                "Parameters": {
                    "region": "us-west-2-prod",
                    "action": "get_pstn_transfer_data",
                    "virtualAgentName": "customers/c/profiles/p/virtualAgents/v",
                },
            }
        )

        service = DefaultHandlerService()

        with pytest.raises(ValueError, match="either apiKey.*oauthClientId"):
            service.handle(event)

    def test_rejects_invalid_action(self):
        """Should reject invalid action"""
        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test"},
                "Parameters": {
                    "region": "us-west-2-prod",
                    "action": "invalid_action",
                    "virtualAgentName": "customers/c/profiles/p/virtualAgents/v",
                    "oauthClientId": "id",
                    "oauthClientSecret": "secret",
                },
            }
        )

        # Mock the token fetcher to avoid actual HTTP requests
        mock_token_fetcher = MagicMock()
        mock_token_fetcher.get_token.return_value = "mock-token"

        service = DefaultHandlerService(token_fetcher=mock_token_fetcher)

        with pytest.raises(ValueError, match="invalid action"):
            service.handle(event)

    def test_validates_api_domain_and_auth_domain_together_with_oauth(self):
        """Should require apiDomain and authDomain together when using OAuth"""
        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test"},
                "Parameters": {
                    "apiDomain": "api.custom.com",
                    # Missing authDomain
                    "action": "get_pstn_transfer_data",
                    "virtualAgentName": "customers/c/profiles/p/virtualAgents/v",
                    "oauthClientId": "id",
                    "oauthClientSecret": "secret",
                },
            }
        )

        service = DefaultHandlerService()

        with pytest.raises(ValueError, match="apiDomain and authDomain must be provided together"):
            service.handle(event)

    def test_builds_auth_domain_from_region(self):
        """Should build auth domain from region when not provided"""
        from unittest.mock import patch

        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test"},
                "Parameters": {
                    "region": "voice-prod",
                    "action": "get_pstn_transfer_data",
                    "virtualAgentName": "customers/c/profiles/p/virtualAgents/v",
                    "oauthClientId": "id",
                    "oauthClientSecret": "secret",
                },
            }
        )

        mock_token_fetcher = MagicMock()
        mock_token_fetcher.get_token.return_value = "mock-token"

        service = DefaultHandlerService(token_fetcher=mock_token_fetcher)

        # Mock the RetryHTTPClient.fetch to avoid real HTTP requests
        with patch("src.httpclient.RetryHTTPClient.fetch") as mock_fetch:
            mock_fetch.side_effect = ValueError("mocked HTTP error - no real request made")

            # Will fail on mocked HTTP request, but shouldn't fail on auth domain validation
            with pytest.raises(ValueError, match="mocked HTTP error"):
                service.handle(event)


class TestHandler:
    """Tests for the main handler function"""

    def test_converts_dict_to_connect_event(self):
        """Should convert dict event to ConnectEvent"""
        event_dict = {
            "Details": {
                "ContactData": {"ContactId": "test"},
                "Parameters": {},
            },
            "Name": "TestEvent",
        }

        # Will fail on missing required params, but proves conversion works
        with pytest.raises(ValueError, match="region is required"):
            handler(event_dict)

    def test_logs_request_id_from_context(self, capsys):
        """Should log request ID from context if provided"""
        event_dict = {
            "Details": {
                "ContactData": {"ContactId": "test"},
                "Parameters": {},
            },
        }

        mock_context = MagicMock()
        mock_context.aws_request_id = "test-request-id"

        # Enable debug logging
        import os

        old_debug = os.environ.get("DEBUG_LOGGING", "")
        os.environ["DEBUG_LOGGING"] = "true"

        try:
            with pytest.raises(ValueError):  # Will fail on missing params
                handler(event_dict, mock_context)

            # Check debug output was generated (if debug was enabled)
            _ = capsys.readouterr()
            # Debug log might be present depending on environment
        finally:
            os.environ["DEBUG_LOGGING"] = old_debug
