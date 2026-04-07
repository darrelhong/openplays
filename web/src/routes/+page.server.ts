import { api } from '$lib/api/client';
import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import type { operations } from '$lib/api/types.gen';

type Sport = NonNullable<operations['list-plays']['parameters']['query']>['sport'];

export const load: PageServerLoad = async ({ url }) => {
	const sport = url.searchParams.get('sport') || undefined;
	const cursor = url.searchParams.get('cursor');
	const limit = url.searchParams.get('limit');

	let response;
	try {
		response = await api.GET('/api/plays/', {
			params: {
				query: {
					sport: sport as Sport,
					cursor: cursor ? Number(cursor) : undefined,
					limit: limit ? Number(limit) : undefined
				}
			}
		});
	} catch {
		// Network error (API down, connection refused, timeout)
		error(503, 'API is currently unavailable');
	}

	if (response.error) {
		error(response.error.status ?? 500, response.error.detail ?? 'Failed to fetch plays');
	}

	return { plays: response.data };
};
