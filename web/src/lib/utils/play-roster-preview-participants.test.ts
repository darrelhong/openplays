import { describe, expect, it } from 'vitest';
import { getPlayRosterPreview } from './play-roster-preview';

describe('getPlayRosterPreview participant previews', () => {
	it('uses confirmed participant preview rows when available', () => {
		const preview = getPlayRosterPreview({
			listing_type: 'play',
			created_by: 'user-1',
			host_name: 'Host',
			max_players: 4,
			slots_left: 3,
			participant_preview: [
				{
					id: 1,
					photo_url: 'https://example.com/player.png',
					rating_code: '4.2',
					is_guest: false
				},
				{
					id: 2,
					display_name: 'Guest One',
					rating_code: '3.5',
					is_guest: true
				}
			]
		});

		expect(preview).toMatchObject({
			occupiedSlots: 2,
			openSlots: 2,
			label: '2/4 joined'
		});
		expect(preview?.slots).toEqual([
			{
				kind: 'known',
				name: 'Player',
				photoUrl: 'https://example.com/player.png',
				ratingCode: '4.2'
			},
			{
				kind: 'known',
				name: 'Guest One',
				photoUrl: undefined,
				ratingCode: '3.5'
			},
			{ kind: 'open', label: 'Open slot 1' },
			{ kind: 'open', label: 'Open slot 2' }
		]);
	});
});
