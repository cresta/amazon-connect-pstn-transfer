"""
HTTP client with retry logic matching the Go and TypeScript implementations
"""

from __future__ import annotations

import random
import time
import urllib.error
import urllib.request
from abc import ABC, abstractmethod
from typing import TYPE_CHECKING

from .types import AuthConfig
from .utils import get_duration_from_env, get_int_from_env

if TYPE_CHECKING:
    from .logger import Logger

# Configuration from environment
HTTP_MAX_RETRIES = get_int_from_env("HTTP_MAX_RETRIES", 3)
HTTP_RETRY_BASE_DELAY = get_duration_from_env("HTTP_RETRY_BASE_DELAY", 100)
HTTP_CLIENT_TIMEOUT = get_duration_from_env("HTTP_CLIENT_TIMEOUT", 10000)


class HTTPClient(ABC):
    """Abstract HTTP client interface"""

    @abstractmethod
    def fetch(
        self,
        method: str,
        url: str,
        headers: dict[str, str],
        body: bytes | None = None,
    ) -> bytes:
        """Make HTTP request and return response body"""
        pass


def is_retryable_error(err: Exception | None, status_code: int) -> bool:
    """
    Determines if an error or status code should trigger a retry.
    Matches the Go implementation: retries network errors (err != None), 5xx status codes,
    429 (Too Many Requests), and 408 (Request Timeout)
    """
    if err:
        # Network errors are retryable
        return True
    # Retry on 5xx server errors, 429 (Too Many Requests), and 408 (Request Timeout)
    return (500 <= status_code < 600) or status_code == 429 or status_code == 408


def exponential_backoff(attempt: int, base_delay_ms: int) -> int:
    """Calculates the delay for the given attempt with jitter"""
    delay = (2**attempt) * base_delay_ms
    # Add jitter: random value between 0 and 25% of delay
    jitter = random.random() * 0.25 * delay
    return int(delay + jitter)


class RetryHTTPClient(HTTPClient):
    """HTTP client with retry logic"""

    def __init__(
        self,
        logger: Logger | None = None,
        auth_config: AuthConfig | None = None,
        max_retries: int | None = None,
        base_delay: int | None = None,
    ) -> None:
        self._logger = logger
        self._auth_config = auth_config
        self._max_retries = max_retries if max_retries is not None else HTTP_MAX_RETRIES
        self._base_delay = base_delay if base_delay is not None else HTTP_RETRY_BASE_DELAY

    def fetch(
        self,
        method: str,
        url: str,
        headers: dict[str, str],
        body: bytes | None = None,
    ) -> bytes:
        """Make HTTP request with retry logic"""
        last_err: Exception | None = None
        last_status: int | None = None

        for attempt in range(self._max_retries + 1):
            if attempt > 0:
                delay = exponential_backoff(attempt - 1, self._base_delay)
                if self._logger:
                    self._logger.debugf(
                        "Retrying request to %s (attempt %d/%d) after %dms",
                        url,
                        attempt + 1,
                        self._max_retries + 1,
                        delay,
                    )
                time.sleep(delay / 1000.0)  # Convert ms to seconds

            # Add authentication header if configured
            request_headers = headers.copy()
            if self._auth_config:
                try:
                    auth_header = self._get_auth_header()
                    if auth_header and "Authorization" not in request_headers:
                        request_headers["Authorization"] = auth_header
                except Exception as auth_err:
                    # Auth failures should not be retried - fail fast
                    raise ValueError(f"error getting auth header: {auth_err}") from auth_err

            try:
                request = urllib.request.Request(
                    url,
                    data=body,
                    headers=request_headers,
                    method=method,
                )

                timeout_seconds = HTTP_CLIENT_TIMEOUT / 1000.0

                with urllib.request.urlopen(request, timeout=timeout_seconds) as response:
                    status_code = response.status
                    response_body: bytes = response.read()

                    # Check if status code is retryable
                    if not is_retryable_error(None, status_code):
                        # Non-retryable, return immediately
                        if status_code != 200:
                            raise ValueError(
                                f"request returned non-200 status: {status_code}, body: {response_body.decode('utf-8')}"
                            )
                        return response_body

                    # Retryable status code
                    last_status = status_code
                    last_err = ValueError(f"request returned retryable status: {status_code}")

            except urllib.error.HTTPError as e:
                status_code = e.code
                error_body = e.read().decode("utf-8") if e.fp else ""

                # Check if status code is retryable
                if not is_retryable_error(None, status_code):
                    # Non-retryable error
                    raise ValueError(
                        f"request returned non-200 status: {status_code}, body: {error_body}"
                    ) from e

                # Retryable status code
                last_status = status_code
                last_err = ValueError(f"request returned retryable status: {status_code}")

            except urllib.error.URLError as e:
                # Network error - retryable
                last_err = e

            except TimeoutError as e:
                # Timeout - retryable
                last_err = e

            except Exception as e:
                # Check if this error should be retried
                if not is_retryable_error(e, 0):
                    raise ValueError(f"error making HTTP request: {e}") from e
                last_err = e

        # All retries exhausted
        if last_err:
            raise ValueError(f"request failed after {self._max_retries + 1} attempts: {last_err}")
        if last_status:
            raise ValueError(
                f"request failed after {self._max_retries + 1} attempts with status: {last_status}"
            )
        raise ValueError(f"request failed after {self._max_retries + 1} attempts")

    def _get_auth_header(self) -> str:
        """Get authentication header value"""
        if not self._auth_config:
            raise ValueError("authConfig is required")

        # OAuth 2 authentication takes precedence
        if self._auth_config.oauth_client_id and self._auth_config.oauth_client_secret:
            if not self._auth_config.token_fetcher:
                raise ValueError("tokenFetcher is required for OAuth authentication")
            if not self._auth_config.auth_domain:
                raise ValueError("authDomain is required for OAuth authentication")

            token = self._auth_config.token_fetcher.get_token(
                self._auth_config.auth_domain,
                self._auth_config.oauth_client_id,
                self._auth_config.oauth_client_secret,
            )
            return f"Bearer {token}"

        # Fall back to API key authentication (deprecated)
        if self._auth_config.api_key:
            return f"ApiKey {self._auth_config.api_key}"

        raise ValueError("no authentication configured")
