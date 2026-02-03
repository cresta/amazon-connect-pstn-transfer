"""
OAuth 2 authentication with token caching matching the Go and TypeScript implementations
"""

import base64
import json
import time
import urllib.error
import urllib.request
from abc import ABC, abstractmethod
from dataclasses import dataclass
from threading import Lock
from typing import TYPE_CHECKING, Optional

if TYPE_CHECKING:
    from .httpclient import HTTPClient
    from .logger import Logger


@dataclass
class CacheEntry:
    """Token cache entry"""

    token: str
    expires_at: float  # Unix timestamp


class TokenCache:
    """Thread-safe token cache"""

    def __init__(self):
        self._cache: dict[str, CacheEntry] = {}
        self._lock = Lock()

    def _cache_key(self, client_id: str) -> str:
        return f"pstn-transfer:tokencache:{client_id}"

    def get_token(self, client_id: str) -> str | None:
        """Get token from cache if not expired"""
        key = self._cache_key(client_id)
        with self._lock:
            entry = self._cache.get(key)
            if entry and entry.token and time.time() < entry.expires_at:
                return entry.token
        return None

    def set_token(self, client_id: str, token: str, expires_in_seconds: int) -> None:
        """Set token in cache with expiration"""
        key = self._cache_key(client_id)

        # Skip caching for tokens that are too short-lived (<= 300 seconds)
        # to avoid setting an expires_at in the past
        if expires_in_seconds <= 300:
            return

        # Subtract 5 minute buffer for safety
        expires_at = time.time() + (expires_in_seconds - 5 * 60)

        with self._lock:
            self._cache[key] = CacheEntry(token=token, expires_at=expires_at)

    def clear_token(self, client_id: str) -> None:
        """Clear token from cache"""
        key = self._cache_key(client_id)
        with self._lock:
            self._cache.pop(key, None)


# Global token cache instance
_token_cache = TokenCache()


def get_token_cache() -> TokenCache:
    """Get the global token cache (for testing)"""
    return _token_cache


class OAuth2TokenFetcher(ABC):
    """Abstract base class for OAuth2 token fetchers"""

    @abstractmethod
    def get_token(
        self,
        auth_domain: str,
        client_id: str,
        client_secret: str,
    ) -> str:
        """Fetch OAuth2 token"""
        pass


class DefaultOAuth2TokenFetcher(OAuth2TokenFetcher):
    """Default implementation of OAuth2 token fetcher"""

    def __init__(self, client: Optional["HTTPClient"] = None, logger: Optional["Logger"] = None):
        from .logger import new_logger

        self._logger = logger or new_logger()
        self._client = client

    def get_token(
        self,
        auth_domain: str,
        client_id: str,
        client_secret: str,
    ) -> str:
        """Fetch OAuth2 token with caching"""
        # Check cache first (use client_id as cache key)
        cached_token = _token_cache.get_token(client_id)
        if cached_token:
            return cached_token

        # Build token URL from auth_domain (domain only, append path)
        token_url = f"{auth_domain}/v1/oauth/regionalToken"

        # Prepare JSON payload
        payload = {"grant_type": "client_credentials"}

        # Create Basic Auth header
        credentials = f"{client_id}:{client_secret}"
        auth_bytes = base64.b64encode(credentials.encode("utf-8")).decode("utf-8")

        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Basic {auth_bytes}",
        }

        # Make request
        if self._client:
            # Use provided HTTP client (for retry logic)
            response_data = self._client.fetch(
                method="POST",
                url=token_url,
                headers=headers,
                body=json.dumps(payload).encode("utf-8"),
            )
            token_response = json.loads(response_data.decode("utf-8"))
        else:
            # Use urllib directly for simple token requests
            request = urllib.request.Request(
                token_url,
                data=json.dumps(payload).encode("utf-8"),
                headers=headers,
                method="POST",
            )

            try:
                with urllib.request.urlopen(request, timeout=30) as response:
                    if response.status != 200:
                        body = response.read().decode("utf-8")
                        raise ValueError(
                            f"token request returned non-200 status: {response.status}, body: {body}"
                        )
                    token_response = json.loads(response.read().decode("utf-8"))
            except urllib.error.HTTPError as e:
                body = e.read().decode("utf-8") if e.fp else ""
                raise ValueError(
                    f"token request returned non-200 status: {e.code}, body: {body}"
                ) from e

        if "access_token" not in token_response or not token_response["access_token"]:
            raise ValueError("missing access_token in token response")

        access_token = token_response["access_token"]
        expires_in = token_response.get("expires_in", 0)

        # Cache the token (use client_id as cache key)
        if expires_in > 0:
            _token_cache.set_token(client_id, access_token, expires_in)

        return access_token
