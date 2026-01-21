/**
 * Scenario loader and types for shared test scenarios
 * Scenarios are executed via built executables (see test-runner/executor.ts)
 */

import * as fs from "node:fs";
import * as path from "node:path";
import type { MockAPIServer } from "./mock-server/mock-server";

export interface ScenarioMock {
	path: string;
	method?: string;
	responses: Array<{
		status: number;
		body: unknown;
		headers?: Record<string, string>;
	}>;
}

export interface ScenarioTest {
	type: "handler";
	action?: string;
	event?: unknown;
	auth?: {
		type: "api_key" | "oauth";
		region?: string;
	};
}

export interface ScenarioExpectations {
	status?: number;
	finalStatus?: number;
	retries?: boolean;
	body?: Record<string, unknown>;
	shouldFail?: boolean;
	errorContains?: string;
}

export interface TestScenario {
	name: string;
	description: string;
	mock: ScenarioMock;
	test: ScenarioTest;
	expectations: ScenarioExpectations;
}

/**
 * Loads scenarios from scenario directories (one file per scenario)
 * Automatically discovers all subfolders in the scenarios directory
 */
export function loadScenarios(): TestScenario[] {
	// Jest/ts-jest provides __dirname at runtime
	// Resolve scenarios directory relative to this file
	const scenariosBaseDir = path.resolve(__dirname, "scenarios");
	const allScenarios: TestScenario[] = [];

	if (
		!fs.existsSync(scenariosBaseDir) ||
		!fs.statSync(scenariosBaseDir).isDirectory()
	) {
		return allScenarios;
	}

	// Read all subdirectories in scenarios folder
	const subdirs = fs.readdirSync(scenariosBaseDir, { withFileTypes: true });

	for (const subdir of subdirs) {
		if (!subdir.isDirectory()) {
			continue;
		}

		const dirPath = path.join(scenariosBaseDir, subdir.name);
		const files = fs.readdirSync(dirPath);

		for (const file of files) {
			if (!file.endsWith(".json")) {
				continue;
			}

			const filePath = path.join(dirPath, file);
			const content = fs.readFileSync(filePath, "utf-8");
			const scenario = JSON.parse(content) as TestScenario;
			allScenarios.push(scenario);
		}
	}

	return allScenarios;
}

/**
 * Registers a scenario's mock configuration on the mock server
 */
export function registerScenarioMock(
	server: MockAPIServer,
	mock: ScenarioMock,
): void {
	server.registerScenario({
		name: mock.path,
		path: mock.path,
		method: mock.method || "GET",
		responses: mock.responses,
	});
}
