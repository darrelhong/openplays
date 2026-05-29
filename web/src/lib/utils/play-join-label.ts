import type { components } from '$lib/api/types.gen';

type PlayPublic = components['schemas']['PlayPublic'];
type User = components['schemas']['User'];
type SportsProfile = components['schemas']['SportsProfile'];

type PlayJoinLabelInput = Pick<PlayPublic, 'sport' | 'level_min' | 'level_max' | 'slots_left'>;
type UserJoinLabelInput = Pick<User, 'sports_profile'> | null | undefined;

export function getPlayJoinLabel(play: PlayJoinLabelInput, user: UserJoinLabelInput) {
	return canDirectJoin(play, user) ? 'Join game' : 'Join waitlist';
}

export function canDirectJoin(play: PlayJoinLabelInput, user: UserJoinLabelInput) {
	if ((play.slots_left ?? 0) <= 0) {
		return false;
	}

	const userLevel = userLevelForSport(user?.sports_profile, play.sport);
	const userOrd = userLevel ? levelOrd(play.sport, userLevel) : null;
	if (userOrd == null) {
		return false;
	}

	const minOrd = play.level_min ? levelOrd(play.sport, play.level_min) : null;
	if (minOrd != null && userOrd < minOrd) {
		return false;
	}

	const maxOrd = play.level_max ? levelOrd(play.sport, play.level_max) : null;
	if (maxOrd != null && userOrd > maxOrd) {
		return false;
	}

	return true;
}

const BADMINTON_LEVEL_ORD: Record<string, number> = {
	LB: 10,
	MB: 20,
	HB: 30,
	LI: 40,
	MI: 50,
	HI: 60,
	A: 70
};

function levelOrd(sport: PlayPublic['sport'], code: string): number | null {
	const trimmed = code.trim();
	switch (sport) {
		case 'badminton':
			return BADMINTON_LEVEL_ORD[trimmed] ?? null;
		case 'tennis':
			return tennisLevelOrd(trimmed);
		default:
			return null;
	}
}

function tennisLevelOrd(code: string): number | null {
	const level = Number(code);
	if (!Number.isFinite(level) || level < 1 || level > 7) {
		return null;
	}

	const scaled = level * 10;
	const rounded = Math.round(scaled);
	return Math.abs(scaled - rounded) <= 1e-9 ? rounded : null;
}

function userLevelForSport(profile: SportsProfile | undefined, sport: PlayPublic['sport']) {
	switch (sport) {
		case 'badminton':
			return profile?.badminton?.level;
		case 'tennis':
			return profile?.tennis?.level;
		default:
			return undefined;
	}
}
