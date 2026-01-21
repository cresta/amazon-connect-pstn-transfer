/**
 * Tests for HTTP client retry logic matching the Go implementation
 */

import { exponentialBackoff, isRetryableError, RetryHTTPClient } from "./httpclient.js";
import { Logger } from "./logger.js";

describe("RetryHTTPClient", () => {
	let logger: Logger;
	const originalFetch = globalThis.fetch;

	beforeAll(() => {
		globalThis.fetch = jest.fn() as typeof fetch;
	});

	afterAll(() => {
		globalThis.fetch = originalFetch;
	});

	beforeEach(() => {
		jest.clearAllMocks();
		(globalThis.fetch as jest.Mock).mockReset();
		logger = new Logger();
	});

	describe("isRetryableError", () => {
		it("should return true for network errors", () => {
			const error = new Error("network error");
			expect(isRetryableError(error, 0)).toBe(true);
		});

		it("should return true for 5xx status codes", () => {
			expect(isRetryableError(null, 500)).toBe(true);
			expect(isRetryableError(null, 503)).toBe(true);
			expect(isRetryableError(null, 599)).toBe(true);
		});

		it("should return true for 429 status code", () => {
			expect(isRetryableError(null, 429)).toBe(true);
		});

		it("should return true for 408 status code", () => {
			expect(isRetryableError(null, 408)).toBe(true);
		});

		it("should return false for 4xx status codes (except 429 and 408)", () => {
			expect(isRetryableError(null, 400)).toBe(false);
			expect(isRetryableError(null, 404)).toBe(false);
			expect(isRetryableError(null, 401)).toBe(false);
			expect(isRetryableError(null, 403)).toBe(false);
		});

		it("should return false for 2xx status codes", () => {
			expect(isRetryableError(null, 200)).toBe(false);
			expect(isRetryableError(null, 201)).toBe(false);
			expect(isRetryableError(null, 204)).toBe(false);
		});
	});

	describe("fetch", () => {
		it("should succeed on first attempt", async () => {
			const mockResponse = {
				status: 200,
				ok: true,
				arrayBuffer: async () => new TextEncoder().encode("success"),
			};

			(globalThis.fetch as jest.Mock).mockResolvedValueOnce(mockResponse);

			const client = new RetryHTTPClient({ logger });
			const request = new Request("https://api.example.com/test");

			const response = await client.fetch(request);

			expect(response).toBe(mockResponse);
			expect(globalThis.fetch).toHaveBeenCalledTimes(1);
		});

		it("should retry on 429 status code and eventually succeed", async () => {
			const mockSuccessResponse = {
				status: 200,
				ok: true,
				arrayBuffer: async () => new TextEncoder().encode("success"),
			};

			const mock429Response = {
				status: 429,
				ok: false,
				arrayBuffer: async () => new TextEncoder().encode("too many requests"),
			};

			(globalThis.fetch as jest.Mock)
				.mockResolvedValueOnce(mock429Response)
				.mockResolvedValueOnce(mockSuccessResponse);

			const client = new RetryHTTPClient({
				logger,
				maxRetries: 2,
				baseDelay: 10,
			});

			const request = new Request("https://api.example.com/test");

			const response = await client.fetch(request);

			expect(response.status).toBe(200);
			expect(globalThis.fetch).toHaveBeenCalledTimes(2);
		});

		it("should retry on 408 status code and eventually succeed", async () => {
			const mockSuccessResponse = {
				status: 200,
				ok: true,
				arrayBuffer: async () => new TextEncoder().encode("success"),
			};

			const mock408Response = {
				status: 408,
				ok: false,
				arrayBuffer: async () => new TextEncoder().encode("request timeout"),
			};

			(globalThis.fetch as jest.Mock)
				.mockResolvedValueOnce(mock408Response)
				.mockResolvedValueOnce(mockSuccessResponse);

			const client = new RetryHTTPClient({
				logger,
				maxRetries: 2,
				baseDelay: 10,
			});

			const request = new Request("https://api.example.com/test");

			const response = await client.fetch(request);

			expect(response.status).toBe(200);
			expect(globalThis.fetch).toHaveBeenCalledTimes(2);
		});

		it("should retry on 5xx status code and eventually succeed", async () => {
			const mockSuccessResponse = {
				status: 200,
				ok: true,
				arrayBuffer: async () => new TextEncoder().encode("success"),
			};

			const mock500Response = {
				status: 500,
				ok: false,
				arrayBuffer: async () => new TextEncoder().encode("server error"),
			};

			(globalThis.fetch as jest.Mock)
				.mockResolvedValueOnce(mock500Response)
				.mockResolvedValueOnce(mockSuccessResponse);

			const client = new RetryHTTPClient({
				logger,
				maxRetries: 2,
				baseDelay: 10,
			});

			const request = new Request("https://api.example.com/test");

			const response = await client.fetch(request);

			expect(response.status).toBe(200);
			expect(globalThis.fetch).toHaveBeenCalledTimes(2);
		});

		it("should not retry on 4xx status codes (except 429 and 408)", async () => {
			const mock400Response = {
				status: 400,
				ok: false,
				arrayBuffer: async () => new TextEncoder().encode("bad request"),
			};

			(globalThis.fetch as jest.Mock).mockResolvedValueOnce(mock400Response);

			const client = new RetryHTTPClient({ logger });
			const request = new Request("https://api.example.com/test");

			const response = await client.fetch(request);

			expect(response.status).toBe(400);
			expect(globalThis.fetch).toHaveBeenCalledTimes(1);
		});

		it("should exhaust retries when all attempts return retryable status", async () => {
			const mock429Response = {
				status: 429,
				ok: false,
				arrayBuffer: async () => new TextEncoder().encode("too many requests"),
			};

			(globalThis.fetch as jest.Mock)
				.mockResolvedValueOnce(mock429Response)
				.mockResolvedValueOnce(mock429Response)
				.mockResolvedValueOnce(mock429Response)
				.mockResolvedValueOnce(mock429Response);

			const client = new RetryHTTPClient({
				logger,
				maxRetries: 3,
				baseDelay: 10,
			});

			const request = new Request("https://api.example.com/test");

			await expect(client.fetch(request)).rejects.toThrow("request failed after 4 attempts");
			expect(globalThis.fetch).toHaveBeenCalledTimes(4);
		});

		it("should not retry on abort errors", async () => {
			const abortError = new Error("aborted");
			abortError.name = "AbortError";

			(globalThis.fetch as jest.Mock).mockRejectedValueOnce(abortError);

			const client = new RetryHTTPClient({
				logger,
				maxRetries: 3,
				baseDelay: 10,
			});

			const request = new Request("https://api.example.com/test");

			await expect(client.fetch(request)).rejects.toThrow("aborted");
			expect(globalThis.fetch).toHaveBeenCalledTimes(1);
		});
	});

	describe("exponentialBackoff", () => {
		it("should calculate exponential backoff with jitter", () => {
			const baseDelay = 100;
			const attempt0 = exponentialBackoff(0, baseDelay);
			const attempt1 = exponentialBackoff(1, baseDelay);
			const attempt2 = exponentialBackoff(2, baseDelay);

			// Base delay (2^0 = 1) with jitter (0-25%)
			expect(attempt0).toBeGreaterThanOrEqual(baseDelay);
			expect(attempt0).toBeLessThanOrEqual(baseDelay * 1.25);

			// 2x delay (2^1 = 2) with jitter
			expect(attempt1).toBeGreaterThanOrEqual(baseDelay * 2);
			expect(attempt1).toBeLessThanOrEqual(baseDelay * 2 * 1.25);

			// 4x delay (2^2 = 4) with jitter
			expect(attempt2).toBeGreaterThanOrEqual(baseDelay * 4);
			expect(attempt2).toBeLessThanOrEqual(baseDelay * 4 * 1.25);
		});
	});
});
