/**
 * Executes scenarios by building and running Go and TypeScript executables
 */

import * as child_process from "node:child_process";
import * as path from "node:path";
import { promisify } from "node:util";
import type { MockServerScenario } from "../mock-server/mock-server";
import type { TestScenario } from "../scenario-runner";

const exec = promisify(child_process.exec);

interface ExecutionResult {
	success: boolean;
	output?: string;
	error?: string;
	exitCode?: number;
}

/**
 * Builds the Go binary for testing
 */
export async function buildGoBinary(
	outputPath: string,
): Promise<{ success: boolean; error?: string }> {
	try {
		const goLambdaPath = path.resolve(
			__dirname,
			"../../../lambdas/pstn-transfer-go",
		);
		// Use environment variables for cross-compilation if set (e.g., in CI)
		// Default to local architecture for local development
		const goos =
			process.env.GOOS || (process.platform === "darwin" ? "darwin" : "linux");
		const goarch =
			process.env.GOARCH || (process.arch === "arm64" ? "arm64" : "amd64");
		const cmd = `cd ${goLambdaPath} && GOOS=${goos} GOARCH=${goarch} go build -tags lambda.norpc -o ${outputPath} .`;
		await exec(cmd);
		return { success: true };
	} catch (error) {
		return {
			success: false,
			error: `Failed to build Go binary: ${error instanceof Error ? error.message : String(error)}`,
		};
	}
}

/**
 * Builds/transpiles TypeScript handler
 */
export async function buildTypeScriptHandler(
	_outputDir: string,
): Promise<{ success: boolean; error?: string }> {
	try {
		const tsLambdaPath = path.resolve(
			__dirname,
			"../../../lambdas/pstn-transfer-ts",
		);
		// Build TypeScript
		await exec(`cd ${tsLambdaPath} && npm run build`);
		return { success: true };
	} catch (error) {
		return {
			success: false,
			error: `Failed to build TypeScript: ${error instanceof Error ? error.message : String(error)}`,
		};
	}
}

/**
 * Executes Go binary with event JSON and environment variables
 */
export async function executeGoBinary(
	binaryPath: string,
	event: unknown,
	env: Record<string, string>,
): Promise<ExecutionResult> {
	return new Promise((resolve) => {
		const eventJson = JSON.stringify(event);
		const envWithPath = {
			...process.env,
			...env,
		};

		const proc = child_process.spawn(binaryPath, ["--test"], {
			env: envWithPath,
			stdio: ["pipe", "pipe", "pipe"],
		});

		let stdout = "";
		let stderr = "";

		proc.stdout.on("data", (data) => {
			stdout += data.toString();
		});

		proc.stderr.on("data", (data) => {
			stderr += data.toString();
		});

		proc.on("close", (code) => {
			if (code === 0) {
				resolve({
					success: true,
					output: stdout.trim(),
				});
			} else {
				resolve({
					success: false,
					output: stdout.trim(),
					error: stderr.trim(),
					exitCode: code ?? undefined,
				});
			}
		});

		proc.on("error", (error) => {
			resolve({
				success: false,
				error: `Failed to spawn process: ${error.message}`,
			});
		});

		// Write event JSON to stdin
		proc.stdin.write(eventJson);
		proc.stdin.end();
	});
}

/**
 * Executes TypeScript handler with event JSON and environment variables
 */
export async function executeTypeScriptHandler(
	event: unknown,
	env: Record<string, string>,
): Promise<ExecutionResult> {
	return new Promise((resolve) => {
		const tsRunnerPath = path.resolve(__dirname, "ts-test-runner.ts");
		const eventJson = JSON.stringify(event);
		const envWithPath = {
			...process.env,
			...env,
		};

		// Use tsx to run TypeScript directly (handles ES modules properly)
		// Run from project root so imports resolve correctly
		const proc = child_process.spawn("npx", ["--yes", "tsx", tsRunnerPath], {
			env: envWithPath,
			stdio: ["pipe", "pipe", "pipe"],
			cwd: path.resolve(__dirname, "../../../../"),
		});

		let stdout = "";
		let stderr = "";

		proc.stdout.on("data", (data) => {
			stdout += data.toString();
		});

		proc.stderr.on("data", (data) => {
			stderr += data.toString();
		});

		proc.on("close", (code) => {
			if (code === 0) {
				resolve({
					success: true,
					output: stdout.trim(),
				});
			} else {
				resolve({
					success: false,
					output: stdout.trim(),
					error: stderr.trim(),
					exitCode: code ?? undefined,
				});
			}
		});

		proc.on("error", (error) => {
			resolve({
				success: false,
				error: `Failed to spawn process: ${error.message}`,
			});
		});

		// Write event JSON to stdin
		proc.stdin.write(eventJson);
		proc.stdin.end();
	});
}

/**
 * Executes a handler scenario against both Go and TypeScript implementations
 */
export async function executeHandlerScenario(
	scenario: TestScenario,
	mockServerURL: string,
	goBinaryPath: string,
	mockServer: { registerScenario: (scenario: MockServerScenario) => void },
): Promise<{
	goResult: ExecutionResult;
	tsResult: ExecutionResult;
}> {
	// Determine auth type from scenario
	const useOAuth = scenario.test.auth?.type === "oauth";
	const region = scenario.test.auth?.region || "us-west-2-prod";

	// If OAuth is used and scenario has auth endpoint mock, register it
	if (useOAuth && scenario.mock.path.includes("/oauth/")) {
		// Register auth endpoint mock
		mockServer.registerScenario({
			name: scenario.mock.path,
			path: scenario.mock.path,
			method: scenario.mock.method || "POST",
			responses: scenario.mock.responses,
		});
	}

	// Create test event from scenario
	const event = createTestEventFromScenario(
		scenario,
		mockServerURL,
		useOAuth,
		region,
	);

	// Set environment variables to override API endpoints
	const env: Record<string, string> = {
		apiDomain: mockServerURL,
		region,
	};

	if (useOAuth) {
		// Use OAuth - set authDomain to point to mock server (path will be appended automatically)
		env.authDomain = mockServerURL;
		env.oauthClientId = "test-client-id";
		env.oauthClientSecret = "test-client-secret";
	} else {
		// Use API key for simplicity
		env.apiKey = "test-api-key";
	}

	// Execute both implementations
	const [goResult, tsResult] = await Promise.all([
		executeGoBinary(goBinaryPath, event, env),
		executeTypeScriptHandler(event, env),
	]);

	return { goResult, tsResult };
}

/**
 * Creates a test event from a scenario
 */
function createTestEventFromScenario(
	scenario: TestScenario,
	mockServerURL: string,
	useOAuth: boolean,
	region: string,
): unknown {
	if (scenario.test.type !== "handler" || !scenario.test.action) {
		throw new Error("Invalid handler scenario");
	}

	const parameters: Record<string, string> = {
		action:
			scenario.test.action === "getPSTNTransferData"
				? "get_pstn_transfer_data"
				: "get_handoff_data",
		virtualAgentName:
			"customers/test-customer/profiles/test-profile/virtualAgents/test-agent",
		apiDomain: mockServerURL,
		region,
	};

	if (useOAuth) {
		// OAuth credentials come from environment variables, not event parameters
	} else {
		parameters.apiKey = "test-api-key";
	}

	const baseEvent: Record<string, unknown> = {
		Details: {
			ContactData: {
				ContactId: "test-contact-id",
				Channel: "VOICE",
				LanguageCode: "en-US",
			},
			Parameters: parameters,
		},
		Name: "ContactFlowEvent",
	};

	// Merge any event overrides from scenario
	if (scenario.test.event && typeof scenario.test.event === "object") {
		// Deep merge logic would go here if needed
	}

	return baseEvent;
}
