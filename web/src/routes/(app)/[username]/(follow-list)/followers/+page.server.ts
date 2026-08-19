import { api } from '$lib/api/client';
import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, cookies, url }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Not authenticated');
	}

	const cursor = url.searchParams.get('cursor');
	const limit = url.searchParams.get('limit');
	const response = await api.GET('/api/users/{username}/followers', {
		headers: { Cookie: `session=${sessionToken}` },
		params: {
			path: { username: params.username },
			query: { cursor: cursor || undefined, limit: limit ? Number(limit) : undefined }
		}
	});

	if (response.error?.status === 403) {
		return { list: null, forbidden: true };
	}
	if (response.error) {
		error(response.error.status ?? 500, response.error.detail ?? 'Failed to fetch followers');
	}
	if (!response.data) {
		error(500, 'Followers were not returned');
	}

	return { list: response.data, forbidden: false };
};
