import { api } from '$lib/api/client';
import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies, locals }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken || !locals.user) {
		error(401, 'Not authenticated');
	}

	const { data, error: apiError } = await api.GET('/api/chat/conversations', {
		headers: { Cookie: `session=${sessionToken}` },
		params: { query: { limit: 50 } }
	});

	if (apiError) {
		error(apiError.status ?? 500, apiError.detail ?? 'Failed to fetch conversations');
	}

	// Conversations are created as soon as a chat is opened; keep them out of
	// the list until someone actually sends a message
	return {
		conversations: (data?.items ?? []).filter((conversation) => conversation.last_message),
		user: locals.user
	};
};
