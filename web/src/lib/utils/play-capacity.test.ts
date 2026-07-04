import { describe, expect, it } from 'vitest';
import { minEditableMaxPlayers } from './play-capacity';

describe('minEditableMaxPlayers', () => {
	it('counts confirmed and added players as reserved', () => {
		expect(minEditableMaxPlayers({ confirmed_count: 1, added_count: 3 })).toBe(4);
	});

	it('floors at 1 for an empty roster', () => {
		expect(minEditableMaxPlayers({})).toBe(1);
		expect(minEditableMaxPlayers({ confirmed_count: 0, added_count: 0 })).toBe(1);
	});

	it('falls back to the participant arrays when counts are missing', () => {
		expect(
			minEditableMaxPlayers({
				confirmed_participants: [{ id: 1 }, { id: 2 }],
				added_participants: [{ id: 3 }]
			})
		).toBe(3);
	});

	it('falls back to the preview when confirmed data is missing', () => {
		expect(minEditableMaxPlayers({ participant_preview: [{ id: 1 }] })).toBe(1);
		expect(
			minEditableMaxPlayers({ participant_preview: [{ id: 1 }, { id: 2 }], added_count: 1 })
		).toBe(3);
	});
});
