"""
AWS Lambda handler for PSTN Transfer (Python implementation)
Matches the Go and TypeScript implementations exactly
"""

import json
import os
import sys
from typing import Any

from .auth import DefaultOAuth2TokenFetcher, OAuth2TokenFetcher
from .client import CrestaAPIClient
from .handlers import Handlers
from .httpclient import AuthConfig
from .logger import Logger, new_logger
from .secretsmanager import get_oauth_credentials_from_secrets_manager
from .types import ConnectEvent, ConnectResponse
from .utils import (
    build_api_domain_from_region,
    extract_region_from_domain,
    get_auth_region,
    get_from_event_parameter_or_env,
    parse_virtual_agent_name,
    validate_domain,
    validate_path_segment,
)


class DefaultHandlerService:
    """Handler service for processing Lambda events"""

    def __init__(
        self,
        logger: Logger | None = None,
        token_fetcher: OAuth2TokenFetcher | None = None,
    ):
        self._logger = logger or new_logger()
        self._token_fetcher = token_fetcher or DefaultOAuth2TokenFetcher()

    def handle(self, event: ConnectEvent) -> ConnectResponse:
        """Process the Lambda event and return a response"""
        self._logger.debugf("Received event: %+v", event.Details)

        # Extract region first - from region parameter or apiDomain
        region_param = get_from_event_parameter_or_env(event, "region", "")
        api_domain_param = get_from_event_parameter_or_env(event, "apiDomain", "")
        auth_domain_param = get_from_event_parameter_or_env(event, "authDomain", "")

        # Get OAuth credentials early to check if OAuth will be used for validation
        oauth_secret_arn_check = get_from_event_parameter_or_env(event, "oauthSecretArn", "")
        oauth_client_id_check = get_from_event_parameter_or_env(event, "oauthClientId", "")
        oauth_client_secret_check = get_from_event_parameter_or_env(event, "oauthClientSecret", "")

        # Validate that apiDomain and authDomain are used together (only when OAuth is used)
        # If using API key, apiDomain alone is fine
        will_use_oauth = oauth_secret_arn_check != "" or (
            oauth_client_id_check != "" and oauth_client_secret_check != ""
        )
        if will_use_oauth:
            if (api_domain_param and not auth_domain_param) or (
                not api_domain_param and auth_domain_param
            ):
                raise ValueError("apiDomain and authDomain must be provided together")

        # Determine region
        region: str
        if region_param:
            region = region_param
        elif api_domain_param:
            # Try to extract region from apiDomain
            try:
                region = extract_region_from_domain(api_domain_param)
            except Exception as e:
                raise ValueError(f"could not extract region from apiDomain: {e}") from e
        else:
            raise ValueError("region is required")

        # Calculate apiDomain from region if not provided, otherwise use provided apiDomain
        domain: str
        if api_domain_param:
            # Add https:// prefix if not present
            if not api_domain_param.startswith("http://") and not api_domain_param.startswith(
                "https://"
            ):
                domain = "https://" + api_domain_param
            else:
                domain = api_domain_param
        else:
            domain = build_api_domain_from_region(region)

        # Validate domain to prevent injection attacks
        try:
            validate_domain(domain)
        except Exception as e:
            raise ValueError(f"invalid domain: {e}") from e

        # Process authDomain if provided
        auth_domain: str = ""
        if auth_domain_param:
            # Add https:// prefix if not present
            if not auth_domain_param.startswith("http://") and not auth_domain_param.startswith(
                "https://"
            ):
                auth_domain = "https://" + auth_domain_param
            else:
                auth_domain = auth_domain_param
            # Validate authDomain to prevent injection attacks
            try:
                validate_domain(auth_domain)
            except Exception as e:
                raise ValueError(f"invalid authDomain: {e}") from e

        action = get_from_event_parameter_or_env(event, "action", "")
        if not action:
            raise ValueError("action is required")

        api_key = get_from_event_parameter_or_env(event, "apiKey", "")  # Deprecated
        oauth_secret_arn = get_from_event_parameter_or_env(event, "oauthSecretArn", "")
        oauth_client_id = get_from_event_parameter_or_env(event, "oauthClientId", "")
        oauth_client_secret = get_from_event_parameter_or_env(event, "oauthClientSecret", "")

        virtual_agent_name = get_from_event_parameter_or_env(event, "virtualAgentName", "")
        if not virtual_agent_name:
            raise ValueError("virtualAgentName is required")

        try:
            customer, profile, virtual_agent_id = parse_virtual_agent_name(virtual_agent_name)
        except Exception as e:
            self._logger.errorf("Error parsing virtual agent name: %v", str(e))
            raise

        # Validate path segments to prevent injection attacks
        validate_path_segment(customer, "customer")
        validate_path_segment(profile, "profile")
        validate_path_segment(virtual_agent_id, "virtualAgentID")

        # Either API key (deprecated) or OAuth 2 credentials must be provided
        # Priority: Secrets Manager > Environment/Parameters > API Key (deprecated)
        auth_config: AuthConfig
        resolved_oauth_client_id = oauth_client_id
        resolved_oauth_client_secret = oauth_client_secret

        # Try to fetch from Secrets Manager if ARN is provided
        if oauth_secret_arn:
            self._logger.infof(
                "Fetching OAuth credentials from Secrets Manager: %s", oauth_secret_arn
            )
            try:
                credentials = get_oauth_credentials_from_secrets_manager(
                    self._logger, oauth_secret_arn
                )
                resolved_oauth_client_id = credentials.oauth_client_id
                resolved_oauth_client_secret = credentials.oauth_client_secret
                self._logger.infof("Successfully retrieved OAuth credentials from Secrets Manager")
            except Exception as e:
                self._logger.errorf(
                    "Failed to retrieve credentials from Secrets Manager: %v", str(e)
                )
                raise ValueError(
                    f"failed to retrieve OAuth credentials from Secrets Manager: {e}"
                ) from e

        if resolved_oauth_client_id and resolved_oauth_client_secret:
            # Use OAuth 2 authentication
            # Determine auth domain: use provided authDomain, or build from region
            final_auth_domain: str
            if auth_domain:
                final_auth_domain = auth_domain
            else:
                # Build auth domain from region
                auth_region = get_auth_region(region)
                final_auth_domain = f"https://auth.{auth_region}.cresta.ai"

            self._logger.infof("Using OAuth 2 authentication")
            auth_config = AuthConfig(
                auth_domain=final_auth_domain,
                oauth_client_id=resolved_oauth_client_id,
                oauth_client_secret=resolved_oauth_client_secret,
                token_fetcher=self._token_fetcher,
            )
        elif api_key:
            # Use API key authentication (deprecated)
            self._logger.warnf("Using API key authentication (deprecated)")
            auth_config = AuthConfig(api_key=api_key)
        else:
            raise ValueError(
                "either apiKey (deprecated), oauthClientId/oauthClientSecret, "
                "or oauthSecretArn must be provided"
            )

        # Get supportedDtmfChars from environment variable only, default to "0123456789*"
        supported_dtmf_chars = os.environ.get("supportedDtmfChars", "0123456789*")

        # Create handlers with authConfig, domain, parsed components, and event
        api_client = CrestaAPIClient(self._logger, auth_config)
        handlers = Handlers(
            self._logger,
            api_client,
            domain,
            customer,
            profile,
            virtual_agent_id,
            supported_dtmf_chars,
            event,
        )

        self._logger.infof(
            "Domain: %s, Region: %s, Action: %s, Virtual Agent Name: %s",
            domain,
            region,
            action,
            virtual_agent_name,
        )

        # Execute action
        result: ConnectResponse
        if action == "get_pstn_transfer_data":
            result = handlers.get_pstn_transfer_data()
        elif action == "get_handoff_data":
            result = handlers.get_handoff_data()
        else:
            raise ValueError(f"invalid action: {action}")

        return result


def handler(event: dict[str, Any], context: Any = None) -> ConnectResponse:
    """
    Lambda handler function for Amazon Connect PSTN Transfer.
    This is the entry point for AWS Lambda.

    Args:
        event: The Lambda event (ConnectEvent structure)
        context: The Lambda context object (optional)

    Returns:
        ConnectResponse dictionary
    """
    logger = new_logger()

    # Log request ID for tracing if context is provided
    if context and hasattr(context, "aws_request_id"):
        logger.debugf("Lambda Request ID: %s", context.aws_request_id)

    # Convert dict event to ConnectEvent
    connect_event = ConnectEvent(
        Details=event.get("Details", {}),
        Name=event.get("Name"),
    )

    service = DefaultHandlerService()
    try:
        return service.handle(connect_event)
    except Exception as error:
        # Log error and rethrow for Lambda to handle
        error_message = str(error)
        logger.errorf("Handler error: %v", error_message)
        raise


# Support test mode: if run directly with --test flag, read from file or stdin and write to stdout
if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--test":
        try:
            # Read from file if provided, otherwise from stdin
            if len(sys.argv) > 2:
                with open(sys.argv[2]) as f:
                    event_data = json.load(f)
            else:
                event_data = json.load(sys.stdin)
            result = handler(event_data)
            json.dump(result, sys.stdout)
            sys.stdout.write("\n")
        except Exception as e:
            sys.stderr.write(f"Error: {e}\n")
            sys.exit(1)
