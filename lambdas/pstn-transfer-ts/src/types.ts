/**
 * Data models matching the Go implementation
 */

export interface ConnectEvent {
	Details: {
		ContactData: {
			ContactId: string;
			[key: string]: unknown;
		};
		Parameters: Record<string, string>;
	};
	Name?: string;
}

export interface ConnectResponse {
	[key: string]: string | number | boolean | null;
}

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
