/**
 * Cresta API client matching the Go implementation
 */

import type { AuthConfig } from "./httpclient.js";
import { type HTTPClient, RetryHTTPClient } from "./httpclient.js";
import type { Logger } from "./logger.js";

export class CrestaAPIClient {
	private logger: Logger;
	private client: HTTPClient;

	constructor(logger: Logger, authConfig: AuthConfig) {
		if (!authConfig) {
			throw new Error("authConfig is required for CrestaAPIClient");
		}
		this.client = new RetryHTTPClient({
			logger,
			authConfig,
		});
		this.logger = logger;
	}

	async makeRequest(
		signal: AbortSignal,
		method: string,
		url: string,
		payload: unknown,
	): Promise<Uint8Array> {
		const jsonData = JSON.stringify(payload);
		this.logger.debugf("Sending request to %s with payload: %s", url, jsonData);

		const request = new Request(url, {
			method,
			headers: {
				"Content-Type": "application/json",
			},
			body: jsonData,
			signal,
		});

		const response = await this.client.fetch(request);

		if (response.status !== 200) {
			const body = await response.text();
			throw new Error(`request returned non-200 status: ${response.status}, body: ${body}`);
		}

		const arrayBuffer = await response.arrayBuffer();
		return new Uint8Array(arrayBuffer);
	}
}
