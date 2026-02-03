"""
AWS Secrets Manager utility for fetching OAuth credentials
"""

import json
from typing import TYPE_CHECKING

import boto3

if TYPE_CHECKING:
    from .logger import Logger

from .types import OAuthCredentials


def get_oauth_credentials_from_secrets_manager(logger: Logger, secret_arn: str) -> OAuthCredentials:
    """
    Fetches OAuth credentials from AWS Secrets Manager.
    The secret should be a JSON object with oauthClientId and oauthClientSecret fields.
    """
    try:
        region = _extract_region_from_secret_arn(secret_arn)
        client = boto3.client("secretsmanager", region_name=region)

        response = client.get_secret_value(SecretId=secret_arn)

        if "SecretString" not in response or not response["SecretString"]:
            raise ValueError("secret value is empty or not a string")

        secret_value = json.loads(response["SecretString"])

        if not isinstance(secret_value, dict):
            raise ValueError("secret must be a JSON object")

        if "oauthClientId" not in secret_value or "oauthClientSecret" not in secret_value:
            raise ValueError("secret must contain oauthClientId and oauthClientSecret fields")

        oauth_client_id = secret_value["oauthClientId"]
        oauth_client_secret = secret_value["oauthClientSecret"]

        # Validate types - must be strings (reject numeric/other types)
        if (
            not isinstance(oauth_client_id, str)
            or not isinstance(oauth_client_secret, str)
            or oauth_client_id == ""
            or oauth_client_secret == ""
        ):
            raise ValueError(
                "secret must contain oauthClientId and oauthClientSecret as non-empty strings"
            )

        logger.debugf("Successfully retrieved OAuth credentials from Secrets Manager")

        return OAuthCredentials(
            oauth_client_id=oauth_client_id,
            oauth_client_secret=oauth_client_secret,
        )

    except Exception as e:
        error_message = str(e)
        logger.errorf("Failed to retrieve credentials from Secrets Manager: %v", error_message)
        raise ValueError(
            f"failed to retrieve OAuth credentials from Secrets Manager: {error_message}"
        ) from e


def _extract_region_from_secret_arn(arn: str) -> str:
    """
    Extracts AWS region from Secrets Manager ARN.
    Format: arn:aws:secretsmanager:REGION:ACCOUNT:secret:NAME
    """
    parts = arn.split(":")
    if len(parts) < 4 or parts[0] != "arn" or parts[1] != "aws" or parts[2] != "secretsmanager":
        raise ValueError(f"invalid Secrets Manager ARN format: {arn}")
    return parts[3]
