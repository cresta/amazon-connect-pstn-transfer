/**
 * Tests for the TypeScript handler matching Go test structure
 */

import type { OAuth2TokenFetcher } from "./auth.js";
import { DefaultHandlerService, handler } from "./handler.js";
import { Logger } from "./logger.js";
import type { FetchAIAgentHandoffResponse } from "./models.js";
import type { ConnectEvent, ConnectResponse } from "./utils.js";

// Mock fetch globally
globalThis.fetch = jest.fn() as typeof fetch;

describe("HandlerService", () => {
	let mockTokenFetcher: jest.Mocked<OAuth2TokenFetcher>;
	let logger: Logger;

	beforeEach(() => {
		jest.clearAllMocks();
		(globalThis.fetch as jest.Mock).mockReset();
		logger = new Logger();
		mockTokenFetcher = {
			getToken: jest.fn(),
		};
	});

	describe("handle", () => {
		it("should successfully handle get_pstn_transfer_data with API key", async () => {
			const mockResponse: ConnectResponse = {
				phoneNumber: "+1234567890",
				dtmfSequence: "1234",
			};

			(globalThis.fetch as jest.Mock).mockResolvedValueOnce({
				status: 200,
				json: async () => mockResponse,
				arrayBuffer: async () => new TextEncoder().encode(JSON.stringify(mockResponse)),
			});

			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {
						action: "get_pstn_transfer_data",
						apiKey: "test-api-key",
						virtualAgentName:
							"customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
						customParam: "customValue",
						region: "us-west-2-prod",
					},
				},
			};

			const service = new DefaultHandlerService(logger);
			const controller = new AbortController();
			const result = await service.handle(controller.signal, event);

			expect(result).toBeDefined();
			expect(result.phoneNumber).toBe("+1234567890");
			expect(result.dtmfSequence).toBe("1234");
		});

		it("should successfully handle get_pstn_transfer_data with OAuth 2", async () => {
			const mockResponse: ConnectResponse = {
				phoneNumber: "+1234567890",
				dtmfSequence: "1234",
			};

			mockTokenFetcher.getToken.mockResolvedValueOnce("test-oauth-token");

			(globalThis.fetch as jest.Mock)
				.mockResolvedValueOnce({
					status: 200,
					json: async () => mockResponse,
					arrayBuffer: async () => new TextEncoder().encode(JSON.stringify(mockResponse)),
				})
				.mockResolvedValueOnce({
					status: 200,
					json: async () => mockResponse,
					arrayBuffer: async () => new TextEncoder().encode(JSON.stringify(mockResponse)),
				});

			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {
						action: "get_pstn_transfer_data",
						oauthClientId: "test-client-id",
						oauthClientSecret: "test-client-secret",
						virtualAgentName:
							"customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
						region: "us-east-1-prod",
					},
				},
			};

			const service = new DefaultHandlerService(logger, mockTokenFetcher);
			const controller = new AbortController();
			const result = await service.handle(controller.signal, event);

			expect(result).toBeDefined();
			expect(result.phoneNumber).toBe("+1234567890");
			expect(result.dtmfSequence).toBe("1234");
		});

		it("should successfully handle get_handoff_data", async () => {
			const mockHandoffResponse: FetchAIAgentHandoffResponse = {
				handoff: {
					conversation: "conversation-id",
					conversationCorrelationId: "correlation-id",
					summary: "test summary",
					transferTarget: "pstn:PSTN1",
				},
			};

			// Mock fetch for the API call (not OAuth token fetch since we're using API key)
			(globalThis.fetch as jest.Mock).mockResolvedValueOnce({
				status: 200,
				json: async () => mockHandoffResponse,
				arrayBuffer: async () => new TextEncoder().encode(JSON.stringify(mockHandoffResponse)),
				text: async () => JSON.stringify(mockHandoffResponse),
			});

			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {
						action: "get_handoff_data",
						apiKey: "test-api-key",
						virtualAgentName:
							"customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
						region: "us-west-2-prod",
					},
				},
			};

			const service = new DefaultHandlerService(logger);
			const controller = new AbortController();
			const result = await service.handle(controller.signal, event);

			expect(result).toBeDefined();
			expect(result.handoff_conversation).toBe("conversation-id");
			expect(result.handoff_conversationCorrelationId).toBe("correlation-id");
			expect(result.handoff_summary).toBe("test summary");
			expect(result.handoff_transferTarget).toBe("pstn:PSTN1");
		});

		it("should throw error when virtualAgentName is missing", async () => {
			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {
						action: "get_pstn_transfer_data",
						apiKey: "test-api-key",
						region: "us-west-2-prod",
					},
				},
			};

			const service = new DefaultHandlerService(logger);
			const controller = new AbortController();
			await expect(service.handle(controller.signal, event)).rejects.toThrow(
				"virtualAgentName is required",
			);
		});

		it("should throw error when authentication is missing", async () => {
			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {
						action: "get_pstn_transfer_data",
						virtualAgentName:
							"customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
						region: "us-west-2-prod",
					},
				},
			};

			const service = new DefaultHandlerService(logger);
			const controller = new AbortController();
			await expect(service.handle(controller.signal, event)).rejects.toThrow(
				"either apiKey (deprecated) or oauthClientId/oauthClientSecret must be provided",
			);
		});

		it("should throw error for invalid action", async () => {
			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {
						action: "invalid_action",
						apiKey: "test-api-key",
						virtualAgentName:
							"customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
						region: "us-west-2-prod",
					},
				},
			};

			const service = new DefaultHandlerService(logger);
			const controller = new AbortController();
			await expect(service.handle(controller.signal, event)).rejects.toThrow(
				"invalid action: invalid_action",
			);
		});

		it("should throw error for invalid virtual agent name format", async () => {
			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {
						action: "get_pstn_transfer_data",
						apiKey: "test-api-key",
						virtualAgentName: "invalid-format",
						region: "us-west-2-prod",
					},
				},
			};

			const service = new DefaultHandlerService(logger);
			const controller = new AbortController();
			await expect(service.handle(controller.signal, event)).rejects.toThrow(
				"invalid virtual agent name",
			);
		});
	});
});

describe("handler", () => {
	it("should be callable as Lambda handler", async () => {
		const mockResponse: ConnectResponse = {
			phoneNumber: "+1234567890",
			dtmfSequence: "1234",
		};

		(globalThis.fetch as jest.Mock).mockResolvedValueOnce({
			status: 200,
			json: async () => mockResponse,
			arrayBuffer: async () => new TextEncoder().encode(JSON.stringify(mockResponse)),
		});

		const event: ConnectEvent = {
			Details: {
				ContactData: {
					ContactId: "test-contact-id",
				},
				Parameters: {
					action: "get_pstn_transfer_data",
					apiKey: "test-api-key",
					virtualAgentName:
						"customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
					region: "us-west-2-prod",
				},
			},
		};

		const result = await handler(event);
		expect(result).toBeDefined();
	});
});
