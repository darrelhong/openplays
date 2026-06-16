import { fail, redirect } from '@sveltejs/kit';
import { api } from '$lib/api/client';
import {
	feeToCents,
	initialSlotsLeft,
	isPositiveInteger,
	optionalNumber
} from '$lib/utils/create-play';
import type { Actions } from './$types';

type CreateFormValues = {
	sport: string;
	venue: string;
	venue_id: string;
	name: string;
	description: string;
	date: string;
	start_time: string;
	starts_at: string;
	duration_minutes: string;
	timezone: string;
	currency: string;
	game_type: string;
	level_min: string;
	level_max: string;
	fee: string;
	max_players: string;
	courts: string;
};

function stringValue(formData: FormData, key: keyof CreateFormValues) {
	const value = formData.get(key);
	return typeof value === 'string' ? value : '';
}

function createFormValues(formData: FormData): CreateFormValues {
	return {
		sport: stringValue(formData, 'sport'),
		venue: stringValue(formData, 'venue'),
		venue_id: stringValue(formData, 'venue_id'),
		name: stringValue(formData, 'name'),
		description: stringValue(formData, 'description'),
		date: stringValue(formData, 'date'),
		start_time: stringValue(formData, 'start_time'),
		starts_at: stringValue(formData, 'starts_at'),
		duration_minutes: stringValue(formData, 'duration_minutes'),
		timezone: stringValue(formData, 'timezone') || 'Asia/Singapore',
		currency: stringValue(formData, 'currency') || 'SGD',
		game_type: stringValue(formData, 'game_type'),
		level_min: stringValue(formData, 'level_min'),
		level_max: stringValue(formData, 'level_max'),
		fee: stringValue(formData, 'fee'),
		max_players: stringValue(formData, 'max_players'),
		courts: stringValue(formData, 'courts')
	};
}

export const actions: Actions = {
	default: async ({ request, cookies }) => {
		const formData = await request.formData();
		const values = createFormValues(formData);
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { error: 'Not authenticated', values });
		}

		const sport = values.sport;
		const venue = values.venue;
		const venueId = optionalNumber(values.venue_id);
		const name = values.name.trim();
		const description = values.description.trim();
		const startsAt = values.starts_at;
		const durationStr = values.duration_minutes;
		const timezone = values.timezone;
		const currency = values.currency;

		// Optional fields
		const gameType = values.game_type;
		const levelMin = values.level_min;
		const levelMax = values.level_max;
		const feeStr = values.fee;
		const maxPlayersStr = values.max_players;
		const courtsStr = values.courts;

		// Validation
		if (!sport) return fail(400, { error: 'Sport is required', values });
		if (!venue?.trim()) return fail(400, { error: 'Venue is required', values });
		if (name.length > 80) return fail(400, { error: 'Name must be at most 80 characters', values });
		if (description.length > 1000) {
			return fail(400, { error: 'Description must be at most 1000 characters', values });
		}
		if (!startsAt) return fail(400, { error: 'Start time is required', values });
		if (!durationStr) return fail(400, { error: 'Duration is required', values });

		// Validate start time is in the future
		const startsAtDate = new Date(startsAt);
		if (isNaN(startsAtDate.getTime())) return fail(400, { error: 'Invalid start time', values });
		if (startsAtDate.getTime() <= Date.now()) {
			return fail(400, { error: 'Start time must be in the future', values });
		}

		const durationMinutes = optionalNumber(durationStr) ?? 0;
		const fee = feeToCents(feeStr);
		const maxPlayers = optionalNumber(maxPlayersStr);
		const courts = optionalNumber(courtsStr);

		if (!isPositiveInteger(maxPlayers)) {
			return fail(400, { error: 'Max players is required', values });
		}

		const slotsLeft = initialSlotsLeft(maxPlayers);

		const createPlayBody = {
			sport: sport as 'badminton' | 'tennis' | 'football' | 'pickleball',
			venue: venue.trim(),
			venue_id: venueId,
			name: name || undefined,
			description: description || undefined,
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

		const { data, error } = await api.POST('/api/plays/', {
			headers: { Cookie: `session=${sessionToken}` },
			body: createPlayBody
		});

		if (error) {
			return fail(error.status ?? 500, { error: error.detail ?? 'Failed to create game', values });
		}
		if (!data?.id) {
			return fail(500, { error: 'Created game was missing an id', values });
		}

		redirect(303, `/play/${data.id}`);
	}
};
