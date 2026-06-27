import type { PushNotificationMessage, UserNotification } from './types';

export function notificationFromPushPayload(payload: unknown): UserNotification | null {
	if (!payload || typeof payload !== 'object') {
		return null;
	}

	const message = payload as PushNotificationMessage;
	const title =
		typeof message.title === 'string' && message.title !== '' ? message.title : 'OpenPlays';
	const optimistic: UserNotification = {
		id: `push:${crypto.randomUUID()}`,
		title,
		created_at: new Date().toISOString()
	};

	if (typeof message.body === 'string') optimistic.body = message.body;
	if (typeof message.url === 'string') optimistic.url = message.url;
	if (typeof message.kind === 'string') optimistic.kind = message.kind;
	if (typeof message.play_id === 'string') optimistic.play_id = message.play_id;

	return optimistic;
}
