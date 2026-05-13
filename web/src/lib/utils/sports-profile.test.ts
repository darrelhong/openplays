import { describe, expect, it } from 'vitest';
import {
	formatTennisLevel,
	PROFILE_TENNIS_LEVELS,
	sportsProfileFromFormData,
	sportsProfileFromLevels,
	sportsProfileToForm,
	tennisLevelNumber
} from './sports-profile';

describe('sportsProfileToForm', () => {
	it('maps an API sports profile into flat form values', () => {
		expect(
			sportsProfileToForm({
				badminton: { level: 'HB' },
				tennis: { level: '3.5' }
			})
		).toEqual({
			activeSports: ['badminton', 'tennis'],
			badmintonLevel: 'HB',
			tennisLevel: '3.5'
		});
	});

	it('uses empty strings when profile values are missing', () => {
		expect(sportsProfileToForm(undefined)).toEqual({
			activeSports: [],
			badmintonLevel: '',
			tennisLevel: ''
		});
	});
});

describe('sportsProfileFromLevels', () => {
	it('builds the typed API profile and trims selected levels', () => {
		expect(
			sportsProfileFromLevels({
				activeSports: ['badminton', 'tennis'],
				badmintonLevel: ' LI ',
				tennisLevel: '3.5'
			})
		).toEqual({
			badminton: { level: 'LI' },
			tennis: { level: '3.5' }
		});
	});

	it('returns an empty profile object so users can clear existing ratings', () => {
		expect(
			sportsProfileFromLevels({
				activeSports: ['badminton', 'tennis'],
				badmintonLevel: '',
				tennisLevel: '   '
			})
		).toEqual({});
	});
});

describe('sportsProfileFromFormData', () => {
	it('reads profile levels from profile form fields', () => {
		const formData = new FormData();
		formData.set('badminton_level', 'MI');
		formData.set('tennis_level', '4.0');

		expect(sportsProfileFromFormData(formData)).toEqual({
			badminton: { level: 'MI' },
			tennis: { level: '4.0' }
		});
	});
});

describe('tennisLevelNumber', () => {
	it('normalizes tennis ratings to one decimal between 1.0 and 7.0', () => {
		expect(tennisLevelNumber('3.44')).toBe(3.4);
		expect(tennisLevelNumber('3.45')).toBe(3.5);
		expect(tennisLevelNumber('0.5')).toBe(1);
		expect(tennisLevelNumber(8)).toBe(7);
	});

	it('formats ratings for the API payload', () => {
		expect(formatTennisLevel(4)).toBe('4.0');
		expect(formatTennisLevel('4.26')).toBe('4.3');
		expect(formatTennisLevel('wat')).toBe('3.0');
	});
});

describe('PROFILE_TENNIS_LEVELS', () => {
	it('provides every 0.1 tennis rating from 1.0 to 7.0', () => {
		expect(PROFILE_TENNIS_LEVELS).toHaveLength(61);
		expect(PROFILE_TENNIS_LEVELS[0]).toEqual({
			value: '1.0',
			label: '1.0 - Beginner'
		});
		expect(PROFILE_TENNIS_LEVELS[24]).toEqual({
			value: '3.4',
			label: '3.4'
		});
		expect(PROFILE_TENNIS_LEVELS.at(-1)).toEqual({
			value: '7.0',
			label: '7.0 - Elite'
		});
	});
});
