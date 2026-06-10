import { api } from '$lib/api/client';
import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url, cookies }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Not authenticated');
	}

	const cursor = url.searchParams.get('cursor');
	const limit = url.searchParams.get('limit');

	const playsResponse = await api
		.GET('/api/me/plays', {
			headers: { Cookie: `session=${sessionToken}` },
			params: {
				query: {
					cursor: cursor || undefined,
					limit: limit ? Number(limit) : undefined
				}
			}
		})
		.catch(() => null);

	if (!playsResponse) {
		error(503, 'API is currently unavailable');
	}
	if (playsResponse.error) {
		error(playsResponse.error.status ?? 500, playsResponse.error.detail ?? 'Failed to fetch plays');
	}

	return {
		plays: playsResponse.data
	};
};
