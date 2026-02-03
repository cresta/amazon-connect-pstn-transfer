"""
Data models matching the Go and TypeScript implementations
"""

from dataclasses import dataclass
from typing import Any


@dataclass
class ConnectEvent:
    """Amazon Connect event structure"""

    Details: dict[str, Any]
    Name: str | None = None

    @property
    def contact_data(self) -> dict[str, Any]:
        """Get ContactData from Details"""
        return self.Details.get("ContactData", {})

    @property
    def contact_id(self) -> str:
        """Get ContactId from ContactData"""
        return self.contact_data.get("ContactId", "")

    @property
    def parameters(self) -> dict[str, str]:
        """Get Parameters from Details"""
        return self.Details.get("Parameters", {})


# Type alias for Connect response
ConnectResponse = dict[str, Any]


@dataclass
class Handoff:
    """Handoff data structure"""

    conversation: str
    conversationCorrelationId: str
    summary: str
    transferTarget: str


@dataclass
class FetchAIAgentHandoffResponse:
    """Response from fetchAIAgentHandoff API"""

    handoff: Handoff


@dataclass
class OAuthCredentials:
    """OAuth credentials structure"""

    oauth_client_id: str
    oauth_client_secret: str


@dataclass
class TokenResponse:
    """OAuth token response"""

    access_token: str
    token_type: str
    expires_in: int
