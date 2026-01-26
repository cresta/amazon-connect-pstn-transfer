/**
 * AWS Secrets Manager utility for fetching OAuth credentials
 */

import { GetSecretValueCommand, SecretsManagerClient } from "@aws-sdk/client-secrets-manager";
import type { Logger } from "./logger.js";

export interface OAuthCredentials {
	oauthClientId: string;
	oauthClientSecret: string;
}

/**
 * Fetches OAuth credentials from AWS Secrets Manager
 * The secret should be a JSON object with oauthClientId and oauthClientSecret fields
 */
export async function getOAuthCredentialsFromSecretsManager(
	logger: Logger,
	secretArn: string,
): Promise<OAuthCredentials> {
	const region = extractRegionFromSecretArn(secretArn);

	// Use AWS SDK v3 client (available in Lambda runtime)
	try {
		const client = new SecretsManagerClient({ region });
		const command = new GetSecretValueCommand({ SecretId: secretArn });

		const response = await client.send(command);

		if (!response.SecretString) {
			throw new Error("secret value is empty or not a string");
		}

		const secretValue = JSON.parse(response.SecretString) as unknown;

		if (
			!secretValue ||
			typeof secretValue !== "object" ||
			!("oauthClientId" in secretValue) ||
			!("oauthClientSecret" in secretValue)
		) {
			throw new Error("secret must contain oauthClientId and oauthClientSecret fields");
		}

		const oauthClientId = String(secretValue.oauthClientId);
		const oauthClientSecret = String(secretValue.oauthClientSecret);

		if (!oauthClientId || !oauthClientSecret) {
			throw new Error("oauthClientId and oauthClientSecret must be non-empty strings");
		}

		logger.debugf("Successfully retrieved OAuth credentials from Secrets Manager");

		return {
			oauthClientId,
			oauthClientSecret,
		};
	} catch (error) {
		const errorMessage = error instanceof Error ? error.message : String(error);
		logger.errorf("Failed to retrieve credentials from Secrets Manager: %v", errorMessage);
		throw new Error(`failed to retrieve OAuth credentials from Secrets Manager: ${errorMessage}`);
	}
}

/**
 * Extracts AWS region from Secrets Manager ARN
 * Format: arn:aws:secretsmanager:REGION:ACCOUNT:secret:NAME
 */
function extractRegionFromSecretArn(arn: string): string {
	const parts = arn.split(":");
	if (
		parts.length < 4 ||
		parts[0] !== "arn" ||
		parts[1] !== "aws" ||
		parts[2] !== "secretsmanager"
	) {
		throw new Error(`invalid Secrets Manager ARN format: ${arn}`);
	}
	return parts[3];
}
