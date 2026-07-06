import { api } from '$lib/api/client';
import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, cookies }) => {
	const id = params.id;
	if (!id) {
		error(404, 'Play not found');
	}
	const sessionToken = cookies.get('session');
	if (!sessionToken) {
		error(401, 'Sign in to review your co-players');
	}

	const headers = { Cookie: `session=${sessionToken}` };
	const [playResponse, reviewsResponse] = await Promise.all([
		api.GET('/api/plays/{id}', { headers, params: { path: { id } } }).catch(() => null),
		api.GET('/api/plays/{id}/reviews', { headers, params: { path: { id } } }).catch(() => null)
	]);

	if (!playResponse || !reviewsResponse) {
		error(503, 'API is currently unavailable');
	}
	if (playResponse.error) {
		error(playResponse.error.status ?? 500, playResponse.error.detail ?? 'Failed to fetch play');
	}
	if (reviewsResponse.error) {
		error(
			reviewsResponse.error.status ?? 500,
			reviewsResponse.error.detail ?? 'Failed to fetch reviews'
		);
	}

	const sheet = reviewsResponse.data;
	const reviewee = (sheet.reviewees ?? []).find((r) => r.username === params.username);
	if (!reviewee) {
		error(404, 'Player not found in this game');
	}

	return {
		play: playResponse.data,
		window: sheet.window,
		peerProps: sheet.peer_props ?? [],
		hostProps: sheet.host_props ?? [],
		reviewee
	};
};

export const actions: Actions = {
	review: async ({ params, request, cookies }) => {
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { error: 'Sign in to review your co-players' });
		}

		const formData = await request.formData();
		const revieweeUserID = formData.get('reviewee_user_id');
		if (typeof revieweeUserID !== 'string' || !revieweeUserID) {
			return fail(400, { error: 'Missing reviewee' });
		}

		const ratingRaw = formData.get('rating');
		const rating = typeof ratingRaw === 'string' && ratingRaw !== '' ? Number(ratingRaw) : null;
		const props = formData.getAll('props').filter((p): p is string => typeof p === 'string');
		const shoutoutRaw = formData.get('shoutout');
		const shoutout = typeof shoutoutRaw === 'string' ? shoutoutRaw.trim() : '';

		const { error: apiError } = await api.PUT('/api/plays/{id}/reviews/{revieweeUserID}', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { id: params.id, revieweeUserID } },
			body: {
				rating: rating ?? undefined,
				props,
				shoutout: shoutout || undefined
			}
		});
		if (apiError) {
			return fail(apiError.status ?? 500, {
				error: apiError.detail ?? 'Failed to save review'
			});
		}

		redirect(303, `/play/${params.id}`);
	}
};
