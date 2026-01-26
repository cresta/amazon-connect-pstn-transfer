/**
 * Tests for Secrets Manager utility
 */

import { SecretsManagerClient } from "@aws-sdk/client-secrets-manager";
import { Logger } from "./logger.js";
import { getOAuthCredentialsFromSecretsManager } from "./secretsmanager.js";

// Mock AWS SDK
jest.mock("@aws-sdk/client-secrets-manager", () => {
	const mockSend = jest.fn();
	return {
		SecretsManagerClient: jest.fn().mockImplementation(() => ({
			send: mockSend,
		})),
		GetSecretValueCommand: jest.fn().mockImplementation((input) => input),
		__mockSend: mockSend,
	};
});

describe("getOAuthCredentialsFromSecretsManager", () => {
	let logger: Logger;
	let mockSend: jest.Mock;

	beforeEach(() => {
		jest.clearAllMocks();
		logger = new Logger();
		// Get the mock send function from the mocked module
		// eslint-disable-next-line @typescript-eslint/no-require-imports
		const secretsManagerModule = require("@aws-sdk/client-secrets-manager");
		mockSend = secretsManagerModule.__mockSend;
	});

	it("should successfully retrieve OAuth credentials from Secrets Manager", async () => {
		const secretArn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret";
		const mockSecretValue = {
			oauthClientId: "test-client-id",
			oauthClientSecret: "test-client-secret",
		};

		mockSend.mockResolvedValueOnce({
			SecretString: JSON.stringify(mockSecretValue),
		});

		const result = await getOAuthCredentialsFromSecretsManager(logger, secretArn);

		expect(result).toEqual({
			oauthClientId: "test-client-id",
			oauthClientSecret: "test-client-secret",
		});
		expect(mockSend).toHaveBeenCalledTimes(1);
	});

	it("should throw error when secret value is empty", async () => {
		const secretArn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret";

		mockSend.mockResolvedValueOnce({
			SecretString: null,
		});

		await expect(getOAuthCredentialsFromSecretsManager(logger, secretArn)).rejects.toThrow(
			"secret value is empty or not a string",
		);
	});

	it("should throw error when secret is missing oauthClientId", async () => {
		const secretArn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret";
		const mockSecretValue = {
			oauthClientSecret: "test-client-secret",
		};

		mockSend.mockResolvedValueOnce({
			SecretString: JSON.stringify(mockSecretValue),
		});

		await expect(getOAuthCredentialsFromSecretsManager(logger, secretArn)).rejects.toThrow(
			"secret must contain oauthClientId and oauthClientSecret fields",
		);
	});

	it("should throw error when secret is missing oauthClientSecret", async () => {
		const secretArn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret";
		const mockSecretValue = {
			oauthClientId: "test-client-id",
		};

		mockSend.mockResolvedValueOnce({
			SecretString: JSON.stringify(mockSecretValue),
		});

		await expect(getOAuthCredentialsFromSecretsManager(logger, secretArn)).rejects.toThrow(
			"secret must contain oauthClientId and oauthClientSecret fields",
		);
	});

	it("should throw error when oauthClientId is empty string", async () => {
		const secretArn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret";
		const mockSecretValue = {
			oauthClientId: "",
			oauthClientSecret: "test-client-secret",
		};

		mockSend.mockResolvedValueOnce({
			SecretString: JSON.stringify(mockSecretValue),
		});

		await expect(getOAuthCredentialsFromSecretsManager(logger, secretArn)).rejects.toThrow(
			"secret must contain oauthClientId and oauthClientSecret as non-empty strings",
		);
	});

	it("should throw error when oauthClientSecret is empty string", async () => {
		const secretArn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret";
		const mockSecretValue = {
			oauthClientId: "test-client-id",
			oauthClientSecret: "",
		};

		mockSend.mockResolvedValueOnce({
			SecretString: JSON.stringify(mockSecretValue),
		});

		await expect(getOAuthCredentialsFromSecretsManager(logger, secretArn)).rejects.toThrow(
			"secret must contain oauthClientId and oauthClientSecret as non-empty strings",
		);
	});

	it("should throw error when secret value is invalid JSON", async () => {
		const secretArn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret";

		mockSend.mockResolvedValueOnce({
			SecretString: "invalid json{",
		});

		await expect(getOAuthCredentialsFromSecretsManager(logger, secretArn)).rejects.toThrow(
			"failed to retrieve OAuth credentials from Secrets Manager",
		);
	});

	it("should throw error when AWS SDK call fails", async () => {
		const secretArn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret";

		mockSend.mockRejectedValueOnce(new Error("AWS SDK error"));

		await expect(getOAuthCredentialsFromSecretsManager(logger, secretArn)).rejects.toThrow(
			"failed to retrieve OAuth credentials from Secrets Manager",
		);
	});

	it("should extract region correctly from ARN", async () => {
		const secretArn = "arn:aws:secretsmanager:eu-west-1:123456789012:secret:test-secret";
		const mockSecretValue = {
			oauthClientId: "test-client-id",
			oauthClientSecret: "test-client-secret",
		};

		mockSend.mockResolvedValueOnce({
			SecretString: JSON.stringify(mockSecretValue),
		});

		await getOAuthCredentialsFromSecretsManager(logger, secretArn);

		// Verify that the client was created with the correct region
		expect(SecretsManagerClient).toHaveBeenCalledWith({ region: "eu-west-1" });
	});

	it("should throw error for invalid ARN format", async () => {
		const invalidArn = "invalid-arn";

		await expect(getOAuthCredentialsFromSecretsManager(logger, invalidArn)).rejects.toThrow(
			"failed to retrieve OAuth credentials from Secrets Manager",
		);
		await expect(getOAuthCredentialsFromSecretsManager(logger, invalidArn)).rejects.toThrow(
			"invalid Secrets Manager ARN format",
		);
	});

	it("should throw error for ARN with wrong service", async () => {
		const invalidArn = "arn:aws:s3:us-west-2:123456789012:bucket:test-bucket";

		await expect(getOAuthCredentialsFromSecretsManager(logger, invalidArn)).rejects.toThrow(
			"failed to retrieve OAuth credentials from Secrets Manager",
		);
		await expect(getOAuthCredentialsFromSecretsManager(logger, invalidArn)).rejects.toThrow(
			"invalid Secrets Manager ARN format",
		);
	});

	it("should reject numeric oauthClientId", async () => {
		const secretArn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret";
		const mockSecretValue = {
			oauthClientId: 12345, // numeric value - should be rejected
			oauthClientSecret: "test-client-secret",
		};

		mockSend.mockResolvedValueOnce({
			SecretString: JSON.stringify(mockSecretValue),
		});

		await expect(getOAuthCredentialsFromSecretsManager(logger, secretArn)).rejects.toThrow(
			"secret must contain oauthClientId and oauthClientSecret as non-empty strings",
		);
	});

	it("should reject numeric oauthClientSecret", async () => {
		const secretArn = "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret";
		const mockSecretValue = {
			oauthClientId: "test-client-id",
			oauthClientSecret: 67890, // numeric value - should be rejected
		};

		mockSend.mockResolvedValueOnce({
			SecretString: JSON.stringify(mockSecretValue),
		});

		await expect(getOAuthCredentialsFromSecretsManager(logger, secretArn)).rejects.toThrow(
			"secret must contain oauthClientId and oauthClientSecret as non-empty strings",
		);
	});
});
