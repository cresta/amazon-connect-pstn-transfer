/**
 * Tests for handlers matching Go test structure
 */

import type { CrestaAPIClient } from "./client.js";
import { Handlers } from "./handlers.js";
import { Logger } from "./logger.js";
import type { FetchAIAgentHandoffResponse } from "./types.js";
import type { ConnectEvent } from "./types.js";

// Mock fetch globally
globalThis.fetch = jest.fn() as typeof fetch;

describe("Handlers", () => {
	let logger: Logger;
	let mockAPIClient: jest.Mocked<CrestaAPIClient>;

	beforeEach(() => {
		jest.clearAllMocks();
		logger = new Logger();
		mockAPIClient = {
			makeRequest: jest.fn(),
		} as unknown as jest.Mocked<CrestaAPIClient>;
	});

	describe("getPSTNTransferData", () => {
		it("should successfully make request with filtered parameters", async () => {
			const mockResponse = {
				phoneNumber: "+1234567890",
				dtmfSequence: "1234",
			};

			mockAPIClient.makeRequest.mockResolvedValueOnce(
				new TextEncoder().encode(JSON.stringify(mockResponse)),
			);

			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {
						customParam: "customValue",
						apiKey: "should-be-filtered",
						region: "should-be-filtered",
					},
				},
			};

			const handlers = new Handlers(
				logger,
				mockAPIClient as unknown as CrestaAPIClient,
				"https://api.example.com",
				"test-customer",
				"test-profile",
				"test-agent",
				"0123456789*",
				event,
			);

			const controller = new AbortController();
			const result = await handlers.getPSTNTransferData(controller.signal);

			expect(result).toBeDefined();
			expect(result.phoneNumber).toBe("+1234567890");
			expect(result.dtmfSequence).toBe("1234");

			// Verify the request was made with correct parameters
			expect(mockAPIClient.makeRequest).toHaveBeenCalledWith(
				expect.any(AbortSignal),
				"POST",
				"https://api.example.com/v1/customers/test-customer/profiles/test-profile/virtualAgents/test-agent:generatePSTNTransferData",
				expect.objectContaining({
					callId: "test-contact-id",
					supportedDtmfChars: "0123456789*",
					ccaasMetadata: expect.objectContaining({
						ContactId: "test-contact-id",
						parameters: expect.objectContaining({
							customParam: "customValue",
						}),
					}),
				}),
			);

			// Verify filtered keys are not in parameters
			const callArgs = mockAPIClient.makeRequest.mock.calls[0];
			const payload = callArgs[3] as {
				ccaasMetadata: {
					parameters: Record<string, unknown>;
				};
			};
			expect(payload.ccaasMetadata.parameters.apiKey).toBeUndefined();
			expect(payload.ccaasMetadata.parameters.region).toBeUndefined();
		});

		it("should handle error response from server", async () => {
			mockAPIClient.makeRequest.mockRejectedValueOnce(
				new Error('request returned non-200 status: 400, body: {"error":"bad request"}'),
			);

			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {},
				},
			};

			const handlers = new Handlers(
				logger,
				mockAPIClient as unknown as CrestaAPIClient,
				"https://api.example.com",
				"test-customer",
				"test-profile",
				"test-agent",
				"0123456789*",
				event,
			);

			const controller = new AbortController();
			await expect(handlers.getPSTNTransferData(controller.signal)).rejects.toThrow();
		});
	});

	describe("getHandoffData", () => {
		it("should successfully make request and transform response", async () => {
			const mockResponse: FetchAIAgentHandoffResponse = {
				handoff: {
					conversation: "conversation-id",
					conversationCorrelationId: "correlation-id",
					summary: "test summary",
					transferTarget: "pstn:PSTN1",
				},
			};

			mockAPIClient.makeRequest.mockResolvedValueOnce(
				new TextEncoder().encode(JSON.stringify(mockResponse)),
			);

			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {},
				},
			};

			const handlers = new Handlers(
				logger,
				mockAPIClient as unknown as CrestaAPIClient,
				"https://api.example.com",
				"test-customer",
				"test-profile",
				"",
				"0123456789*",
				event,
			);

			const controller = new AbortController();
			const result = await handlers.getHandoffData(controller.signal);

			expect(result).toBeDefined();
			expect(result.handoff_conversation).toBe("conversation-id");
			expect(result.handoff_conversationCorrelationId).toBe("correlation-id");
			expect(result.handoff_summary).toBe("test summary");
			expect(result.handoff_transferTarget).toBe("pstn:PSTN1");

			// Verify the request was made with correct parameters
			expect(mockAPIClient.makeRequest).toHaveBeenCalledWith(
				expect.any(AbortSignal),
				"POST",
				"https://api.example.com/v1/customers/test-customer/profiles/test-profile/handoffs:fetchAIAgentHandoff",
				{
					correlationId: "test-contact-id",
				},
			);
		});

		it("should handle error response from server", async () => {
			mockAPIClient.makeRequest.mockRejectedValueOnce(
				new Error('request returned non-200 status: 404, body: {"error":"not found"}'),
			);

			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {},
				},
			};

			const handlers = new Handlers(
				logger,
				mockAPIClient as unknown as CrestaAPIClient,
				"https://api.example.com",
				"test-customer",
				"test-profile",
				"",
				"0123456789*",
				event,
			);

			const controller = new AbortController();
			await expect(handlers.getHandoffData(controller.signal)).rejects.toThrow();
		});
	});
});
