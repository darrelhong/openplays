import { api } from '$lib/api/client';
import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, cookies, url }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Not authenticated');
	}

	const headers = { Cookie: `session=${sessionToken}` };
	const cursor = url.searchParams.get('cursor');
	const [profileResponse, shoutoutsResponse] = await Promise.all([
		api.GET('/api/users/{username}', {
			headers,
			params: { path: { username: params.username } }
		}),
		api.GET('/api/users/{username}/shoutouts', {
			headers,
			params: {
				path: { username: params.username },
				query: { cursor: cursor || undefined }
			}
		})
	]);

	if (profileResponse.error) {
		error(
			profileResponse.error.status ?? 500,
			profileResponse.error.detail ?? 'Failed to fetch user profile'
		);
	}
	if (!profileResponse.data) {
		error(404, 'User not found');
	}
	if (shoutoutsResponse.error) {
		error(
			shoutoutsResponse.error.status ?? 500,
			shoutoutsResponse.error.detail ?? 'Failed to fetch shoutouts'
		);
	}

	return { profile: profileResponse.data, shoutouts: shoutoutsResponse.data };
};

export const actions: Actions = {
	message: async ({ params, cookies, locals }) => {
		const sessionToken = cookies.get('session');
		if (!sessionToken || !locals.user) {
			return fail(401, { error: 'Sign in to send messages' });
		}

		const profileResponse = await api.GET('/api/users/{username}', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { username: params.username } }
		});

		if (profileResponse.error) {
			return fail(profileResponse.error.status ?? 500, {
				error: profileResponse.error.detail ?? 'Failed to fetch user profile'
			});
		}
		if (!profileResponse.data) {
			return fail(404, { error: 'User not found' });
		}
		if (profileResponse.data.id === locals.user.id) {
			return fail(400, { error: 'You cannot message yourself' });
		}

		const { data, error: apiError } = await api.POST('/api/chat/dms', {
			headers: { Cookie: `session=${sessionToken}` },
			body: { recipient_user_id: profileResponse.data.id }
		});

		if (apiError) {
			return fail(apiError.status ?? 500, {
				error: apiError.detail ?? 'Failed to start conversation'
			});
		}
		if (!data) {
			return fail(500, { error: 'Conversation was not returned' });
		}

		redirect(303, `/chat/${data.id}`);
	}
};
