/**
 * Logger provides structured logging functionality matching the Go implementation
 */

export class Logger {
	private debugEnabled: boolean;

	constructor() {
		this.debugEnabled = process.env.DEBUG_LOGGING === "true";
	}

	debugf(format: string, ...args: unknown[]): void {
		if (this.debugEnabled) {
			console.log(`[DEBUG] ${this.formatMessage(format, ...args)}`);
		}
	}

	infof(format: string, ...args: unknown[]): void {
		console.log(`[INFO] ${this.formatMessage(format, ...args)}`);
	}

	errorf(format: string, ...args: unknown[]): void {
		console.error(`[ERROR] ${this.formatMessage(format, ...args)}`);
	}

	warnf(format: string, ...args: unknown[]): void {
		console.warn(`[WARN] ${this.formatMessage(format, ...args)}`);
	}

	private formatMessage(format: string, ...args: unknown[]): string {
		// Simple format string replacement: %s, %v, %d, etc.
		let message = format;
		for (const arg of args) {
			let value: string;
			if (typeof arg === "object" && arg !== null) {
				value = JSON.stringify(arg);
			} else if (typeof arg === "number") {
				value = String(arg);
			} else {
				value = String(arg);
			}
			message = message.replace(/%[svd]/, value);
		}
		return message;
	}
}

export function newLogger(): Logger {
	return new Logger();
}
