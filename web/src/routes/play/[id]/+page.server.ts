import { api } from '$lib/api/client';
import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, cookies, locals }) => {
	const id = params.id;

	if (!id) {
		error(404, 'Play not found');
	}

	const sessionToken = cookies.get('session');
	const selectedPlayResponse = await api
		.GET('/api/plays/{id}', {
			headers: sessionToken ? { Cookie: `session=${sessionToken}` } : undefined,
			params: {
				path: {
					id
				}
			}
		})
		.catch(() => null);

	if (!selectedPlayResponse) {
		error(503, 'API is currently unavailable');
	}
	if (selectedPlayResponse.error) {
		error(
			selectedPlayResponse.error.status ?? 500,
			selectedPlayResponse.error.detail ?? 'Failed to fetch play'
		);
	}

	return {
		play: selectedPlayResponse.data,
		user: locals.user
	};
};

export const actions: Actions = {
	join: async ({ params, cookies }) => {
		const id = params.id;
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { error: 'Sign in to join this game' });
		}

		const { error: apiError } = await api.POST('/api/plays/{id}/join', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { id } }
		});
		if (apiError) {
			return fail(apiError.status ?? 500, {
				error: apiError.detail ?? 'Failed to join game'
			});
		}

		redirect(303, `/play/${id}`);
	},
	leave: async ({ params, cookies }) => {
		const id = params.id;
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { error: 'Sign in to update your roster status' });
		}

		const { error: apiError } = await api.DELETE('/api/plays/{id}/participants/me', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { id } }
		});
		if (apiError) {
			return fail(apiError.status ?? 500, {
				error: apiError.detail ?? 'Failed to leave game'
			});
		}

		redirect(303, `/play/${id}`);
	}
};
