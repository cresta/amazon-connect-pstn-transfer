/**
 * Handlers for different API actions matching the Go implementation
 */

import type { CrestaAPIClient } from "./client.js";
import type { Logger } from "./logger.js";
import type { ConnectEvent, ConnectResponse, FetchAIAgentHandoffResponse } from "./types.js";
import { copyMap } from "./utils.js";
import { VERSION } from "./version.js";

const FilteredKeys: Record<string, boolean> = {
	apiDomain: true,
	region: true,
	action: true,
	apiKey: true,
	oauthClientId: true,
	oauthClientSecret: true,
	virtualAgentName: true,
	supportedDtmfChars: true,
};

export class Handlers {
	private logger: Logger;
	private apiClient: CrestaAPIClient;
	private domain: string;
	private customerID: string;
	private profileID: string;
	private virtualAgentID: string;
	private supportedDtmfChars: string;
	private event: ConnectEvent;

	constructor(
		logger: Logger,
		apiClient: CrestaAPIClient,
		domain: string,
		customerID: string,
		profileID: string,
		virtualAgentID: string,
		supportedDtmfChars: string,
		event: ConnectEvent,
	) {
		this.logger = logger;
		this.apiClient = apiClient;
		this.domain = domain;
		this.customerID = customerID;
		this.profileID = profileID;
		this.virtualAgentID = virtualAgentID;
		this.supportedDtmfChars = supportedDtmfChars;
		this.event = event;
	}

	async getPSTNTransferData(signal: AbortSignal): Promise<ConnectResponse> {
		const virtualAgentName = `customers/${this.customerID}/profiles/${this.profileID}/virtualAgents/${this.virtualAgentID}`;
		const url = `${this.domain}/v1/${virtualAgentName}:generatePSTNTransferData`;

		const filteredParameters = copyMap(this.event.Details.Parameters, FilteredKeys);

		// Merge ContactData with parameters as a sub-field of ccaasMetadata
		const ccaasMetadata: Record<string, unknown> = {
			...this.event.Details.ContactData,
			parameters: filteredParameters,
			version: VERSION,
		};

		const payload = {
			callId: this.event.Details.ContactData.ContactId,
			ccaasMetadata,
			supportedDtmfChars: this.supportedDtmfChars,
		};

		this.logger.debugf("Making request to %s with payload: %+v", url, payload);

		const body = await this.apiClient.makeRequest(signal, "POST", url, payload);

		let result: ConnectResponse;
		try {
			result = JSON.parse(new TextDecoder().decode(body)) as ConnectResponse;
		} catch (error) {
			throw new Error(
				`failed to parse JSON response from ${url}: ${error instanceof Error ? error.message : String(error)}`,
			);
		}

		this.logger.debugf("Received response: %+v", result);
		return result;
	}

	async getHandoffData(signal: AbortSignal): Promise<ConnectResponse> {
		const url = `${this.domain}/v1/customers/${this.customerID}/profiles/${this.profileID}/handoffs:fetchAIAgentHandoff`;
		const payload = {
			correlationId: this.event.Details.ContactData.ContactId,
		};

		this.logger.debugf("Making request to %s with payload: %+v", url, payload);

		const body = await this.apiClient.makeRequest(signal, "POST", url, payload);

		const decodedBody = new TextDecoder().decode(body);
		const parsed = JSON.parse(decodedBody) as unknown;

		if (
			!parsed ||
			typeof parsed !== "object" ||
			!("handoff" in parsed) ||
			!parsed.handoff ||
			typeof parsed.handoff !== "object"
		) {
			throw new Error(`invalid handoff response structure: ${JSON.stringify(parsed)}`);
		}

		const result = parsed as FetchAIAgentHandoffResponse;
		this.logger.debugf("Received response: %+v", result);

		// Validate required handoff fields exist and are the expected types
		const handoff = result.handoff;
		if (
			typeof handoff.conversation !== "string" ||
			typeof handoff.conversationCorrelationId !== "string" ||
			typeof handoff.summary !== "string" ||
			typeof handoff.transferTarget !== "string"
		) {
			throw new Error(
				`invalid handoff response: missing or invalid required fields (conversation, conversationCorrelationId, summary, transferTarget)`,
			);
		}

		return {
			handoff_conversation: handoff.conversation,
			handoff_conversationCorrelationId: handoff.conversationCorrelationId,
			handoff_summary: handoff.summary,
			handoff_transferTarget: handoff.transferTarget,
		};
	}
}
