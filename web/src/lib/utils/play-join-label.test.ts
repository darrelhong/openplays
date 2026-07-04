import { describe, expect, it } from 'vitest';
import { canDirectJoin, getPlayJoinLabel } from './play-join-label';

describe('getPlayJoinLabel', () => {
	it('shows Join game when the user has a matching level and an open slot', () => {
		expect(
			getPlayJoinLabel(
				{
					sport: 'badminton',
					level_min: 'MB',
					level_max: 'HI',
					slots_left: 1
				},
				{ sports_profile: { badminton: { level: 'LI' } } }
			)
		).toBe('Join game');
	});

	it('shows Request to join when the play is full', () => {
		expect(
			getPlayJoinLabel(
				{
					sport: 'badminton',
					level_min: 'MB',
					level_max: 'HI',
					slots_left: 0
				},
				{ sports_profile: { badminton: { level: 'LI' } } }
			)
		).toBe('Request to join');
	});

	it('shows Request to join when the user level is outside the play range', () => {
		expect(
			getPlayJoinLabel(
				{
					sport: 'badminton',
					level_min: 'MB',
					level_max: 'HI',
					slots_left: 1
				},
				{ sports_profile: { badminton: { level: 'LB' } } }
			)
		).toBe('Request to join');

		expect(
			getPlayJoinLabel(
				{
					sport: 'badminton',
					level_min: 'MB',
					level_max: 'HI',
					slots_left: 1
				},
				{ sports_profile: { badminton: { level: 'A' } } }
			)
		).toBe('Request to join');
	});

	it('shows Request to join when the user does not have a known level for the sport', () => {
		expect(
			getPlayJoinLabel(
				{
					sport: 'badminton',
					level_min: 'MB',
					level_max: 'HI',
					slots_left: 1
				},
				{ sports_profile: { tennis: { level: '3.0' } } }
			)
		).toBe('Request to join');
	});

	it('shows Request to join on require-waitlist plays even with a matching level and open slot', () => {
		const play = {
			sport: 'badminton',
			level_min: 'MB',
			level_max: 'HI',
			slots_left: 3,
			require_waitlist: true
		} as const;
		const user = { sports_profile: { badminton: { level: 'LI' } } };

		expect(getPlayJoinLabel(play, user)).toBe('Request to join');
		expect(canDirectJoin(play, user)).toBe(false);
	});
});

describe('canDirectJoin', () => {
	it('allows tennis joins within range', () => {
		expect(
			canDirectJoin(
				{
					sport: 'tennis',
					level_min: '3.0',
					level_max: '4.0',
					slots_left: 2
				},
				{ sports_profile: { tennis: { level: '3.5' } } }
			)
		).toBe(true);
	});

	it('uses fine-grained tennis ordinals like the backend', () => {
		expect(
			getPlayJoinLabel(
				{
					sport: 'tennis',
					level_min: '3.0',
					level_max: '3.4',
					slots_left: 2
				},
				{ sports_profile: { tennis: { level: '3.5' } } }
			)
		).toBe('Request to join');

		expect(
			canDirectJoin(
				{
					sport: 'tennis',
					level_min: '3.0',
					level_max: '3.4',
					slots_left: 2
				},
				{ sports_profile: { tennis: { level: '3.4' } } }
			)
		).toBe(true);
	});

	it('treats unsupported sport profiles as request-only', () => {
		expect(
			canDirectJoin(
				{
					sport: 'football',
					slots_left: 2
				},
				{ sports_profile: { badminton: { level: 'HI' } } }
			)
		).toBe(false);
	});
});
