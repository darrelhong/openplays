import type { UserNotification } from './types';

export async function fetchNotifications(fallback: UserNotification[]) {
	try {
		const response = await fetch('/notifications?limit=50');
		if (!response.ok) {
			return fallback;
		}
		const body = (await response.json()) as { notifications?: UserNotification[] | null };
		return body.notifications ?? [];
	} catch {
		return fallback;
	}
}

export async function markNotificationsRead(ids: string[]) {
	if (ids.length === 0) {
		return null;
	}

	const response = await fetch('/notifications/read', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify({ ids })
	});
	if (!response.ok) {
		return null;
	}

	return new Date().toISOString();
}
