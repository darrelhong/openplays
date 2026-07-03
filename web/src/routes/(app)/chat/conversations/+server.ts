import { api } from '$lib/api/client';
import { error, json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ cookies, url }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Not authenticated');
	}

	const cursor = url.searchParams.get('cursor');
	if (!cursor) {
		error(400, 'Missing cursor');
	}

	const { data, error: apiError } = await api.GET('/api/chat/conversations', {
		headers: { Cookie: `session=${sessionToken}` },
		params: { query: { limit: 50, cursor } }
	});

	if (apiError) {
		error(apiError.status ?? 500, apiError.detail ?? 'Failed to fetch conversations');
	}

	return json({ items: data?.items ?? [], next_cursor: data?.next_cursor ?? null });
};
