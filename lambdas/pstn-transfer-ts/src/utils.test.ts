/**
 * Tests for utility functions matching Go test structure
 */

import {
	buildAPIDomainFromRegion,
	extractRegionFromDomain,
	getAuthRegion,
	getFromEventParameterOrEnv,
	parseVirtualAgentName,
	validateDomain,
} from "./utils.js";
import type { ConnectEvent } from "./types.js";

describe("Utils", () => {
	describe("extractRegionFromDomain", () => {
		it("should extract region from valid domain with -prod suffix", () => {
			const domain = "https://api.us-west-2-prod.cresta.ai";
			const result = extractRegionFromDomain(domain);
			expect(result).toBe("us-west-2-prod");
		});

		it("should extract region from valid domain with -staging suffix", () => {
			const domain = "https://api.us-east-1-staging.cresta.ai";
			const result = extractRegionFromDomain(domain);
			expect(result).toBe("us-east-1-staging");
		});

		it("should extract region from valid domain without protocol", () => {
			const domain = "api.eu-west-1-prod.cresta.ai";
			const result = extractRegionFromDomain(domain);
			expect(result).toBe("eu-west-1-prod");
		});

		it("should extract region from api-customer-profile.cresta.com domain", () => {
			const domain = "https://api-customer-profile.cresta.com";
			const result = extractRegionFromDomain(domain);
			expect(result).toBe("customer-profile");
		});

		it("should extract region from api-customer-profile.cresta.com domain without protocol", () => {
			const domain = "api-customer-profile.cresta.com";
			const result = extractRegionFromDomain(domain);
			expect(result).toBe("customer-profile");
		});

		it("should throw error for invalid domain format", () => {
			const domain = "https://invalid-domain.com";
			expect(() => extractRegionFromDomain(domain)).toThrow();
		});

		it("should throw error for domain without cresta.ai/cresta.com", () => {
			const domain = "https://api.us-west-2-prod.example.com";
			expect(() => extractRegionFromDomain(domain)).toThrow();
		});

		it("should throw error for empty string", () => {
			const domain = "";
			expect(() => extractRegionFromDomain(domain)).toThrow();
		});
	});

	describe("buildAPIDomainFromRegion", () => {
		it("should build domain from region with -prod suffix", () => {
			const region = "us-west-2-prod";
			const result = buildAPIDomainFromRegion(region);
			expect(result).toBe("https://api.us-west-2-prod.cresta.com");
		});

		it("should build domain from region with -staging suffix", () => {
			const region = "us-east-1-staging";
			const result = buildAPIDomainFromRegion(region);
			expect(result).toBe("https://api.us-east-1-staging.cresta.ai");
		});

		it("should build domain from region with custom suffix", () => {
			const region = "eu-west-1-dev";
			const result = buildAPIDomainFromRegion(region);
			expect(result).toBe("https://api.eu-west-1-dev.cresta.ai");
		});

		it("should build domain from customer-profile region", () => {
			const region = "customer-profile";
			const result = buildAPIDomainFromRegion(region);
			expect(result).toBe("https://api.customer-profile.cresta.ai");
		});
	});

	describe("getFromEventParameterOrEnv", () => {
		it("should get value from event parameters", () => {
			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {
						testKey: "event-value",
					},
				},
			};
			const result = getFromEventParameterOrEnv(event, "testKey", "default-value");
			expect(result).toBe("event-value");
		});

		it("should get value from environment variable", () => {
			const originalEnv = process.env.testKey;
			process.env.testKey = "env-value";
			try {
				const event: ConnectEvent = {
					Details: {
						ContactData: {
							ContactId: "test-contact-id",
						},
						Parameters: {},
					},
				};
				const result = getFromEventParameterOrEnv(event, "testKey", "default-value");
				expect(result).toBe("env-value");
			} finally {
				if (originalEnv !== undefined) {
					process.env.testKey = originalEnv;
				} else {
					delete process.env.testKey;
				}
			}
		});

		it("should return default value when not in event or env", () => {
			const event: ConnectEvent = {
				Details: {
					ContactData: {
						ContactId: "test-contact-id",
					},
					Parameters: {},
				},
			};
			const result = getFromEventParameterOrEnv(event, "testKey", "default-value");
			expect(result).toBe("default-value");
		});

		it("should prioritize event parameter over env", () => {
			const originalEnv = process.env.testKey;
			process.env.testKey = "env-value";
			try {
				const event: ConnectEvent = {
					Details: {
						ContactData: {
							ContactId: "test-contact-id",
						},
						Parameters: {
							testKey: "event-value",
						},
					},
				};
				const result = getFromEventParameterOrEnv(event, "testKey", "default-value");
				expect(result).toBe("event-value");
			} finally {
				if (originalEnv !== undefined) {
					process.env.testKey = originalEnv;
				} else {
					delete process.env.testKey;
				}
			}
		});
	});

	describe("parseVirtualAgentName", () => {
		it("should parse valid virtual agent name", () => {
			const virtualAgentName =
				"customers/test-customer/profiles/test-profile/virtualAgents/test-agent";
			const result = parseVirtualAgentName(virtualAgentName);
			expect(result.customer).toBe("test-customer");
			expect(result.profile).toBe("test-profile");
			expect(result.virtualAgentID).toBe("test-agent");
		});

		it("should throw error for invalid format", () => {
			const virtualAgentName = "invalid-format";
			expect(() => parseVirtualAgentName(virtualAgentName)).toThrow();
		});
	});

	describe("validateDomain", () => {
		it("should validate HTTPS domain", () => {
			const domain = "https://api.example.com";
			expect(() => validateDomain(domain)).not.toThrow();
		});

		it("should validate HTTPS domain with trailing slash", () => {
			const domain = "https://api.example.com/";
			expect(() => validateDomain(domain)).not.toThrow();
		});

		it("should validate HTTP localhost domain", () => {
			const domain = "http://localhost:8080";
			expect(() => validateDomain(domain)).not.toThrow();
		});

		it("should validate api-customer-profile.cresta.com domain", () => {
			const domain = "https://api-customer-profile.cresta.com";
			expect(() => validateDomain(domain)).not.toThrow();
		});

		it("should reject HTTP non-localhost domain", () => {
			const domain = "http://api.example.com";
			expect(() => validateDomain(domain)).toThrow();
		});

		it("should reject domain with path", () => {
			const domain = "https://api.example.com/path";
			expect(() => validateDomain(domain)).toThrow();
		});

		it("should reject empty domain", () => {
			const domain = "";
			expect(() => validateDomain(domain)).toThrow();
		});
	});

	describe("getAuthRegion", () => {
		it("should map chat-prod to us-west-2-prod", () => {
			const region = "chat-prod";
			const result = getAuthRegion(region);
			expect(result).toBe("us-west-2-prod");
		});

		it("should map voice-prod to us-west-2-prod", () => {
			const region = "voice-prod";
			const result = getAuthRegion(region);
			expect(result).toBe("us-west-2-prod");
		});

		it("should return valid auth region as-is", () => {
			const region = "us-west-2-prod";
			const result = getAuthRegion(region);
			expect(result).toBe("us-west-2-prod");
		});

		it("should return us-east-1-prod as-is", () => {
			const region = "us-east-1-prod";
			const result = getAuthRegion(region);
			expect(result).toBe("us-east-1-prod");
		});

		it("should return chat-staging as-is", () => {
			const region = "chat-staging";
			const result = getAuthRegion(region);
			expect(result).toBe("chat-staging");
		});

		it("should return unknown custom region as-is", () => {
			const region = "customer-profile";
			const result = getAuthRegion(region);
			expect(result).toBe("customer-profile");
		});

		it("should return customer-profile as-is", () => {
			const region = "customer-profile";
			const result = getAuthRegion(region);
			expect(result).toBe("customer-profile");
		});
	});
});
