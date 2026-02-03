"""
Data models matching the Go and TypeScript implementations
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from .auth import OAuth2TokenFetcher


@dataclass
class ConnectEvent:
    """Amazon Connect event structure"""

    Details: dict[str, Any]
    Name: str | None = None

    @property
    def contact_data(self) -> dict[str, Any]:
        """Get ContactData from Details"""
        result: dict[str, Any] = self.Details.get("ContactData", {})
        return result

    @property
    def contact_id(self) -> str:
        """Get ContactId from ContactData"""
        result: str = self.contact_data.get("ContactId", "")
        return result

    @property
    def parameters(self) -> dict[str, str]:
        """Get Parameters from Details"""
        result: dict[str, str] = self.Details.get("Parameters", {})
        return result


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


@dataclass
class AuthConfig:
    """Authentication configuration"""

    api_key: str | None = None  # Deprecated
    auth_domain: str | None = None
    oauth_client_id: str | None = None
    oauth_client_secret: str | None = None
    token_fetcher: OAuth2TokenFetcher | None = None
