import { api } from '$lib/api/client';
import { error, json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ params, cookies, url }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Not authenticated');
	}

	const beforeID = Number(url.searchParams.get('before_id'));
	if (!Number.isSafeInteger(beforeID) || beforeID <= 0) {
		error(400, 'Invalid cursor');
	}

	const { data, error: apiError } = await api.GET('/api/chat/conversations/{id}/messages', {
		headers: { Cookie: `session=${sessionToken}` },
		params: { path: { id: params.id }, query: { limit: 50, before_id: beforeID } }
	});

	if (apiError) {
		error(apiError.status ?? 500, apiError.detail ?? 'Failed to fetch messages');
	}

	return json({ items: data?.items ?? [] });
};
