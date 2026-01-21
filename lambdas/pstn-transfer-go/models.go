package main

// FetchAIAgentHandoffResponse represents the API response for fetching AI agent handoff data.
type FetchAIAgentHandoffResponse struct {
	Handoff Handoff `json:"handoff"`
}

// Handoff contains handoff information from the API.
type Handoff struct {
	Conversation              string `json:"conversation"`
	ConversationCorrelationID string `json:"conversationCorrelationId"`
	Summary                   string `json:"summary"`
	TransferTarget            string `json:"transferTarget"`
	// NOTE: We don't need the metadataByTaxonomy field.
}
