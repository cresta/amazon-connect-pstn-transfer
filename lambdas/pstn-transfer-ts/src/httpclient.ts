/**
 * HTTP client with retry logic matching the Go implementation
 */

import type { OAuth2TokenFetcher } from "./auth.js";
import type { Logger } from "./logger.js";
import { getDurationFromEnv, getIntFromEnv } from "./utils.js";

const HTTP_MAX_RETRIES = getIntFromEnv("HTTP_MAX_RETRIES", 3);
const HTTP_RETRY_BASE_DELAY = getDurationFromEnv("HTTP_RETRY_BASE_DELAY", 100);
const HTTP_CLIENT_TIMEOUT = getDurationFromEnv("HTTP_CLIENT_TIMEOUT", 10000);

export interface HTTPClient {
	fetch(request: Request): Promise<Response>;
}

export interface AuthConfig {
	apiKey?: string; // Deprecated
	authDomain?: string; // Auth domain without path (e.g., "https://auth.us-west-2-prod.cresta.ai")
	oauthClientID?: string;
	oauthClientSecret?: string;
	tokenFetcher?: OAuth2TokenFetcher;
}

/**
 * Checks if an error is an abort/cancellation error
 */
function isAbortError(err: unknown): boolean {
	if (!(err instanceof Error)) {
		return false;
	}
	return (
		err.name === "AbortError" ||
		err.name === "AbortedError" ||
		/aborted|cancelled|canceled/i.test(err.message)
	);
}

/**
 * Creates a timeout signal that combines the parent signal with a timeout
 * Returns both the signal and a cleanup function
 */
function withTimeoutSignal(
	parent: AbortSignal | null | undefined,
	timeoutMs: number,
): { signal: AbortSignal; cleanup: () => void } {
	const controller = new AbortController();
	let timeoutId: ReturnType<typeof setTimeout> | null = null;
	let abortListener: (() => void) | null = null;

	// Set up timeout
	timeoutId = setTimeout(() => {
		controller.abort(new Error(`request timeout after ${timeoutMs}ms`));
	}, timeoutMs);

	// Listen to parent signal if provided
	if (parent) {
		if (parent.aborted) {
			controller.abort(parent.reason);
		} else {
			abortListener = () => {
				controller.abort(parent.reason);
			};
			parent.addEventListener("abort", abortListener, { once: true });
		}
	}

	const cleanup = () => {
		if (timeoutId !== null) {
			clearTimeout(timeoutId);
			timeoutId = null;
		}
		if (parent && abortListener) {
			parent.removeEventListener("abort", abortListener);
			abortListener = null;
		}
	};

	return { signal: controller.signal, cleanup };
}

export class RetryHTTPClient implements HTTPClient {
	private logger?: Logger;
	private authConfig?: AuthConfig;
	private maxRetries: number;
	private baseDelay: number;

	constructor(options?: {
		logger?: Logger;
		authConfig?: AuthConfig;
		maxRetries?: number;
		baseDelay?: number;
	}) {
		this.logger = options?.logger;
		this.authConfig = options?.authConfig;
		this.maxRetries = options?.maxRetries ?? HTTP_MAX_RETRIES;
		this.baseDelay = options?.baseDelay ?? HTTP_RETRY_BASE_DELAY;
	}

	async fetch(request: Request): Promise<Response> {
		// Check if AbortSignal is already aborted
		if (request.signal?.aborted) {
			throw new Error(`context cancelled: ${request.signal.reason}`);
		}

		// Read request body into memory so we can recreate it for retries
		let bodyBytes: ArrayBuffer | null = null;
		if (request.body) {
			// Clone the request to read body without consuming the original
			const clonedRequest = request.clone();
			bodyBytes = await clonedRequest.arrayBuffer();
		}

		let lastErr: Error | null = null;
		let lastResp: Response | null = null;

		for (let attempt = 0; attempt <= this.maxRetries; attempt++) {
			if (attempt > 0) {
				// Check if AbortSignal is aborted before retrying
				if (request.signal?.aborted) {
					throw new Error(`context cancelled: ${request.signal.reason}`);
				}

				const delay = exponentialBackoff(attempt - 1, this.baseDelay);
				if (this.logger) {
					this.logger.debugf(
						"Retrying request to %s (attempt %d/%d) after %dms",
						request.url,
						attempt + 1,
						this.maxRetries + 1,
						delay,
					);
				}

				await new Promise((resolve) => setTimeout(resolve, delay));
			}

			// Recreate request preserving all original fields, only overriding body
			// Use new Request(request, {...}) to preserve: credentials, mode, cache, redirect, etc.
			const retryRequest = new Request(request, {
				body: bodyBytes ?? null,
				// Don't set signal here - we'll use the combined timeout signal
			});

			// Add authentication header if configured
			// Use retryRequest (not original request) for consistency
			if (this.authConfig) {
				try {
					const authHeader = await this.getAuthHeader(retryRequest);
					if (authHeader && !retryRequest.headers.get("Authorization")) {
						retryRequest.headers.set("Authorization", authHeader);
					}
				} catch (authErr) {
					// Auth failures should not be retried - fail fast
					const authError = authErr instanceof Error ? authErr : new Error(String(authErr));
					throw new Error(`error getting auth header: ${authError.message}`);
				}
			}

			// Create combined timeout signal with proper cleanup
			const { signal: timeoutSignal, cleanup } = withTimeoutSignal(
				request.signal ?? null,
				HTTP_CLIENT_TIMEOUT,
			);

			try {
				const response = await fetch(retryRequest, {
					signal: timeoutSignal,
				});

				// Check if status code is retryable
				if (!isRetryableError(null, response.status)) {
					// Non-retryable, return immediately
					return response;
				}

				// Retryable status code - consume body and prepare for retry
				// Consume body to release resources
				try {
					await response.arrayBuffer();
				} catch (bodyErr) {
					// If we can't consume the body, log but continue
					if (this.logger) {
						this.logger.debugf(
							"Error consuming response body: %v",
							bodyErr instanceof Error ? bodyErr.message : String(bodyErr),
						);
					}
				}

				// Store response and error for final error message
				lastResp = response;
				lastErr = new Error(`request returned retryable status: ${response.status}`);
			} catch (err) {
				const error = err instanceof Error ? err : new Error(String(err));

				// Don't retry on abort/cancellation errors - fail immediately
				if (isAbortError(error)) {
					throw error;
				}

				// Check if this error should be retried
				if (!isRetryableError(error, 0)) {
					// Non-retryable error (e.g., programming errors, auth failures)
					throw new Error(`error making HTTP request: ${error.message}`);
				}

				// Retryable error - store and continue
				lastErr = error;
			} finally {
				// Always cleanup timeout and event listeners
				cleanup();
			}
		}

		// All retries exhausted
		if (lastErr) {
			throw new Error(`request failed after ${this.maxRetries + 1} attempts: ${lastErr.message}`);
		}
		if (lastResp) {
			throw new Error(
				`request failed after ${this.maxRetries + 1} attempts with status: ${lastResp.status}`,
			);
		}
		throw new Error(`request failed after ${this.maxRetries + 1} attempts`);
	}

	private async getAuthHeader(request: Request): Promise<string> {
		if (!this.authConfig) {
			throw new Error("authConfig is required");
		}

		// OAuth 2 authentication takes precedence
		if (this.authConfig.oauthClientID && this.authConfig.oauthClientSecret) {
			if (!this.authConfig.tokenFetcher) {
				throw new Error("tokenFetcher is required for OAuth authentication");
			}
			if (!this.authConfig.authDomain) {
				throw new Error("authDomain is required for OAuth authentication");
			}

			const token = await this.authConfig.tokenFetcher.getToken(
				request.signal ?? new AbortController().signal,
				this.authConfig.authDomain,
				this.authConfig.oauthClientID,
				this.authConfig.oauthClientSecret,
			);
			return `Bearer ${token}`;
		}

		// Fall back to API key authentication (deprecated)
		if (this.authConfig.apiKey) {
			return `ApiKey ${this.authConfig.apiKey}`;
		}

		throw new Error("no authentication configured");
	}
}

/**
 * isRetryableError determines if an error or status code should trigger a retry
 * Matches the Go implementation: retries network errors (err != nil), 5xx status codes,
 * 429 (Too Many Requests), and 408 (Request Timeout)
 */
export function isRetryableError(err: Error | null, statusCode: number): boolean {
	if (err) {
		// Network errors are retryable (but abort errors are filtered out before this)
		return true;
	}
	// Retry on 5xx server errors, 429 (Too Many Requests), and 408 (Request Timeout)
	return (statusCode >= 500 && statusCode < 600) || statusCode === 429 || statusCode === 408;
}

/**
 * exponentialBackoff calculates the delay for the given attempt with jitter
 */
export function exponentialBackoff(attempt: number, baseDelayMs: number): number {
	const delay = 2 ** attempt * baseDelayMs;
	// Add jitter: random value between 0 and 25% of delay
	const jitter = Math.random() * 0.25 * delay;
	return delay + jitter;
}
