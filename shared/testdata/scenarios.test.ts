/**
 * Integration tests that validate both Go and TypeScript implementations
 * by building executables and running them against a mock server
 */

import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import type { MockAPIServer } from "./mock-server/mock-server";
import { loadScenarios, registerScenarioMock } from "./scenario-runner";
import {
	buildGoBinary,
	buildTypeScriptHandler,
	executeHandlerScenario,
} from "./test-runner/executor";

describe("Integration Tests - Shared Scenarios", () => {
	let mockServer: MockAPIServer;
	let serverURL: string;
	let goBinaryPath: string;
	let tempDir: string;

	beforeAll(async () => {
		// Create temp directory for binaries
		tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "pstn-transfer-test-"));
		goBinaryPath = path.join(tempDir, "handler-go");

		// Build executables
		console.log("Building Go binary...");
		const goBuildResult = await buildGoBinary(goBinaryPath);
		if (!goBuildResult.success) {
			throw new Error(`Failed to build Go binary: ${goBuildResult.error}`);
		}

		console.log("Building TypeScript handler...");
		const tsBuildResult = await buildTypeScriptHandler(tempDir);
		if (!tsBuildResult.success) {
			throw new Error(`Failed to build TypeScript: ${tsBuildResult.error}`);
		}

		// Create and start mock server
		const { MockAPIServer } = await import("./mock-server/mock-server");
		mockServer = new MockAPIServer();
		const port = await mockServer.start();
		serverURL = `http://localhost:${port}`;
		console.log(`Mock server started on ${serverURL}`);
	}, 60000); // 60 second timeout for building

	afterAll(async () => {
		if (mockServer) {
			await mockServer.stop();
		}
		// Cleanup temp directory
		if (tempDir && fs.existsSync(tempDir)) {
			fs.rmSync(tempDir, { recursive: true, force: true });
		}
	});

	beforeEach(() => {
		if (mockServer) {
			mockServer.reset();
		}
	});

	/**
	 * Filters log lines from output and extracts JSON
	 */
	function extractJSON(output: string): string {
		return output
			.split("\n")
			.filter((line) => !line.match(/^\[(INFO|WARN|ERROR|DEBUG)\]/))
			.join("\n")
			.trim();
	}

	// Run all scenarios from scenario JSON files
	describe("Scenarios", () => {
		// Load scenarios at test execution time, not module load time
		const testScenarios = loadScenarios();

		for (const scenario of testScenarios) {
			it(`should execute scenario: ${scenario.name}`, async () => {
				// Register the mock scenario (API endpoint)
				registerScenarioMock(mockServer, scenario.mock);

				// Execute both implementations
				const { goResult, tsResult } = await executeHandlerScenario(
					scenario,
					serverURL,
					goBinaryPath,
					mockServer,
				);

				// Check if scenario expects failure (explicit shouldFail or error status)
				const expectsFailure =
					scenario.expectations.shouldFail === true ||
					(scenario.expectations.status !== undefined &&
						scenario.expectations.status >= 400 &&
						scenario.expectations.status < 500);

				if (expectsFailure) {
					// Both should fail with similar errors
					expect(goResult.success).toBe(false);
					expect(tsResult.success).toBe(false);

					// Check error messages contain expected text
					if (scenario.expectations.errorContains) {
						const goError = goResult.error?.toLowerCase() || "";
						const tsError = tsResult.error?.toLowerCase() || "";
						const expectedText =
							scenario.expectations.errorContains.toLowerCase();

						expect(goError).toContain(expectedText);
						expect(tsError).toContain(expectedText);
					}

					// Both should fail in the same way
					expect(goResult.exitCode).toBe(tsResult.exitCode);
					return;
				}

				// Validate Go result
				expect(goResult.success).toBe(true);
				if (!goResult.success) {
					throw new Error(`Go execution failed: ${goResult.error}`);
				}

				// Validate TypeScript result
				expect(tsResult.success).toBe(true);
				if (!tsResult.success) {
					throw new Error(`TypeScript execution failed: ${tsResult.error}`);
				}

				// Parse and compare responses
				// Filter out log lines and extract JSON
				const goResponse = JSON.parse(
					extractJSON(goResult.output || "") || "{}",
				);
				const tsResponse = JSON.parse(
					extractJSON(tsResult.output || "") || "{}",
				);

				// Validate expectations
				if (scenario.expectations.body) {
					for (const [key, expectedValue] of Object.entries(
						scenario.expectations.body,
					)) {
						// Handle nested objects (e.g., handoff.conversation)
						if (
							typeof expectedValue === "object" &&
							expectedValue !== null &&
							!Array.isArray(expectedValue)
						) {
							// This is a nested object, check each nested property
							for (const [nestedKey, nestedValue] of Object.entries(
								expectedValue as Record<string, unknown>,
							)) {
								const flatKey = `${key}_${nestedKey}`;
								expect(goResponse[flatKey]).toBe(nestedValue);
								expect(tsResponse[flatKey]).toBe(nestedValue);
							}
						} else {
							expect(goResponse[key]).toBe(expectedValue);
							expect(tsResponse[key]).toBe(expectedValue);
						}
					}
				}

				// Ensure both implementations return the same result
				expect(goResponse).toEqual(tsResponse);
			}, 30000); // 30 second timeout per test
		}
	});
});
