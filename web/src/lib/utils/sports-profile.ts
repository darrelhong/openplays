import type { components } from '$lib/api/types.gen';

export type SportsProfile = components['schemas']['SportsProfile'];

export type ProfileSport = 'badminton' | 'tennis';

export const PROFILE_SPORT_OPTIONS: ReadonlyArray<{ value: ProfileSport; label: string }> = [
	{ value: 'badminton', label: 'Badminton' },
	{ value: 'tennis', label: 'Tennis' }
];

const TENNIS_INTERVAL_LABELS = new Map([
	['1.0', '1.0 - Beginner'],
	['2.0', '2.0 - Novice'],
	['3.0', '3.0 - Intermediate'],
	['4.0', '4.0 - Advanced Intermediate'],
	['5.0', '5.0 - Advanced'],
	['6.0', '6.0 - Tournament'],
	['7.0', '7.0 - Elite']
]);

export const PROFILE_TENNIS_LEVELS = Array.from({ length: 61 }, (_, index) => {
	const value = ((10 + index) / 10).toFixed(1);
	return {
		value,
		label: TENNIS_INTERVAL_LABELS.get(value) ?? value
	};
});

export type SportsProfileFormValues = {
	activeSports: ProfileSport[];
	badmintonLevel: string;
	tennisLevel: string;
};

export function sportsProfileToForm(
	profile: SportsProfile | null | undefined
): SportsProfileFormValues {
	const badmintonLevel = profile?.badminton?.level ?? '';
	const tennisLevel = profile?.tennis?.level ?? '';

	return {
		activeSports: [
			...(badmintonLevel ? (['badminton'] as const) : []),
			...(tennisLevel ? (['tennis'] as const) : [])
		],
		badmintonLevel,
		tennisLevel
	};
}

export function sportsProfileFromLevels(values: SportsProfileFormValues): SportsProfile {
	const badmintonLevel = values.badmintonLevel.trim();
	const tennisLevel = values.tennisLevel.trim();
	const profile: SportsProfile = {};

	if (badmintonLevel) {
		profile.badminton = { level: badmintonLevel };
	}
	if (tennisLevel) {
		profile.tennis = { level: tennisLevel };
	}

	return profile;
}

export function sportsProfileFromFormData(formData: FormData): SportsProfile {
	return sportsProfileFromLevels({
		activeSports: [],
		badmintonLevel: stringFormValue(formData.get('badminton_level')),
		tennisLevel: stringFormValue(formData.get('tennis_level'))
	});
}

export function tennisLevelNumber(value: string | number | null | undefined, fallback = 3): number {
	const numeric = typeof value === 'number' ? value : Number(value);
	const safeValue = Number.isFinite(numeric) ? numeric : fallback;
	const clamped = Math.min(7, Math.max(1, safeValue));
	return Math.round(clamped * 10) / 10;
}

export function formatTennisLevel(value: string | number | null | undefined): string {
	return tennisLevelNumber(value).toFixed(1);
}

function stringFormValue(value: FormDataEntryValue | null): string {
	return typeof value === 'string' ? value : '';
}
