import { api } from '$lib/api/client';
import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import type { operations } from '$lib/api/types.gen';

type Sport = NonNullable<operations['list-plays']['parameters']['query']>['sport'];

export const load: PageServerLoad = async ({ url }) => {
	const sport = url.searchParams.get('sport') || undefined;
	const cursor = url.searchParams.get('cursor');
	const limit = url.searchParams.get('limit');
	const lat = url.searchParams.get('lat');
	const lng = url.searchParams.get('lng');
	const startsAfter = url.searchParams.get('starts_after');
	const levelMin = url.searchParams.get('level_min');
	const levelMax = url.searchParams.get('level_max');

	const [playsResponse, venuesResponse] = await Promise.all([
		api
			.GET('/api/plays/', {
				params: {
					query: {
						sport: sport as Sport,
						cursor: cursor || undefined,
						limit: limit ? Number(limit) : undefined,
						lat: lat ? Number(lat) : undefined,
						lng: lng ? Number(lng) : undefined,
						starts_after: startsAfter || undefined,
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
