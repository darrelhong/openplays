import { afterEach, describe, expect, it, vi } from 'vitest';
import { notificationFromPushPayload } from './optimistic-notification';

const temporaryUUID = '00000000-0000-4000-8000-000000000001';

describe('notificationFromPushPayload', () => {
	afterEach(() => {
		vi.useRealTimers();
		vi.restoreAllMocks();
	});

	it('builds an unread temporary notification from a push payload', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-06-27T03:04:05Z'));
		vi.spyOn(crypto, 'randomUUID').mockReturnValue(temporaryUUID);

		expect(
			notificationFromPushPayload({
				title: 'Friday Friendly',
				body: 'Seed Advanced joined the game',
				url: '/play/play-1',
				kind: 'play.player_joined',
				play_id: 'play-1'
			})
		).toEqual({
			id: `push:${temporaryUUID}`,
			title: 'Friday Friendly',
			body: 'Seed Advanced joined the game',
			url: '/play/play-1',
			kind: 'play.player_joined',
			play_id: 'play-1',
			created_at: '2026-06-27T03:04:05.000Z'
		});
	});

	it('falls back to OpenPlays when the payload title is missing', () => {
		vi.spyOn(crypto, 'randomUUID').mockReturnValue(temporaryUUID);

		expect(notificationFromPushPayload({ body: 'A game changed' })).toMatchObject({
			id: `push:${temporaryUUID}`,
			title: 'OpenPlays',
			body: 'A game changed'
		});
	});

	it('ignores malformed push payloads', () => {
		expect(notificationFromPushPayload(null)).toBeNull();
		expect(notificationFromPushPayload('not an object')).toBeNull();
	});
});
