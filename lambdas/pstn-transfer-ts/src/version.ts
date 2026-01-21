/**
 * Version information for the Lambda function
 * This is injected at build time via esbuild --define flag
 * Falls back to environment variable or "unknown" if not set
 */

// This will be replaced at build time by esbuild --define
// The identifier __VERSION__ is replaced with the actual version string during build
// @ts-expect-error - __VERSION__ is injected at build time by esbuild, TypeScript doesn't know about it
const VERSION_VALUE: unknown = typeof __VERSION__ !== "undefined" ? __VERSION__ : undefined;
export const VERSION: string =
	(typeof VERSION_VALUE === "string" ? VERSION_VALUE : undefined) ||
	process.env.LAMBDA_VERSION ||
	"unknown";
