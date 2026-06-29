import { api } from '$lib/api/client';
import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, cookies }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Not authenticated');
	}

	const { data, error: apiError } = await api.GET('/api/users/{username}', {
		headers: { Cookie: `session=${sessionToken}` },
		params: { path: { username: params.username } }
	});

	if (apiError) {
		error(apiError.status ?? 500, apiError.detail ?? 'Failed to fetch user profile');
	}
	if (!data) {
		error(404, 'User not found');
	}

	return { profile: data };
};
