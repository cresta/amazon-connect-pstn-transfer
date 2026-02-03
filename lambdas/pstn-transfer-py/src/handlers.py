"""
Handlers for different API actions matching the Go and TypeScript implementations
"""

import json
from typing import TYPE_CHECKING, Any

from . import version as version_module
from .types import ConnectEvent, ConnectResponse
from .utils import copy_map

if TYPE_CHECKING:
    from .client import CrestaAPIClient
    from .logger import Logger

# Keys to filter out from parameters
FILTERED_KEYS: set[str] = {
    "apiDomain",
    "region",
    "action",
    "apiKey",
    "oauthClientId",
    "oauthClientSecret",
    "virtualAgentName",
    "supportedDtmfChars",
}


class Handlers:
    """Business logic handlers for Lambda actions"""

    def __init__(
        self,
        logger: Logger,
        api_client: CrestaAPIClient,
        domain: str,
        customer_id: str,
        profile_id: str,
        virtual_agent_id: str,
        supported_dtmf_chars: str,
        event: ConnectEvent,
    ):
        self._logger = logger
        self._api_client = api_client
        self._domain = domain
        self._customer_id = customer_id
        self._profile_id = profile_id
        self._virtual_agent_id = virtual_agent_id
        self._supported_dtmf_chars = supported_dtmf_chars
        self._event = event

    def get_pstn_transfer_data(self) -> ConnectResponse:
        """Get PSTN transfer data (phone number and DTMF sequence)"""
        virtual_agent_name = (
            f"customers/{self._customer_id}/profiles/{self._profile_id}"
            f"/virtualAgents/{self._virtual_agent_id}"
        )
        url = f"{self._domain}/v1/{virtual_agent_name}:generatePSTNTransferData"

        filtered_parameters = copy_map(self._event.parameters, FILTERED_KEYS)

        # Merge ContactData with parameters as a sub-field of ccaasMetadata
        ccaas_metadata: dict[str, Any] = {
            **self._event.contact_data,
            "parameters": filtered_parameters,
            "version": version_module.VERSION,
        }

        payload = {
            "callId": self._event.contact_id,
            "ccaasMetadata": ccaas_metadata,
            "supportedDtmfChars": self._supported_dtmf_chars,
        }

        self._logger.debugf("Making request to %s with payload: %+v", url, payload)

        body = self._api_client.make_request("POST", url, payload)

        try:
            result = json.loads(body.decode("utf-8"))
        except json.JSONDecodeError as e:
            raise ValueError(f"failed to parse JSON response from {url}: {e}") from e

        self._logger.debugf("Received response: %+v", result)
        return result

    def get_handoff_data(self) -> ConnectResponse:
        """Get handoff data for BOT conversations"""
        url = (
            f"{self._domain}/v1/customers/{self._customer_id}"
            f"/profiles/{self._profile_id}/handoffs:fetchAIAgentHandoff"
        )

        payload = {
            "correlationId": self._event.contact_id,
        }

        self._logger.debugf("Making request to %s with payload: %+v", url, payload)

        body = self._api_client.make_request("POST", url, payload)

        try:
            decoded_body = body.decode("utf-8")
            parsed = json.loads(decoded_body)
        except json.JSONDecodeError as e:
            raise ValueError(f"failed to parse JSON response from {url}: {e}") from e

        # Validate response structure
        if (
            not parsed
            or not isinstance(parsed, dict)
            or "handoff" not in parsed
            or not parsed["handoff"]
            or not isinstance(parsed["handoff"], dict)
        ):
            raise ValueError(f"invalid handoff response structure: {json.dumps(parsed)}")

        handoff = parsed["handoff"]
        self._logger.debugf("Received response: %+v", parsed)

        # Validate required handoff fields exist and are the expected types
        required_fields = ["conversation", "conversationCorrelationId", "summary", "transferTarget"]
        for field in required_fields:
            if field not in handoff or not isinstance(handoff[field], str):
                raise ValueError(
                    "invalid handoff response: missing or invalid required fields "
                    "(conversation, conversationCorrelationId, summary, transferTarget)"
                )

        return {
            "handoff_conversation": handoff["conversation"],
            "handoff_conversationCorrelationId": handoff["conversationCorrelationId"],
            "handoff_summary": handoff["summary"],
            "handoff_transferTarget": handoff["transferTarget"],
        }
