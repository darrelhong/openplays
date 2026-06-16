import { error as httpError, json } from '@sveltejs/kit';
import { api } from '$lib/api/client';
import type { RequestHandler } from './$types';
import type { Cookies } from '@sveltejs/kit';

function sessionCookie(cookies: Cookies) {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		throw httpError(401, 'Not authenticated');
	}
	return `session=${sessionToken}`;
}

export const GET: RequestHandler = async ({ url, cookies }) => {
	const q = url.searchParams.get('q') ?? '';
	const sessionToken = url.searchParams.get('session_token') ?? undefined;
	const limit = Number(url.searchParams.get('limit') ?? 5);

	const { data, error } = await api.GET('/api/venues/search', {
		headers: { Cookie: sessionCookie(cookies) },
		params: {
			query: {
				q,
				session_token: sessionToken,
				limit: Number.isFinite(limit) ? limit : 5
			}
		}
	});

	if (error) {
		throw httpError(error.status ?? 500, error.detail ?? 'Failed to search venues');
	}
	return json({ items: data?.items ?? [] });
};

export const POST: RequestHandler = async ({ request, cookies }) => {
	const body = await request.json();
	const { data, error } = await api.POST('/api/venues/resolve-google', {
		headers: { Cookie: sessionCookie(cookies) },
		body
	});

	if (error) {
		throw httpError(error.status ?? 500, error.detail ?? 'Failed to resolve venue');
	}
	return json(data);
};
