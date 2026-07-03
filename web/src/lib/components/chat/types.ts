import type { components } from '$lib/api/types.gen';

export type Conversation = components['schemas']['ChatConversationSummary'];
export type Message = components['schemas']['ChatMessagePublic'];

export type Viewer = {
	id: string;
	display_name: string;
	username?: string;
	photo_url?: string;
};
