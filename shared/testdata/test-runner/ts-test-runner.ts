#!/usr/bin/env node
/// <reference types="node" />
/**
 * Test runner for TypeScript handler
 * Reads event JSON from stdin and outputs response JSON to stdout
 */

// Import from source (tsx will compile on the fly)
import { handler } from "../../../lambdas/pstn-transfer-ts/src/handler";

async function main() {
	try {
		// Read event from stdin
		let input = "";
		process.stdin.setEncoding("utf8");
		for await (const chunk of process.stdin) {
			input += chunk;
		}

		const event = JSON.parse(input);
		const result = await handler(event);

		// Output response to stdout
		process.stdout.write(`${JSON.stringify(result)}\n`);
		process.exit(0);
	} catch (error) {
		const errorMessage = error instanceof Error ? error.message : String(error);
		process.stderr.write(`Error: ${errorMessage}\n`);
		process.exit(1);
	}
}

main();
