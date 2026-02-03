"""
Tests for handlers matching Go and TypeScript test structure
"""

import json
from pathlib import Path
from unittest.mock import MagicMock

import pytest

from src.logger import Logger
from src.types import ConnectEvent

# Read version from VERSION file for test verification
VERSION_FILE_PATH = Path(__file__).parent.parent.parent.parent / "VERSION"
EXPECTED_VERSION = VERSION_FILE_PATH.read_text().strip()

# Patch VERSION before importing handlers module to ensure correct version in tests
import src.version  # noqa: E402

src.version.VERSION = EXPECTED_VERSION

from src.handlers import Handlers  # noqa: E402


@pytest.fixture
def logger():
    """Create a logger instance"""
    return Logger()


@pytest.fixture
def mock_api_client():
    """Create a mock API client"""
    return MagicMock()


class TestGetPSTNTransferData:
    """Tests for get_pstn_transfer_data handler"""

    def test_success_with_filtered_parameters(self, logger, mock_api_client):
        """Should successfully make request with filtered parameters"""
        mock_response = {
            "phoneNumber": "+1234567890",
            "dtmfSequence": "1234",
        }
        mock_api_client.make_request.return_value = json.dumps(mock_response).encode("utf-8")

        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test-contact-id"},
                "Parameters": {
                    "customParam": "customValue",
                    "apiKey": "should-be-filtered",
                    "region": "should-be-filtered",
                },
            }
        )

        handlers = Handlers(
            logger=logger,
            api_client=mock_api_client,
            domain="https://api.example.com",
            customer_id="test-customer",
            profile_id="test-profile",
            virtual_agent_id="test-agent",
            supported_dtmf_chars="0123456789*",
            event=event,
        )

        result = handlers.get_pstn_transfer_data()

        assert result is not None
        assert result["phoneNumber"] == "+1234567890"
        assert result["dtmfSequence"] == "1234"

        # Verify the request was made with correct parameters
        mock_api_client.make_request.assert_called_once()
        call_args = mock_api_client.make_request.call_args

        assert call_args[0][0] == "POST"
        assert call_args[0][1] == (
            "https://api.example.com/v1/customers/test-customer/"
            "profiles/test-profile/virtualAgents/test-agent:generatePSTNTransferData"
        )

        payload = call_args[0][2]
        assert payload["callId"] == "test-contact-id"
        assert payload["supportedDtmfChars"] == "0123456789*"
        assert "ccaasMetadata" in payload
        assert payload["ccaasMetadata"]["ContactId"] == "test-contact-id"
        assert payload["ccaasMetadata"]["parameters"]["customParam"] == "customValue"

        # Verify filtered keys are not in parameters
        assert "apiKey" not in payload["ccaasMetadata"]["parameters"]
        assert "region" not in payload["ccaasMetadata"]["parameters"]

        # Verify version is present and matches VERSION file
        assert "version" in payload["ccaasMetadata"]
        assert payload["ccaasMetadata"]["version"] == EXPECTED_VERSION

    def test_handles_error_response(self, logger, mock_api_client):
        """Should handle error response from server"""
        mock_api_client.make_request.side_effect = ValueError(
            'request returned non-200 status: 400, body: {"error":"bad request"}'
        )

        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test-contact-id"},
                "Parameters": {},
            }
        )

        handlers = Handlers(
            logger=logger,
            api_client=mock_api_client,
            domain="https://api.example.com",
            customer_id="test-customer",
            profile_id="test-profile",
            virtual_agent_id="test-agent",
            supported_dtmf_chars="0123456789*",
            event=event,
        )

        with pytest.raises(ValueError):
            handlers.get_pstn_transfer_data()


class TestGetHandoffData:
    """Tests for get_handoff_data handler"""

    def test_success_transforms_response(self, logger, mock_api_client):
        """Should successfully make request and transform response"""
        mock_response = {
            "handoff": {
                "conversation": "conversation-id",
                "conversationCorrelationId": "correlation-id",
                "summary": "test summary",
                "transferTarget": "pstn:PSTN1",
            }
        }
        mock_api_client.make_request.return_value = json.dumps(mock_response).encode("utf-8")

        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test-contact-id"},
                "Parameters": {},
            }
        )

        handlers = Handlers(
            logger=logger,
            api_client=mock_api_client,
            domain="https://api.example.com",
            customer_id="test-customer",
            profile_id="test-profile",
            virtual_agent_id="",
            supported_dtmf_chars="0123456789*",
            event=event,
        )

        result = handlers.get_handoff_data()

        assert result is not None
        assert result["handoff_conversation"] == "conversation-id"
        assert result["handoff_conversationCorrelationId"] == "correlation-id"
        assert result["handoff_summary"] == "test summary"
        assert result["handoff_transferTarget"] == "pstn:PSTN1"

        # Verify the request was made with correct parameters
        mock_api_client.make_request.assert_called_once()
        call_args = mock_api_client.make_request.call_args

        assert call_args[0][0] == "POST"
        assert call_args[0][1] == (
            "https://api.example.com/v1/customers/test-customer/"
            "profiles/test-profile/handoffs:fetchAIAgentHandoff"
        )
        assert call_args[0][2] == {"correlationId": "test-contact-id"}

    def test_handles_error_response(self, logger, mock_api_client):
        """Should handle error response from server"""
        mock_api_client.make_request.side_effect = ValueError(
            'request returned non-200 status: 404, body: {"error":"not found"}'
        )

        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test-contact-id"},
                "Parameters": {},
            }
        )

        handlers = Handlers(
            logger=logger,
            api_client=mock_api_client,
            domain="https://api.example.com",
            customer_id="test-customer",
            profile_id="test-profile",
            virtual_agent_id="",
            supported_dtmf_chars="0123456789*",
            event=event,
        )

        with pytest.raises(ValueError):
            handlers.get_handoff_data()

    def test_validates_handoff_response_structure(self, logger, mock_api_client):
        """Should validate handoff response structure"""
        # Missing handoff field
        mock_api_client.make_request.return_value = json.dumps({}).encode("utf-8")

        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test-contact-id"},
                "Parameters": {},
            }
        )

        handlers = Handlers(
            logger=logger,
            api_client=mock_api_client,
            domain="https://api.example.com",
            customer_id="test-customer",
            profile_id="test-profile",
            virtual_agent_id="",
            supported_dtmf_chars="0123456789*",
            event=event,
        )

        with pytest.raises(ValueError, match="invalid handoff response"):
            handlers.get_handoff_data()

    def test_validates_required_handoff_fields(self, logger, mock_api_client):
        """Should validate required handoff fields"""
        # Missing required field
        mock_response = {
            "handoff": {
                "conversation": "conversation-id",
                # Missing other required fields
            }
        }
        mock_api_client.make_request.return_value = json.dumps(mock_response).encode("utf-8")

        event = ConnectEvent(
            Details={
                "ContactData": {"ContactId": "test-contact-id"},
                "Parameters": {},
            }
        )

        handlers = Handlers(
            logger=logger,
            api_client=mock_api_client,
            domain="https://api.example.com",
            customer_id="test-customer",
            profile_id="test-profile",
            virtual_agent_id="",
            supported_dtmf_chars="0123456789*",
            event=event,
        )

        with pytest.raises(ValueError, match="missing or invalid required fields"):
            handlers.get_handoff_data()
