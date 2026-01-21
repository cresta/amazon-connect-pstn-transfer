#!/usr/bin/env node
/**
 * Test runner for TypeScript Lambda handler
 * Reads event JSON from file or stdin and calls the handler
 */

import * as fs from "node:fs";
import * as path from "node:path";
import { handler } from "./handler.js";
import type { ConnectEvent } from "./utils.js";

async function main() {
	try {
		const eventFile = process.argv[2];

		let eventJson: string;
		if (eventFile) {
			// Read from file
			const filePath = path.isAbsolute(eventFile)
				? eventFile
				: path.resolve(process.cwd(), eventFile);
			eventJson = fs.readFileSync(filePath, "utf-8");
		} else {
			// Read from stdin
			let input = "";
			process.stdin.setEncoding("utf8");
			for await (const chunk of process.stdin) {
				input += chunk;
			}
			eventJson = input;
		}

		const event = JSON.parse(eventJson) as ConnectEvent;
		const result = await handler(event);

		// Output response to stdout
		console.log(JSON.stringify(result, null, 2));
		process.exit(0);
	} catch (error) {
		const errorMessage = error instanceof Error ? error.message : String(error);
		console.error(`Error: ${errorMessage}`);
		if (error instanceof Error && error.stack) {
			console.error(error.stack);
		}
		process.exit(1);
	}
}

void main();
