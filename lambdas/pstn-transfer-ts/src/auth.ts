/**
 * OAuth 2 authentication with token caching matching the Go implementation
 */

import type { HTTPClient } from "./httpclient.js";
import { RetryHTTPClient } from "./httpclient.js";
import { type Logger, newLogger } from "./logger.js";

interface CacheEntry {
	token: string;
	expiresAt: Date;
}

class TokenCache {
	private cache: Map<string, CacheEntry> = new Map();

	private cacheKey(region: string, clientID: string): string {
		return `pstn-transfer:tokencache:${region}:${clientID}`;
	}

	getToken(region: string, clientID: string): string | null {
		const key = this.cacheKey(region, clientID);
		const entry = this.cache.get(key);
		if (entry?.token && new Date() < entry.expiresAt) {
			return entry.token;
		}
		return null;
	}

	setToken(region: string, clientID: string, token: string, expiresInSeconds: number): void {
		const key = this.cacheKey(region, clientID);
		// Skip caching for tokens that are too short-lived (<= 300 seconds)
		// to avoid setting an expiresAt in the past
		if (expiresInSeconds <= 300) {
			return;
		}
		// Subtract 5 minute buffer for safety
		const expiresAt = new Date(Date.now() + (expiresInSeconds - 5 * 60) * 1000);
		this.cache.set(key, { token, expiresAt });
	}

	clearToken(region: string, clientID: string): void {
		const key = this.cacheKey(region, clientID);
		this.cache.delete(key);
	}
}

const tokenCache = new TokenCache();

export interface OAuth2TokenFetcher {
	getToken(
		signal: AbortSignal,
		region: string,
		clientID: string,
		clientSecret: string,
	): Promise<string>;
}

export class DefaultOAuth2TokenFetcher implements OAuth2TokenFetcher {
	private client: HTTPClient;
	private logger: Logger;

	constructor(client?: HTTPClient, logger?: Logger) {
		this.logger = logger ?? newLogger();
		this.client =
			client ??
			new RetryHTTPClient({
				logger: this.logger,
			});
	}

	async getToken(
		signal: AbortSignal,
		region: string,
		clientID: string,
		clientSecret: string,
	): Promise<string> {
		// Check cache first
		const cachedToken = tokenCache.getToken(region, clientID);
		if (cachedToken) {
			return cachedToken;
		}

		// Construct token endpoint URL using the same region
		// Allow override via environment variable for testing
		const tokenURL =
			process.env.AUTH_ENDPOINT_OVERRIDE ||
			`https://auth.${region}.cresta.ai/v1/oauth/regionalToken`;

		// Prepare JSON payload
		const payload = {
			grant_type: "client_credentials",
		};

		// Create request with Basic Auth
		const auth = Buffer.from(`${clientID}:${clientSecret}`).toString("base64");
		const request = new Request(tokenURL, {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				Authorization: `Basic ${auth}`,
			},
			body: JSON.stringify(payload),
			signal,
		});

		const response = await this.client.fetch(request);

		if (response.status !== 200) {
			const body = await response.text();
			throw new Error(`token request returned non-200 status: ${response.status}, body: ${body}`);
		}

		interface TokenResponse {
			access_token: string;
			token_type: string;
			expires_in: number;
		}

		const tokenResponse = (await response.json()) as TokenResponse;

		if (!tokenResponse.access_token) {
			throw new Error("missing access_token in token response");
		}

		// Cache the token
		if (tokenResponse.expires_in > 0) {
			tokenCache.setToken(region, clientID, tokenResponse.access_token, tokenResponse.expires_in);
		}

		return tokenResponse.access_token;
	}
}

// Export tokenCache for testing
export function getTokenCache(): TokenCache {
	return tokenCache;
}
