"""
Tests for Secrets Manager matching Go and TypeScript test structure
"""

import json
from unittest.mock import MagicMock, patch

import pytest

from src.logger import Logger
from src.secretsmanager import (
    _extract_region_from_secret_arn,
    get_oauth_credentials_from_secrets_manager,
)


class TestExtractRegionFromSecretArn:
    """Tests for _extract_region_from_secret_arn"""

    def test_extracts_region(self):
        """Should extract region from valid ARN"""
        arn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:my-secret"
        result = _extract_region_from_secret_arn(arn)
        assert result == "us-west-2"

    def test_raises_on_invalid_format(self):
        """Should raise error on invalid ARN format"""
        with pytest.raises(ValueError, match="invalid Secrets Manager ARN"):
            _extract_region_from_secret_arn("invalid-arn")

    def test_raises_on_wrong_service(self):
        """Should raise error when service is not secretsmanager"""
        arn = "arn:aws:s3:us-west-2:123456789012:bucket:my-bucket"
        with pytest.raises(ValueError, match="invalid Secrets Manager ARN"):
            _extract_region_from_secret_arn(arn)


class TestGetOAuthCredentialsFromSecretsManager:
    """Tests for get_oauth_credentials_from_secrets_manager"""

    @patch("boto3.client")
    def test_retrieves_credentials(self, mock_boto_client):
        """Should retrieve credentials from Secrets Manager"""
        mock_client = MagicMock()
        mock_client.get_secret_value.return_value = {
            "SecretString": json.dumps(
                {
                    "oauthClientId": "test-client-id",
                    "oauthClientSecret": "test-client-secret",
                }
            )
        }
        mock_boto_client.return_value = mock_client

        logger = Logger()
        arn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:my-secret"

        result = get_oauth_credentials_from_secrets_manager(logger, arn)

        assert result.oauth_client_id == "test-client-id"
        assert result.oauth_client_secret == "test-client-secret"

        mock_boto_client.assert_called_with("secretsmanager", region_name="us-west-2")
        mock_client.get_secret_value.assert_called_with(SecretId=arn)

    @patch("boto3.client")
    def test_raises_on_empty_secret(self, mock_boto_client):
        """Should raise error when secret is empty"""
        mock_client = MagicMock()
        mock_client.get_secret_value.return_value = {"SecretString": ""}
        mock_boto_client.return_value = mock_client

        logger = Logger()
        arn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:my-secret"

        with pytest.raises(ValueError, match="failed to retrieve OAuth credentials"):
            get_oauth_credentials_from_secrets_manager(logger, arn)

    @patch("boto3.client")
    def test_raises_on_missing_fields(self, mock_boto_client):
        """Should raise error when required fields are missing"""
        mock_client = MagicMock()
        mock_client.get_secret_value.return_value = {
            "SecretString": json.dumps(
                {
                    "oauthClientId": "test-client-id",
                    # Missing oauthClientSecret
                }
            )
        }
        mock_boto_client.return_value = mock_client

        logger = Logger()
        arn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:my-secret"

        with pytest.raises(ValueError, match="must contain oauthClientId and oauthClientSecret"):
            get_oauth_credentials_from_secrets_manager(logger, arn)

    @patch("boto3.client")
    def test_raises_on_empty_credential_values(self, mock_boto_client):
        """Should raise error when credential values are empty"""
        mock_client = MagicMock()
        mock_client.get_secret_value.return_value = {
            "SecretString": json.dumps(
                {
                    "oauthClientId": "",
                    "oauthClientSecret": "test-secret",
                }
            )
        }
        mock_boto_client.return_value = mock_client

        logger = Logger()
        arn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:my-secret"

        with pytest.raises(ValueError, match="non-empty strings"):
            get_oauth_credentials_from_secrets_manager(logger, arn)

    @patch("boto3.client")
    def test_raises_on_non_string_values(self, mock_boto_client):
        """Should raise error when credential values are not strings"""
        mock_client = MagicMock()
        mock_client.get_secret_value.return_value = {
            "SecretString": json.dumps(
                {
                    "oauthClientId": 12345,  # Number instead of string
                    "oauthClientSecret": "test-secret",
                }
            )
        }
        mock_boto_client.return_value = mock_client

        logger = Logger()
        arn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:my-secret"

        with pytest.raises(ValueError, match="non-empty strings"):
            get_oauth_credentials_from_secrets_manager(logger, arn)
