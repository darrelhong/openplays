import type { components } from '$lib/api/types.gen';
import { BADMINTON_LEVELS, TENNIS_LEVELS, levelIndex } from '$lib/consts/levels';
import type { SelectItem } from '$lib/consts/sports';

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

	const levels = levelsForSport(play.sport);
	const userLevel = userLevelForSport(user?.sports_profile, play.sport);
	if (!levels || !userLevel) {
		return false;
	}

	const userIndex = levelIndex(levels, userLevel);
	if (userIndex < 0) {
		return false;
	}

	const minIndex = play.level_min ? levelIndex(levels, play.level_min) : -1;
	if (minIndex >= 0 && userIndex < minIndex) {
		return false;
	}

	const maxIndex = play.level_max ? levelIndex(levels, play.level_max) : -1;
	if (maxIndex >= 0 && userIndex > maxIndex) {
		return false;
	}

	return true;
}

function levelsForSport(sport: PlayPublic['sport']): SelectItem[] | null {
	switch (sport) {
		case 'badminton':
			return BADMINTON_LEVELS;
		case 'tennis':
			return TENNIS_LEVELS;
		default:
			return null;
	}
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
