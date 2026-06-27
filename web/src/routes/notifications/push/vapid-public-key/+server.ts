import { API_BASE_URL } from '$env/static/private';
import { error, json, type RequestHandler } from '@sveltejs/kit';

export const GET: RequestHandler = async ({ cookies, fetch }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Not authenticated');
	}

	const response = await fetch(`${API_BASE_URL}/api/notifications/push/vapid-public-key`, {
		headers: { Cookie: `session=${sessionToken}` }
	});
	const body = await response.json().catch(() => null);

	if (!response.ok) {
		error(response.status, body?.detail ?? 'Failed to get push public key');
	}

	return json(body);
};
