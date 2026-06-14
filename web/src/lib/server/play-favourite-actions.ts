import { api } from '$lib/api/client';
import { fail, redirect, type RequestEvent } from '@sveltejs/kit';

function playIDFrom(event: RequestEvent, formData: FormData): string | null {
	const raw = formData.get('play_id');
	if (typeof raw === 'string' && raw.trim() !== '') {
		return raw;
	}
	return event.params.id ?? null;
}

function redirectBack(event: RequestEvent): never {
	redirect(303, `${event.url.pathname}${event.url.search}`);
}

async function updateFavourite(event: RequestEvent, favourited: boolean) {
	const sessionToken = event.cookies.get('session');
	if (!sessionToken) {
		return fail(401, { error: 'Sign in to update favourites' });
	}

	const playID = playIDFrom(event, await event.request.formData());
	if (!playID) {
		return fail(400, { error: 'Invalid play' });
	}

	const request = favourited
		? api.PUT('/api/plays/{id}/favourite', {
				headers: { Cookie: `session=${sessionToken}` },
				params: { path: { id: playID } }
			})
		: api.DELETE('/api/plays/{id}/favourite', {
				headers: { Cookie: `session=${sessionToken}` },
				params: { path: { id: playID } }
			});

	const { error: apiError } = await request;
	if (apiError) {
		return fail(apiError.status ?? 500, {
			error: apiError.detail ?? 'Failed to update favourites'
		});
	}

	redirectBack(event);
}

export const favouritePlay = (event: RequestEvent) => updateFavourite(event, true);
export const unfavouritePlay = (event: RequestEvent) => updateFavourite(event, false);
