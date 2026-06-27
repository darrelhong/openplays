import { API_BASE_URL } from '$env/static/private';
import { error, json, type RequestHandler } from '@sveltejs/kit';

export const POST: RequestHandler = async ({ cookies, fetch, request }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Not authenticated');
	}

	const subscription = await request.json().catch(() => null);
	if (!subscription) {
		error(400, 'Invalid subscription');
	}

	const response = await fetch(`${API_BASE_URL}/api/notifications/push/subscriptions`, {
		method: 'POST',
		headers: {
			Cookie: `session=${sessionToken}`,
			'content-type': 'application/json'
		},
		body: JSON.stringify(subscription)
	});
	const body = await response.json().catch(() => null);

	if (!response.ok) {
		error(response.status, body?.detail ?? 'Failed to save push subscription');
	}

	return json({});
};
