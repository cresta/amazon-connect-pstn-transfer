/**
 * Handlers for different API actions matching the Go implementation
 */

import type { CrestaAPIClient } from "./client.js";
import type { Logger } from "./logger.js";
import type { FetchAIAgentHandoffResponse } from "./models.js";
import { type ConnectEvent, type ConnectResponse, copyMap } from "./utils.js";

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
		};

		const payload = {
			callId: this.event.Details.ContactData.ContactId,
			ccaasMetadata,
			supportedDtmfChars: this.supportedDtmfChars,
		};

		this.logger.debugf("Making request to %s with payload: %+v", url, payload);

		const body = await this.apiClient.makeRequest(signal, "POST", url, payload);

		const result: ConnectResponse = JSON.parse(new TextDecoder().decode(body)) as ConnectResponse;

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

		return {
			handoff_conversation: result.handoff.conversation,
			handoff_conversationCorrelationId: result.handoff.conversationCorrelationId,
			handoff_summary: result.handoff.summary,
			handoff_transferTarget: result.handoff.transferTarget,
		};
	}
}
