// Optimistically rendered sends: the composer adds an entry the moment the
// form submits, the message list renders it after the real messages, and the
// composer removes it once the refreshed data (or a failure) comes back
export type PendingMessage = {
	localId: number;
	conversationId: string;
	body: string;
	createdAt: string;
};

let nextLocalId = 1;

export const pendingMessages = $state<PendingMessage[]>([]);

export function addPendingMessage(conversationId: string, body: string) {
	const localId = nextLocalId++;
	// eslint-disable-next-line svelte/prefer-svelte-reactivity -- one-shot timestamp, not reactive state
	const createdAt = new Date().toISOString();
	pendingMessages.push({ localId, conversationId, body, createdAt });
	return localId;
}

export function removePendingMessage(localId: number) {
	const index = pendingMessages.findIndex((message) => message.localId === localId);
	if (index !== -1) {
		pendingMessages.splice(index, 1);
	}
}
