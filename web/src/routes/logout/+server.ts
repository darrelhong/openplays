import { redirect } from '@sveltejs/kit';
import { api } from '$lib/api/client';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ cookies }) => {
	const sessionToken = cookies.get('session');

	if (sessionToken) {
		await api
			.POST('/api/auth/logout', {
				headers: { Cookie: `session=${sessionToken}` }
			})
			.catch(() => {});

		cookies.delete('session', { path: '/' });
	}

	redirect(303, '/');
};
