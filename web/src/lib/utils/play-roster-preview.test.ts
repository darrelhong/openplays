import { describe, expect, it } from 'vitest';
import { getPlayRosterPreview } from './play-roster-preview';

describe('getPlayRosterPreview', () => {
	it('returns null for scraped or non-play listings', () => {
		expect(
			getPlayRosterPreview({
				listing_type: 'play',
				host_name: 'Host',
				max_players: 4
			})
		).toBeNull();

		expect(
			getPlayRosterPreview({
				listing_type: 'sell_booking',
				created_by: 'user-1',
				host_name: 'Host',
				max_players: 4
			})
		).toBeNull();
	});

	it('shows the creator and open slots when no slot count has been persisted', () => {
		const preview = getPlayRosterPreview({
			listing_type: 'play',
			created_by: 'user-1',
			host_name: 'Fallback Host',
			creator_display_name: 'Creator',
			creator_photo_url: 'https://example.com/me.png',
			max_players: 4
		});

		expect(preview).toMatchObject({
			totalSlots: 4,
			occupiedSlots: 1,
			openSlots: 3,
			hiddenSlots: 0,
			label: '1/4 joined'
		});
		expect(preview?.slots).toEqual([
			{ kind: 'known', name: 'Creator', photoUrl: 'https://example.com/me.png' },
			{ kind: 'open', label: 'Open slot 1' },
			{ kind: 'open', label: 'Open slot 2' },
			{ kind: 'open', label: 'Open slot 3' }
		]);
	});

	it('infers occupied placeholders from slots_left', () => {
		const preview = getPlayRosterPreview({
			listing_type: 'play',
			created_by: 'user-1',
			host_name: 'Host',
			max_players: 4,
			slots_left: 1
		});

		expect(preview).toMatchObject({
			occupiedSlots: 3,
			openSlots: 1,
			label: '3/4 joined'
		});
		expect(preview?.slots.map((slot) => slot.kind)).toEqual([
			'known',
			'occupied',
			'occupied',
			'open'
		]);
	});

	it('caps visible slots and reports hidden slots', () => {
		const preview = getPlayRosterPreview(
			{
				listing_type: 'play',
				created_by: 'user-1',
				host_name: 'Host',
				max_players: 10
			},
			6
		);

		expect(preview?.slots).toHaveLength(6);
		expect(preview?.hiddenSlots).toBe(4);
	});
});
