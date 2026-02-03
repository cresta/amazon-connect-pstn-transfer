"""
Utility functions matching the Go and TypeScript implementations
"""

import os
import re
from typing import Any
from urllib.parse import urlparse

from .types import ConnectEvent

# Regex patterns
API_DOMAIN_REGEX = re.compile(r"api[-.]([a-z0-9-]+)\.cresta\.(ai|com)")
VIRTUAL_AGENT_NAME_REGEX = re.compile(r"^customers/([^/]+)/profiles/([^/]+)/virtualAgents/([^/]+)$")


def get_from_event_parameter_or_env(event: ConnectEvent, key: str, default_value: str) -> str:
    """
    Retrieves a value from event parameters or environment variables.
    Event parameters take precedence over environment variables.
    """
    # Check event parameters first
    if key in event.parameters and event.parameters[key]:
        return event.parameters[key]

    # Fall back to environment variable
    env_value = os.environ.get(key, "")
    if env_value:
        return env_value

    return default_value


def copy_map(original: dict[str, str], filtered_keys: set[str]) -> dict[str, Any]:
    """Creates a copy of a map excluding filtered keys"""
    return {k: v for k, v in original.items() if k not in filtered_keys}


def parse_virtual_agent_name(virtual_agent_name: str) -> tuple[str, str, str]:
    """
    Parses a virtual agent name.
    Format: customers/{customer}/profiles/{profile}/virtualAgents/{virtualAgentID}

    Returns: (customer, profile, virtual_agent_id)
    """
    match = VIRTUAL_AGENT_NAME_REGEX.match(virtual_agent_name)
    if not match or len(match.groups()) != 3:
        raise ValueError(
            f"invalid virtual agent name: {virtual_agent_name}. "
            f"Expected format: customers/{{customer}}/profiles/{{profile}}/virtualAgents/{{virtualAgentID}}"
        )

    return match.group(1), match.group(2), match.group(3)


def build_api_domain_from_region(region: str) -> str:
    """
    Builds an API domain URL from a region.
    e.g., "us-west-2-prod" -> "https://api.us-west-2-prod.cresta.com"
    e.g., "us-west-2-staging" -> "https://api.us-west-2-staging.cresta.ai"
    """
    normalized = region.lower()
    if not re.match(r"^[a-z0-9-]+$", normalized):
        raise ValueError(f"invalid region: {region}")

    if normalized.endswith("-prod"):
        return f"https://api.{normalized}.cresta.com"
    return f"https://api.{normalized}.cresta.ai"


def extract_region_from_domain(api_domain: str) -> str:
    """Extracts the AWS region from the API domain"""
    match = API_DOMAIN_REGEX.search(api_domain)
    if not match:
        raise ValueError(f"could not extract region from domain: {api_domain}")
    return match.group(1)


def get_auth_region(region: str) -> str:
    """
    Maps a region to its corresponding auth region.
    Some custom regions (like chat-prod, voice-prod) map to standard auth regions.
    If no mapping exists, the region is returned as-is.
    """
    # Map of custom regions to their auth regions
    region_to_auth_region = {
        "chat-prod": "us-west-2-prod",  # chat-prod uses us-west-2-prod auth endpoint
        "voice-prod": "us-west-2-prod",  # voice-prod uses us-west-2-prod auth endpoint
    }

    # Check if there's a mapping
    if region in region_to_auth_region:
        return region_to_auth_region[region]

    # Return region as-is (no validation - allows for customer-specific regions)
    return region


def get_int_from_env(key: str, default_value: int) -> int:
    """Retrieves an integer from environment variable or returns default"""
    value = os.environ.get(key, "")
    if value:
        try:
            return int(value)
        except ValueError:
            pass
    return default_value


def get_duration_from_env(key: str, default_value_ms: int) -> int:
    """
    Retrieves a duration from environment variable or returns default.
    Accepts duration strings like "100ms", "2s", "1m", etc.
    Returns value in milliseconds.
    """
    value = os.environ.get(key, "")
    if value:
        match = re.match(r"^(\d+)(ms|s|m|h)$", value)
        if match:
            num = int(match.group(1))
            unit = match.group(2)
            multipliers = {
                "ms": 1,
                "s": 1000,
                "m": 60 * 1000,
                "h": 60 * 60 * 1000,
            }
            return num * multipliers[unit]
    return default_value_ms


def validate_domain(domain: str) -> None:
    """Validates that a domain is a safe URL for API requests"""
    if not domain:
        raise ValueError("domain cannot be empty")

    try:
        parsed_url = urlparse(domain)
    except Exception as e:
        raise ValueError(f"invalid domain URL: {e}") from e

    # Require HTTPS scheme for security, except for localhost (testing)
    is_localhost = (
        parsed_url.hostname == "localhost"
        or parsed_url.hostname == "127.0.0.1"
        or (parsed_url.hostname and parsed_url.hostname.startswith("127."))
    )

    if parsed_url.scheme != "https" and not (parsed_url.scheme == "http" and is_localhost):
        raise ValueError(f"domain must use HTTPS scheme, got: {parsed_url.scheme}")

    # Reject domains with path, query, or fragment components
    if parsed_url.path and parsed_url.path != "/":
        raise ValueError(f"domain cannot contain path components: {parsed_url.path}")
    if parsed_url.query:
        raise ValueError(f"domain cannot contain query parameters: {parsed_url.query}")
    if parsed_url.fragment:
        raise ValueError(f"domain cannot contain fragment: {parsed_url.fragment}")

    # Ensure host is present
    if not parsed_url.netloc:
        raise ValueError("domain must have a host")

    # Check for path traversal attempts in host
    if "/" in parsed_url.netloc or ".." in parsed_url.netloc:
        raise ValueError("domain host contains invalid characters")


def validate_path_segment(segment: str, name: str) -> None:
    """Validates that a path segment is safe"""
    if not segment:
        raise ValueError(f"{name} cannot be empty")

    # Reject path traversal attempts
    if ".." in segment or "/" in segment:
        raise ValueError(f"{name} contains invalid characters (path traversal detected): {segment}")

    # Reject URL-encoded path traversal (case-insensitive)
    if "%2e%2e" in segment.lower():
        raise ValueError(f"{name} contains URL-encoded path traversal: {segment}")

    # Reject null bytes
    if "\x00" in segment:
        raise ValueError(f"{name} contains null byte")
