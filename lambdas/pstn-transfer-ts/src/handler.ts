/**
 * AWS Lambda handler for PSTN Transfer (TypeScript implementation)
 * Matches the Go implementation exactly
 */

import { DefaultOAuth2TokenFetcher, type OAuth2TokenFetcher } from "./auth.js";
import { CrestaAPIClient } from "./client.js";
import { Handlers } from "./handlers.js";
import type { AuthConfig } from "./httpclient.js";
import { type Logger, newLogger } from "./logger.js";
import {
	buildAPIDomainFromRegion,
	extractRegionFromDomain,
	getFromEventParameterOrEnv,
	parseVirtualAgentName,
	validateDomain,
	validatePathSegment,
} from "./utils.js";
import type { ConnectEvent, ConnectResponse } from "./types.js";

export interface HandlerService {
	handle(signal: AbortSignal, event: ConnectEvent): Promise<ConnectResponse>;
}

export class DefaultHandlerService implements HandlerService {
	private readonly logger: Logger;
	private readonly tokenFetcher: OAuth2TokenFetcher;

	constructor(logger?: Logger, tokenFetcher?: OAuth2TokenFetcher) {
		this.logger = logger ?? newLogger();
		this.tokenFetcher = tokenFetcher ?? new DefaultOAuth2TokenFetcher();
	}

	async handle(signal: AbortSignal, event: ConnectEvent): Promise<ConnectResponse> {
		this.logger.debugf("Received event: %+v", event);

		// Extract region first - from region parameter or apiDomain (deprecated)
		const regionParam = getFromEventParameterOrEnv(event, "region", "");
		const apiDomainParam = getFromEventParameterOrEnv(event, "apiDomain", ""); // Deprecated: use region instead

		let region: string;
		if (regionParam) {
			region = regionParam;
		} else if (apiDomainParam) {
			// Try to extract region from apiDomain, but don't fail if it doesn't match the pattern
			try {
				region = extractRegionFromDomain(apiDomainParam);
			} catch (err) {
				const errorMessage = err instanceof Error ? err.message : String(err);
				throw new Error(`could not extract region from apiDomain: ${errorMessage}`);
			}
		} else {
			throw new Error("region is required");
		}

		// Calculate apiDomain from region if not provided, otherwise use provided apiDomain
		let domain: string;
		if (apiDomainParam) {
			domain = apiDomainParam;
		} else {
			domain = buildAPIDomainFromRegion(region);
		}

		// Validate domain to prevent injection attacks
		try {
			validateDomain(domain);
		} catch (err) {
			const errorMessage = err instanceof Error ? err.message : String(err);
			throw new Error(`invalid domain: ${errorMessage}`);
		}

		const action = getFromEventParameterOrEnv(event, "action", "");
		if (!action) {
			throw new Error("action is required");
		}

		const apiKey = getFromEventParameterOrEnv(event, "apiKey", ""); // Deprecated: use oauthClientId/oauthClientSecret instead
		const oauthClientID = getFromEventParameterOrEnv(event, "oauthClientId", "");
		const oauthClientSecret = getFromEventParameterOrEnv(event, "oauthClientSecret", "");

		const virtualAgentName = getFromEventParameterOrEnv(event, "virtualAgentName", "");
		if (!virtualAgentName) {
			throw new Error("virtualAgentName is required");
		}

		let customer: string;
		let profile: string;
		let virtualAgentID: string;
		try {
			const parsed = parseVirtualAgentName(virtualAgentName);
			customer = parsed.customer;
			profile = parsed.profile;
			virtualAgentID = parsed.virtualAgentID;
		} catch (err) {
			const errorMessage = err instanceof Error ? err.message : String(err);
			this.logger.errorf("Error parsing virtual agent name: %v", errorMessage);
			throw err instanceof Error ? err : new Error(String(err));
		}

		// Validate path segments to prevent injection attacks
		try {
			validatePathSegment(customer, "customer");
			validatePathSegment(profile, "profile");
			validatePathSegment(virtualAgentID, "virtualAgentID");
		} catch (err) {
			throw err instanceof Error ? err : new Error(String(err));
		}

		// Either API key (deprecated) or OAuth 2 credentials must be provided
		let authConfig: AuthConfig;
		if (oauthClientID && oauthClientSecret) {
			// Use OAuth 2 authentication
			this.logger.infof("Using OAuth 2 authentication");
			authConfig = {
				region,
				oauthClientID,
				oauthClientSecret,
				tokenFetcher: this.tokenFetcher,
			};
		} else if (apiKey) {
			// Use API key authentication (deprecated)
			this.logger.warnf("Using API key authentication (deprecated)");
			authConfig = {
				apiKey,
			};
		} else {
			throw new Error(
				"either apiKey (deprecated) or oauthClientId/oauthClientSecret must be provided",
			);
		}

		// Get supportedDtmfChars from environment variable only, default to "0123456789*"
		const supportedDtmfChars = process.env.supportedDtmfChars || "0123456789*";

		// Create handlers with authConfig, domain, parsed components, and event
		const apiClient = new CrestaAPIClient(this.logger, authConfig);
		const handlers = new Handlers(
			this.logger,
			apiClient,
			domain,
			customer,
			profile,
			virtualAgentID,
			supportedDtmfChars,
			event,
		);

		this.logger.infof(
			"Domain: %s, Region: %s, Action: %s, Virtual Agent Name: %s",
			domain,
			region,
			action,
			virtualAgentName,
		);

		let result: ConnectResponse;
		switch (action) {
			case "get_pstn_transfer_data":
				result = await handlers.getPSTNTransferData(signal);
				break;
			case "get_handoff_data":
				result = await handlers.getHandoffData(signal);
				break;
			default:
				throw new Error(`invalid action: ${action}`);
		}

		return result;
	}
}

import type { Context } from "aws-lambda";

/**
 * Lambda handler function for Amazon Connect PSTN Transfer
 * This is the entry point for AWS Lambda
 *
 * @param event - The Lambda event (ConnectEvent)
 * @param context - The Lambda context object (optional but recommended for best practices)
 * @returns Promise resolving to ConnectResponse
 */
export const handler = async (event: ConnectEvent, context?: Context): Promise<ConnectResponse> => {
	// Log request ID for tracing if context is provided
	const logger = newLogger();
	if (context) {
		logger.debugf("Lambda Request ID: %s", context.awsRequestId);
	}

	const controller = new AbortController();
	const service = new DefaultHandlerService();
	try {
		return await service.handle(controller.signal, event);
	} catch (error) {
		// Log error and rethrow for Lambda to handle
		const errorMessage = error instanceof Error ? error.message : String(error);
		logger.errorf("Handler error: %v", errorMessage);
		if (error instanceof Error && error.stack) {
			logger.debugf("Stack trace: %v", error.stack);
		}
		throw error instanceof Error ? error : new Error(String(error));
	}
};
