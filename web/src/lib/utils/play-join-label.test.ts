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

	it('shows Join waitlist when the play is full', () => {
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
		).toBe('Join waitlist');
	});

	it('shows Join waitlist when the user level is outside the play range', () => {
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
		).toBe('Join waitlist');

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
		).toBe('Join waitlist');
	});

	it('shows Join waitlist when the user does not have a known level for the sport', () => {
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
		).toBe('Join waitlist');
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
		).toBe('Join waitlist');

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

	it('treats unsupported sport profiles as waitlist-only', () => {
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
