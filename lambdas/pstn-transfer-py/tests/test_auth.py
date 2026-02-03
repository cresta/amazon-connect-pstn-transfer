"""
Tests for OAuth 2 authentication matching Go and TypeScript test structure
"""

import json
from unittest.mock import MagicMock, patch

import pytest

from src.auth import DefaultOAuth2TokenFetcher, TokenCache, get_token_cache


class TestTokenCache:
    """Tests for TokenCache"""

    def test_get_token_returns_none_when_empty(self):
        """Should return None when cache is empty"""
        cache = TokenCache()
        result = cache.get_token("client-id")
        assert result is None

    def test_set_and_get_token(self):
        """Should set and retrieve token"""
        cache = TokenCache()
        cache.set_token("client-id", "test-token", 3600)
        result = cache.get_token("client-id")
        assert result == "test-token"

    def test_caches_short_lived_tokens_with_adaptive_buffer(self):
        """Should cache short-lived tokens with adaptive safety buffer (matching Go)"""
        import time as time_module
        from unittest.mock import patch

        cache = TokenCache()
        initial_time = 1000000.0

        with patch.object(time_module, "time") as mock_time:
            mock_time.return_value = initial_time
            # For 300 second token, safety buffer = 300 // 2 = 150
            # So token should be cached for 150 seconds
            cache.set_token("client-id", "short-token", 300)
            result = cache.get_token("client-id")
            assert result == "short-token"

            # Token should still be valid after 100 seconds
            mock_time.return_value = initial_time + 100
            result = cache.get_token("client-id")
            assert result == "short-token"

            # Token should be expired after 151 seconds (past the 150s cache time)
            mock_time.return_value = initial_time + 151
            result = cache.get_token("client-id")
            assert result is None

    def test_returns_none_for_expired_token(self):
        """Should return None for expired token"""
        cache = TokenCache()

        # Mock time to control expiration deterministically
        import time as time_module
        from unittest.mock import patch

        initial_time = 1000000.0

        with patch.object(time_module, "time") as mock_time:
            # Set time for when token is stored
            mock_time.return_value = initial_time
            # Set token with 301 second expiration (effective TTL = 301 - 300 = 1 second)
            cache.set_token("client-id", "expired-token", 301)

            # Token should still be valid immediately after setting
            result = cache.get_token("client-id")
            assert result == "expired-token"

            # Advance time past the effective TTL (1 second + buffer)
            mock_time.return_value = initial_time + 2

            # Token should now be expired
            result = cache.get_token("client-id")
            assert result is None

    def test_clear_token(self):
        """Should clear token from cache"""
        cache = TokenCache()
        cache.set_token("client-id", "test-token", 3600)
        cache.clear_token("client-id")
        result = cache.get_token("client-id")
        assert result is None


class TestDefaultOAuth2TokenFetcher:
    """Tests for DefaultOAuth2TokenFetcher"""

    def test_returns_cached_token(self):
        """Should return cached token without making request"""
        # Pre-populate cache
        cache = get_token_cache()
        cache.set_token("test-client-id", "cached-token", 3600)

        fetcher = DefaultOAuth2TokenFetcher()
        result = fetcher.get_token(
            auth_domain="https://auth.example.com",
            client_id="test-client-id",
            client_secret="test-secret",
        )

        assert result == "cached-token"

        # Clean up
        cache.clear_token("test-client-id")

    @patch("urllib.request.urlopen")
    def test_fetches_and_caches_new_token(self, mock_urlopen):
        """Should fetch new token and cache it"""
        # Clear cache first
        cache = get_token_cache()
        cache.clear_token("new-client-id")

        # Mock response
        mock_response = MagicMock()
        mock_response.status = 200
        mock_response.read.return_value = json.dumps(
            {
                "access_token": "new-token",
                "token_type": "Bearer",
                "expires_in": 3600,
            }
        ).encode("utf-8")
        mock_response.__enter__ = MagicMock(return_value=mock_response)
        mock_response.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_response

        fetcher = DefaultOAuth2TokenFetcher()
        result = fetcher.get_token(
            auth_domain="https://auth.example.com",
            client_id="new-client-id",
            client_secret="test-secret",
        )

        assert result == "new-token"

        # Verify token was cached
        cached = cache.get_token("new-client-id")
        assert cached == "new-token"

        # Clean up
        cache.clear_token("new-client-id")

    @patch("urllib.request.urlopen")
    def test_raises_on_missing_access_token(self, mock_urlopen):
        """Should raise error when access_token is missing"""
        # Clear cache first
        cache = get_token_cache()
        cache.clear_token("error-client-id")

        # Mock response without access_token
        mock_response = MagicMock()
        mock_response.status = 200
        mock_response.read.return_value = json.dumps(
            {
                "token_type": "Bearer",
                "expires_in": 3600,
            }
        ).encode("utf-8")
        mock_response.__enter__ = MagicMock(return_value=mock_response)
        mock_response.__exit__ = MagicMock(return_value=False)
        mock_urlopen.return_value = mock_response

        fetcher = DefaultOAuth2TokenFetcher()

        with pytest.raises(ValueError, match="missing access_token"):
            fetcher.get_token(
                auth_domain="https://auth.example.com",
                client_id="error-client-id",
                client_secret="test-secret",
            )
