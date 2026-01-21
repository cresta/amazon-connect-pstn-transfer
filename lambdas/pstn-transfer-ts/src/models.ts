/**
 * Data models matching the Go implementation
 */

export interface FetchAIAgentHandoffResponse {
	handoff: Handoff;
}

export interface Handoff {
	conversation: string;
	conversationCorrelationId: string;
	summary: string;
	transferTarget: string;
	// NOTE: We don't need the metadataByTaxonomy field.
}
