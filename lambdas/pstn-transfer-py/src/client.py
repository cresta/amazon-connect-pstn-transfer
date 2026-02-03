"""
Cresta API client matching the Go and TypeScript implementations
"""

from __future__ import annotations

import json
from typing import TYPE_CHECKING, Any

from .httpclient import AuthConfig, RetryHTTPClient

if TYPE_CHECKING:
    from .logger import Logger


class CrestaAPIClient:
    """API client for Cresta services"""

    def __init__(self, logger: Logger, auth_config: AuthConfig) -> None:
        if not auth_config:
            raise ValueError("authConfig is required for CrestaAPIClient")

        self._logger = logger
        self._client = RetryHTTPClient(
            logger=logger,
            auth_config=auth_config,
        )

    def make_request(
        self,
        method: str,
        url: str,
        payload: Any,
    ) -> bytes:
        """Make API request and return response body"""
        json_data = json.dumps(payload)
        self._logger.debugf("Sending request to %s with payload: %s", url, json_data)

        headers = {
            "Content-Type": "application/json",
        }

        response = self._client.fetch(
            method=method,
            url=url,
            headers=headers,
            body=json_data.encode("utf-8"),
        )

        return response
