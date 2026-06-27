import { afterEach, describe, expect, it, vi } from 'vitest';
import { fetchNotifications, markNotificationsRead } from './notification-api';
import type { UserNotification } from './types';

const originalFetch = globalThis.fetch;

describe('notification-api', () => {
	afterEach(() => {
		globalThis.fetch = originalFetch;
		vi.restoreAllMocks();
	});

	it('fetches notifications from the feed endpoint', async () => {
		const notification: UserNotification = {
			id: 'notification-1',
			title: 'Friday Friendly',
			body: 'Seed Advanced joined the game',
			created_at: '2026-06-27T02:00:00Z'
		};
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ notifications: [notification] }), {
				status: 200,
				headers: { 'content-type': 'application/json' }
			})
		);
		globalThis.fetch = fetchMock;

		await expect(fetchNotifications([])).resolves.toEqual([notification]);
		expect(fetchMock).toHaveBeenCalledWith('/notifications?limit=50');
	});

	it('returns the fallback notifications when fetching fails', async () => {
		const fallback: UserNotification[] = [
			{ id: 'existing', title: 'Existing', created_at: '2026-06-27T02:00:00Z' }
		];
		globalThis.fetch = vi.fn().mockResolvedValue(new Response('{}', { status: 500 }));

		await expect(fetchNotifications(fallback)).resolves.toBe(fallback);
	});

	it('marks selected notifications read and returns the client read timestamp', async () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-06-27T03:04:05Z'));
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		globalThis.fetch = fetchMock;

		await expect(markNotificationsRead(['notification-1', 'notification-2'])).resolves.toBe(
			'2026-06-27T03:04:05.000Z'
		);
		expect(fetchMock).toHaveBeenCalledWith('/notifications/read', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ ids: ['notification-1', 'notification-2'] })
		});
		vi.useRealTimers();
	});

	it('skips the mark-read request when there are no IDs', async () => {
		const fetchMock = vi.fn();
		globalThis.fetch = fetchMock;

		await expect(markNotificationsRead([])).resolves.toBeNull();
		expect(fetchMock).not.toHaveBeenCalled();
	});
});
