import { api } from '$lib/api/client';
import { error } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ params, cookies }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Not authenticated');
	}

	const profileResponse = await api.GET('/api/users/{username}', {
		headers: { Cookie: `session=${sessionToken}` },
		params: { path: { username: params.username } }
	});

	if (profileResponse.error) {
		error(
			profileResponse.error.status ?? 500,
			profileResponse.error.detail ?? 'Failed to fetch user profile'
		);
	}
	if (!profileResponse.data) {
		error(404, 'User not found');
	}

	return { profile: profileResponse.data };
};
