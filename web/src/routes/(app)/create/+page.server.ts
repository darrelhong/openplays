import { redirect } from '@sveltejs/kit';
import { api } from '$lib/api/client';
import {
	feeToCents,
	initialSlotsLeft,
	isPositiveInteger,
	optionalNumber
} from '$lib/utils/create-play';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ parent }) => {
	const { user } = await parent();
	const venuesResponse = await api.GET('/api/venues/').catch(() => null);
	return {
		user: user!,
		venues: venuesResponse?.data?.items ?? []
	};
};

export const actions: Actions = {
	default: async ({ request, cookies }) => {
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return { error: 'Not authenticated' };
		}

		const formData = await request.formData();

		const sport = formData.get('sport') as string;
		const venue = formData.get('venue') as string;
		const startsAt = formData.get('starts_at') as string;
		const durationStr = formData.get('duration_minutes') as string;
		const timezone = (formData.get('timezone') as string) || 'Asia/Singapore';
		const currency = (formData.get('currency') as string) || 'SGD';

		// Optional fields
		const gameType = formData.get('game_type') as string | null;
		const levelMin = formData.get('level_min') as string | null;
		const levelMax = formData.get('level_max') as string | null;
		const feeStr = formData.get('fee') as string | null;
		const maxPlayersStr = formData.get('max_players') as string | null;
		const courtsStr = formData.get('courts') as string | null;

		// Validation
		if (!sport) return { error: 'Sport is required' };
		if (!venue?.trim()) return { error: 'Venue is required' };
		if (!startsAt) return { error: 'Start time is required' };
		if (!durationStr) return { error: 'Duration is required' };

		// Validate start time is in the future
		const startsAtDate = new Date(startsAt);
		if (isNaN(startsAtDate.getTime())) return { error: 'Invalid start time' };
		if (startsAtDate.getTime() <= Date.now()) return { error: 'Start time must be in the future' };

		const durationMinutes = optionalNumber(durationStr) ?? 0;
		const fee = feeToCents(feeStr);
		const maxPlayers = optionalNumber(maxPlayersStr);
		const courts = optionalNumber(courtsStr);

		if (!isPositiveInteger(maxPlayers)) {
			return { error: 'Max players is required' };
		}

		const slotsLeft = initialSlotsLeft(maxPlayers);

		const createPlayBody = {
			sport: sport as 'badminton' | 'tennis' | 'football' | 'pickleball',
			venue: venue.trim(),
			starts_at: startsAt,
			duration_minutes: durationMinutes,
			timezone,
			currency,
			game_type: (gameType || undefined) as 'doubles' | 'singles' | 'mixed_doubles' | undefined,
			level_min: levelMin || undefined,
			level_max: levelMax || undefined,
			fee,
			max_players: maxPlayers,
			slots_left: slotsLeft,
			courts
		};

		const { error } = await api.POST('/api/plays/', {
			headers: { Cookie: `session=${sessionToken}` },
			body: createPlayBody
		});

		if (error) {
			return { error: error.detail ?? 'Failed to create game' };
		}

		redirect(303, '/');
	}
};
