import { api } from '$lib/api/client';
import { favouritePlay, unfavouritePlay } from '$lib/server/play-favourite-actions';
import { error } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import type { operations } from '$lib/api/types.gen';

type Sport = NonNullable<operations['list-plays']['parameters']['query']>['sport'];
type Availability = NonNullable<operations['list-plays']['parameters']['query']>['availability'];

export const load: PageServerLoad = async ({ url, cookies }) => {
	const sport = url.searchParams.get('sport') || undefined;
	const source = url.searchParams.get('source') || undefined;
	const availability = url.searchParams.get('availability') || undefined;
	const cursor = url.searchParams.get('cursor');
	const limit = url.searchParams.get('limit');
	const lat = url.searchParams.get('lat');
	const lng = url.searchParams.get('lng');
	const startsAfter = url.searchParams.get('starts_after');
	const startsBefore = url.searchParams.get('starts_before');
	const timezone = url.searchParams.get('timezone');
	const levelMin = url.searchParams.get('level_min');
	const levelMax = url.searchParams.get('level_max');
	const sessionToken = cookies.get('session');

	const [playsResponse, venuesResponse] = await Promise.all([
		api
			.GET('/api/plays/', {
				headers: sessionToken ? { Cookie: `session=${sessionToken}` } : undefined,
				params: {
					query: {
						sport: sport as Sport,
						source: source as 'user' | 'telegram' | undefined,
						availability: availability as Availability,
						cursor: cursor || undefined,
						limit: limit ? Number(limit) : undefined,
						lat: lat ? Number(lat) : undefined,
						lng: lng ? Number(lng) : undefined,
						starts_after: startsAfter || undefined,
						starts_before: startsBefore || undefined,
						timezone: timezone || undefined,
						level_min: levelMin || undefined,
						level_max: levelMax || undefined
					}
				}
			})
			.catch(() => null),
		api.GET('/api/venues/').catch(() => null)
	]);

	if (!playsResponse) {
		error(503, 'API is currently unavailable');
	}
	if (playsResponse.error) {
		error(playsResponse.error.status ?? 500, playsResponse.error.detail ?? 'Failed to fetch plays');
	}

	return {
		plays: playsResponse.data,
		venues: venuesResponse?.data?.items ?? []
	};
};

export const actions: Actions = {
	favourite: favouritePlay,
	unfavourite: unfavouritePlay
};
