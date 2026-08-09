import { error as httpError, json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import type { operations } from '$lib/api/types.gen';
import { api } from '$lib/api/client';

type SearchQuery = NonNullable<operations['search-users']['parameters']['query']>;

function sessionCookie(cookies: Parameters<RequestHandler>[0]['cookies']) {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		throw httpError(401, 'Not authenticated');
	}
	return `session=${sessionToken}`;
}

export const GET: RequestHandler = async ({ url, cookies, params }) => {
	const rawLimit = Number(url.searchParams.get('limit') ?? 10);
	const limit = Number.isFinite(rawLimit) ? Math.min(Math.max(rawLimit, 1), 25) : 10;
	const sport = url.searchParams.get('sport') as SearchQuery['sport'];

	const { data, error } = await api.GET('/api/users/search', {
		headers: { Cookie: sessionCookie(cookies) },
		params: {
			query: {
				q: url.searchParams.get('q') ?? '',
				sport,
				limit,
				exclude_play_id: params.id
			}
		}
	});

	if (error) {
		throw httpError(error.status ?? 500, error.detail ?? 'Failed to search users');
	}
	return json({ items: data?.items ?? [] });
};
