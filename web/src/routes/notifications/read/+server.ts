import { API_BASE_URL } from '$env/static/private';
import { error, json, type RequestHandler } from '@sveltejs/kit';

export const POST: RequestHandler = async ({ cookies, fetch, request }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Not authenticated');
	}

	const body = await request.json().catch(() => ({ ids: [] }));
	const response = await fetch(`${API_BASE_URL}/api/notifications/read`, {
		method: 'POST',
		headers: {
			Cookie: `session=${sessionToken}`,
			'content-type': 'application/json'
		},
		body: JSON.stringify(body)
	});
	const responseBody = await response.json().catch(() => null);

	if (!response.ok) {
		error(response.status, responseBody?.detail ?? 'Failed to mark notifications read');
	}

	return json({});
};
