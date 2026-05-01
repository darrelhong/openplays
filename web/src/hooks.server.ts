import type { Handle } from '@sveltejs/kit';
import { api } from '$lib/api/client';

export const handle: Handle = async ({ event, resolve }) => {
	const sessionToken = event.cookies.get('session');

	if (sessionToken) {
		const { data, response } = await api.GET('/api/auth/me', {
			headers: { Cookie: `session=${sessionToken}` }
		});

		if (data) {
			event.locals.user = data;
		} else {
			event.locals.user = null;
			if (response.status === 401) {
				event.cookies.delete('session', { path: '/' });
			}
		}
	} else {
		event.locals.user = null;
	}

	return resolve(event);
};
