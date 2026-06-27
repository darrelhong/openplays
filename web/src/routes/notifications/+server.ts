import { API_BASE_URL } from '$env/static/private';
import { error, json, type RequestHandler } from '@sveltejs/kit';

export const GET: RequestHandler = async ({ cookies, fetch, url }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Not authenticated');
	}

	const rawLimit = Number(url.searchParams.get('limit') ?? '50');
	const limit = Number.isSafeInteger(rawLimit) && rawLimit > 0 ? Math.min(rawLimit, 50) : 50;
	const response = await fetch(`${API_BASE_URL}/api/notifications/?limit=${limit}`, {
		headers: { Cookie: `session=${sessionToken}` }
	});
	const body = await response.json().catch(() => null);

	if (!response.ok) {
		error(response.status, body?.detail ?? 'Failed to get notifications');
	}

	return json(body);
};
