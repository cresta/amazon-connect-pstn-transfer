/**
 * Utility functions matching the Go implementation
 */

import type { ConnectEvent } from "./types.js";

const API_DOMAIN_REGEX = /api\.([a-z0-9-]+)\.cresta\.(ai|com)/;
const VIRTUAL_AGENT_NAME_REGEX = /^customers\/([^/]+)\/profiles\/([^/]+)\/virtualAgents\/([^/]+)$/;

/**
 * getFromEventParameterOrEnv retrieves a value from event parameters or environment variables
 */
export function getFromEventParameterOrEnv(
	event: ConnectEvent,
	key: string,
	defaultValue: string,
): string {
	if (event.Details.Parameters[key]) {
		return event.Details.Parameters[key];
	}
	if (process.env[key]) {
		return process.env[key];
	}
	return defaultValue;
}

/**
 * copyMap creates a copy of a map excluding filtered keys
 */
export function copyMap(
	original: Record<string, string>,
	filteredKeys: Record<string, boolean>,
): Record<string, unknown> {
	const result: Record<string, unknown> = {};
	for (const [k, v] of Object.entries(original)) {
		if (!filteredKeys[k]) {
			result[k] = v;
		}
	}
	return result;
}

/**
 * parseVirtualAgentName parses a virtual agent name
 * Format: customers/{customer}/profiles/{profile}/virtualAgents/{virtualAgentID}
 */
export function parseVirtualAgentName(virtualAgentName: string): {
	customer: string;
	profile: string;
	virtualAgentID: string;
} {
	const matches = VIRTUAL_AGENT_NAME_REGEX.exec(virtualAgentName);
	if (!matches || matches.length !== 4) {
		throw new Error(
			`invalid virtual agent name: ${virtualAgentName}. Expected format: customers/{customer}/profiles/{profile}/virtualAgents/{virtualAgentID}`,
		);
	}
	return {
		customer: matches[1],
		profile: matches[2],
		virtualAgentID: matches[3],
	};
}

/**
 * buildAPIDomainFromRegion builds an API domain URL from a region
 * e.g., "us-west-2-prod" -> "https://api.us-west-2-prod.cresta.com"
 * e.g., "us-west-2-staging" -> "https://api.us-west-2-staging.cresta.ai"
 */
export function buildAPIDomainFromRegion(region: string): string {
	const normalized = region.toLowerCase();
	if (!/^[a-z0-9-]+$/.test(normalized)) {
		throw new Error(`invalid region: ${region}`);
	}
	if (normalized.endsWith("-prod")) {
		return `https://api.${normalized}.cresta.com`;
	}
	return `https://api.${normalized}.cresta.ai`;
}

/**
 * extractRegionFromDomain extracts the AWS region from the API domain
 */
export function extractRegionFromDomain(apiDomain: string): string {
	const matches = API_DOMAIN_REGEX.exec(apiDomain);
	if (!matches || matches.length < 2) {
		throw new Error(`could not extract region from domain: ${apiDomain}`);
	}
	return matches[1];
}

/**
 * getIntFromEnv retrieves an integer from environment variable or returns default
 */
export function getIntFromEnv(key: string, defaultValue: number): number {
	const value = process.env[key];
	if (value) {
		const intValue = parseInt(value, 10);
		if (!Number.isNaN(intValue)) {
			return intValue;
		}
	}
	return defaultValue;
}

/**
 * getDurationFromEnv retrieves a duration from environment variable or returns default
 * Accepts duration strings like "100ms", "2s", "1m", etc.
 */
export function getDurationFromEnv(key: string, defaultValueMs: number): number {
	const value = process.env[key];
	if (value) {
		// Simple duration parsing: supports ms, s, m, h
		const match = value.match(/^(\d+)(ms|s|m|h)$/);
		if (match) {
			const num = parseInt(match[1], 10);
			const unit = match[2];
			const multipliers: Record<string, number> = {
				ms: 1,
				s: 1000,
				m: 60 * 1000,
				h: 60 * 60 * 1000,
			};
			return num * multipliers[unit];
		}
	}
	return defaultValueMs;
}

/**
 * validateDomain validates that a domain is a safe URL for API requests
 */
export function validateDomain(domain: string): void {
	if (!domain) {
		throw new Error("domain cannot be empty");
	}

	let parsedURL: URL;
	try {
		parsedURL = new URL(domain);
	} catch (err) {
		const errorMessage = err instanceof Error ? err.message : String(err);
		throw new Error(`invalid domain URL: ${errorMessage}`);
	}

	// Require HTTPS scheme for security, except for localhost (testing)
	const isLocalhost =
		parsedURL.hostname === "localhost" ||
		parsedURL.hostname === "127.0.0.1" ||
		parsedURL.hostname.startsWith("127.");
	if (parsedURL.protocol !== "https:" && !(parsedURL.protocol === "http:" && isLocalhost)) {
		throw new Error(`domain must use HTTPS scheme, got: ${parsedURL.protocol}`);
	}

	// Reject domains with path, query, or fragment components
	if (parsedURL.pathname && parsedURL.pathname !== "/") {
		throw new Error(`domain cannot contain path components: ${parsedURL.pathname}`);
	}
	if (parsedURL.search) {
		throw new Error(`domain cannot contain query parameters: ${parsedURL.search}`);
	}
	if (parsedURL.hash) {
		throw new Error(`domain cannot contain fragment: ${parsedURL.hash}`);
	}

	// Ensure host is present
	if (!parsedURL.host) {
		throw new Error("domain must have a host");
	}

	// Check for path traversal attempts in host
	if (parsedURL.host.includes("/") || parsedURL.host.includes("..")) {
		throw new Error("domain host contains invalid characters");
	}
}

/**
 * validatePathSegment validates that a path segment is safe
 */
export function validatePathSegment(segment: string, name: string): void {
	if (!segment) {
		throw new Error(`${name} cannot be empty`);
	}

	// Reject path traversal attempts
	if (segment.includes("..") || segment.includes("/")) {
		throw new Error(`${name} contains invalid characters (path traversal detected): ${segment}`);
	}

	// Reject URL-encoded path traversal (case-insensitive)
	const lowerSegment = segment.toLowerCase();
	if (lowerSegment.includes("%2e%2e")) {
		throw new Error(`${name} contains URL-encoded path traversal: ${segment}`);
	}

	// Reject null bytes
	if (segment.includes("\x00")) {
		throw new Error(`${name} contains null byte`);
	}
}
