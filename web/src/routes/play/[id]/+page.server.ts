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

function participantIDFrom(formData: FormData) {
	const raw = formData.get('participant_id');
	const participantID = typeof raw === 'string' ? Number(raw) : NaN;
	return Number.isSafeInteger(participantID) && participantID > 0 ? participantID : null;
}

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
	confirmParticipant: async ({ params, cookies }) => {
		const id = params.id;
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { error: 'Sign in to confirm your spot' });
		}

		const { error: apiError } = await api.POST('/api/plays/{id}/participants/me/confirm', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { id } }
		});
		if (apiError) {
			return fail(apiError.status ?? 500, {
				error: apiError.detail ?? 'Failed to confirm your spot'
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
	},
	acceptParticipant: async ({ params, request, cookies }) => {
		const id = params.id;
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { error: 'Sign in to manage this roster' });
		}

		const participantID = participantIDFrom(await request.formData());
		if (participantID == null) {
			return fail(400, { error: 'Invalid participant' });
		}

		const { error: apiError } = await api.POST(
			'/api/plays/{id}/participants/{participantID}/accept',
			{
				headers: { Cookie: `session=${sessionToken}` },
				params: { path: { id, participantID } }
			}
		);
		if (apiError) {
			return fail(apiError.status ?? 500, {
				error: apiError.detail ?? 'Failed to add player'
			});
		}

		redirect(303, `/play/${id}`);
	},
	removeParticipant: async ({ params, request, cookies }) => {
		const id = params.id;
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { error: 'Sign in to manage this roster' });
		}

		const participantID = participantIDFrom(await request.formData());
		if (participantID == null) {
			return fail(400, { error: 'Invalid participant' });
		}

		const { error: apiError } = await api.DELETE('/api/plays/{id}/participants/{participantID}', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { id, participantID } }
		});
		if (apiError) {
			return fail(apiError.status ?? 500, {
				error: apiError.detail ?? 'Failed to remove player'
			});
		}

		redirect(303, `/play/${id}`);
	}
};
