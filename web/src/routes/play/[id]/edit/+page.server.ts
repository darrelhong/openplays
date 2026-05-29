import { api } from '$lib/api/client';
import {
	buildRFC3339,
	feeToCents,
	isPositiveInteger,
	optionalNumber
} from '$lib/utils/create-play';
import { error, fail, redirect } from '@sveltejs/kit';
import type { components } from '$lib/api/types.gen';
import type { Actions, PageServerLoad } from './$types';

type UpdatePlayBody = components['schemas']['UpdatePlayInputBody'];
type UpdatePlayValues = {
	date: string;
	start_time: string;
	duration_minutes: string;
	timezone: string;
	tz_offset: string;
	game_type: string;
	level_min: string;
	level_max: string;
	fee: string;
	max_players: string;
	courts: string;
};

export const load: PageServerLoad = async ({ params, cookies, locals }) => {
	const id = params.id;
	const sessionToken = cookies.get('session');
	if (!sessionToken || !locals.user) {
		redirect(303, '/login');
	}

	const selectedPlayResponse = await api
		.GET('/api/plays/{id}', {
			headers: { Cookie: `session=${sessionToken}` },
			params: {
				path: { id }
			}
		})
		.catch(() => null);

	if (!selectedPlayResponse) {
		error(503, 'API is currently unavailable');
	}
	if (selectedPlayResponse.error) {
		error(
			selectedPlayResponse.error.status ?? 500,
			selectedPlayResponse.error.detail ?? 'Failed to fetch play'
		);
	}
	if (!selectedPlayResponse.data.can_manage || selectedPlayResponse.data.cancelled_at) {
		redirect(303, `/play/${id}`);
	}

	return {
		play: selectedPlayResponse.data,
		user: locals.user
	};
};

function stringValue(formData: FormData, key: keyof UpdatePlayValues) {
	const value = formData.get(key);
	return typeof value === 'string' ? value : '';
}

function updatePlayValues(formData: FormData): UpdatePlayValues {
	return {
		date: stringValue(formData, 'date'),
		start_time: stringValue(formData, 'start_time'),
		duration_minutes: stringValue(formData, 'duration_minutes'),
		timezone: stringValue(formData, 'timezone') || 'Asia/Singapore',
		tz_offset: stringValue(formData, 'tz_offset') || '+08:00',
		game_type: stringValue(formData, 'game_type'),
		level_min: stringValue(formData, 'level_min'),
		level_max: stringValue(formData, 'level_max'),
		fee: stringValue(formData, 'fee'),
		max_players: stringValue(formData, 'max_players'),
		courts: stringValue(formData, 'courts')
	};
}

function failUpdate(status: number, error: string, values: UpdatePlayValues) {
	return fail(status, { intent: 'update' as const, error, values });
}

export const actions: Actions = {
	update: async ({ params, request, cookies }) => {
		const id = params.id;
		const formData = await request.formData();
		const values = updatePlayValues(formData);
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return failUpdate(401, 'Sign in to manage this game', values);
		}

		const startsAt = buildRFC3339(values.date, values.start_time, values.tz_offset);
		if (!values.date) return failUpdate(400, 'Date is required', values);
		if (!values.start_time) return failUpdate(400, 'Start time is required', values);
		if (!startsAt) return failUpdate(400, 'Start time is required', values);

		const startsAtDate = new Date(startsAt);
		if (isNaN(startsAtDate.getTime())) return failUpdate(400, 'Invalid start time', values);
		if (startsAtDate.getTime() <= Date.now()) {
			return failUpdate(400, 'Start time must be in the future', values);
		}

		const durationMinutes = optionalNumber(values.duration_minutes);
		if (
			!durationMinutes ||
			durationMinutes < 15 ||
			durationMinutes > 300 ||
			durationMinutes % 15 !== 0
		) {
			return failUpdate(
				400,
				'Duration must be a 15 minute increment from 15 to 300 minutes',
				values
			);
		}

		const maxPlayers = optionalNumber(values.max_players);
		if (!isPositiveInteger(maxPlayers)) {
			return failUpdate(400, 'Max players is required', values);
		}

		const feeWasCleared = values.fee.trim() === '';
		const fee = feeToCents(values.fee);
		if (!feeWasCleared && fee === undefined) {
			return failUpdate(400, 'Fee must be a valid amount', values);
		}
		if (fee != null && fee < 0) {
			return failUpdate(400, 'Fee must be at least 0', values);
		}

		const courtsWasCleared = values.courts.trim() === '';
		const courts = optionalNumber(values.courts);
		if (!courtsWasCleared && !isPositiveInteger(courts)) {
			return failUpdate(400, 'Courts must be a whole number', values);
		}

		const body: UpdatePlayBody = {
			starts_at: startsAt,
			duration_minutes: durationMinutes,
			timezone: values.timezone,
			game_type: values.game_type as UpdatePlayBody['game_type'],
			level_min: values.level_min,
			level_max: values.level_max,
			fee: feeWasCleared ? undefined : fee,
			fee_clear: feeWasCleared,
			max_players: maxPlayers,
			courts: courtsWasCleared ? undefined : courts,
			courts_clear: courtsWasCleared
		};

		const { error: apiError } = await api.PATCH('/api/plays/{id}', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { id } },
			body
		});
		if (apiError) {
			return failUpdate(apiError.status ?? 500, apiError.detail ?? 'Failed to update game', values);
		}

		redirect(303, `/play/${id}`);
	},
	cancelPlay: async ({ params, cookies }) => {
		const id = params.id;
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { intent: 'cancel' as const, error: 'Sign in to manage this game' });
		}

		const { error: apiError } = await api.DELETE('/api/plays/{id}', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { id } }
		});
		if (apiError) {
			return fail(apiError.status ?? 500, {
				intent: 'cancel' as const,
				error: apiError.detail ?? 'Failed to cancel game'
			});
		}

		redirect(303, `/play/${id}`);
	}
};
