/**
 * Mock API server for testing both Go and TypeScript implementations
 * This server can simulate various response scenarios
 */

import * as http from "node:http";
import * as url from "node:url";

export interface MockServerScenario {
	name: string;
	path: string;
	method?: string;
	responses: Array<{
		status: number;
		body: unknown;
		headers?: Record<string, string>;
	}>;
	currentAttempt?: number;
}

export class MockAPIServer {
	private server: http.Server | null = null;
	private port: number = 0;
	private scenarios: Map<string, MockServerScenario> = new Map();

	/**
	 * Registers a scenario for a specific path
	 */
	registerScenario(scenario: MockServerScenario): void {
		const key = `${scenario.method || "GET"}:${scenario.path}`;
		scenario.currentAttempt = 0;
		this.scenarios.set(key, scenario);
	}

	/**
	 * Starts the mock server on a specific port or random port if not specified
	 */
	start(port?: number): Promise<number> {
		return new Promise((resolve, reject) => {
			this.server = http.createServer((req, res) => {
				const parsedUrl = url.parse(req.url || "", true);
				const pathname = parsedUrl.pathname || "";
				const method = req.method || "GET";
				const key = `${method}:${pathname}`;

				const scenario = this.scenarios.get(key);

				if (!scenario) {
					res.writeHead(404, { "Content-Type": "application/json" });
					res.end(JSON.stringify({ error: "Scenario not found", key }));
					return;
				}

				const attempt = scenario.currentAttempt || 0;
				const response =
					scenario.responses[attempt] ||
					scenario.responses[scenario.responses.length - 1];

				// Increment attempt for next call
				scenario.currentAttempt = Math.min(
					attempt + 1,
					scenario.responses.length - 1,
				);

				// Set headers
				const headers = {
					"Content-Type": "application/json",
					...response.headers,
				};

				res.writeHead(response.status, headers);
				res.end(JSON.stringify(response.body));
			});

			const listenPort = port || 0;
			this.server.listen(listenPort, () => {
				const address = this.server?.address();
				if (address && typeof address === "object") {
					this.port = address.port;
					resolve(this.port);
				} else {
					reject(new Error("Failed to get server port"));
				}
			});

			this.server.on("error", reject);
		});
	}

	/**
	 * Gets the server URL
	 */
	getURL(): string {
		return `http://localhost:${this.port}`;
	}

	/**
	 * Resets all scenarios to their initial state
	 */
	reset(): void {
		for (const scenario of this.scenarios.values()) {
			scenario.currentAttempt = 0;
		}
	}

	/**
	 * Stops the mock server
	 */
	stop(): Promise<void> {
		return new Promise((resolve, reject) => {
			if (!this.server) {
				resolve();
				return;
			}

			this.server.close((err) => {
				if (err) {
					reject(err);
				} else {
					this.server = null;
					resolve();
				}
			});
		});
	}
}

/**
 * Creates an empty mock server
 * Scenarios should be registered from JSON files using registerScenarioMock
 */
export function createMockServer(): MockAPIServer {
	return new MockAPIServer();
}
