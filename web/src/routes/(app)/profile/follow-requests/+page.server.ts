import { api } from '$lib/api/client';
import { error, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies, url }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Not authenticated');
	}

	const cursor = url.searchParams.get('cursor');
	const limit = url.searchParams.get('limit');
	const response = await api.GET('/api/me/follow-requests', {
		headers: { Cookie: `session=${sessionToken}` },
		params: {
			query: { cursor: cursor || undefined, limit: limit ? Number(limit) : undefined }
		}
	});

	if (response.error) {
		error(response.error.status ?? 500, response.error.detail ?? 'Failed to fetch follow requests');
	}
	if (!response.data) {
		error(500, 'Follow requests were not returned');
	}

	return { requests: response.data };
};

export const actions: Actions = {
	accept: async ({ request, cookies }) => {
		const userID = String((await request.formData()).get('user_id') ?? '').trim();
		if (!userID) {
			return fail(400, { error: 'Follow request is missing a user' });
		}

		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { error: 'Not authenticated' });
		}

		const { error: apiError } = await api.POST('/api/me/follow-requests/{user_id}/accept', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { user_id: userID } }
		});

		if (apiError) {
			return fail(apiError.status ?? 500, {
				error: apiError.detail ?? 'Failed to accept follow request'
			});
		}

		return { success: true };
	},
	decline: async ({ request, cookies }) => {
		const userID = String((await request.formData()).get('user_id') ?? '').trim();
		if (!userID) {
			return fail(400, { error: 'Follow request is missing a user' });
		}

		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { error: 'Not authenticated' });
		}

		const { error: apiError } = await api.POST('/api/me/follow-requests/{user_id}/decline', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { user_id: userID } }
		});

		if (apiError) {
			return fail(apiError.status ?? 500, {
				error: apiError.detail ?? 'Failed to decline follow request'
			});
		}

		return { success: true };
	}
};
