import { api } from '$lib/api/client';
import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, cookies, locals }) => {
	const sessionToken = cookies.get('session');
	if (!sessionToken || !locals.user) {
		error(401, 'Not authenticated');
	}

	const [conversationResponse, messageResponse] = await Promise.all([
		api.GET('/api/chat/conversations', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { query: { limit: 50 } }
		}),
		api.GET('/api/chat/conversations/{id}/messages', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { id: params.id }, query: { limit: 50, before_id: 0 } }
		})
	]);

	if (conversationResponse.error) {
		error(
			conversationResponse.error.status ?? 500,
			conversationResponse.error.detail ?? 'Failed to fetch conversations'
		);
	}
	if (messageResponse.error) {
		error(
			messageResponse.error.status ?? 500,
			messageResponse.error.detail ?? 'Failed to fetch messages'
		);
	}
	const messages = messageResponse.data?.items ?? [];
	const lastMessage = messages.at(-1);
	if (lastMessage) {
		await api.POST('/api/chat/conversations/{id}/read', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { id: params.id } },
			body: { last_read_message_id: lastMessage.id }
		});
	}

	// The list was fetched before the conversation was marked read
	const conversations = (conversationResponse.data?.items ?? []).map((conversation) =>
		conversation.id === params.id ? { ...conversation, unread_count: 0 } : conversation
	);

	return {
		conversations,
		messages,
		selectedConversationId: params.id,
		user: locals.user
	};
};

export const actions: Actions = {
	send: async ({ params, cookies, request }) => {
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { error: 'Sign in to send messages' });
		}

		const formData = await request.formData();
		const body = String(formData.get('body') ?? '').trim();
		if (!body) {
			return fail(400, { error: 'Message cannot be empty' });
		}

		const { error: apiError } = await api.POST('/api/chat/conversations/{id}/messages', {
			headers: { Cookie: `session=${sessionToken}` },
			params: { path: { id: params.id } },
			body: { body }
		});

		if (apiError) {
			return fail(apiError.status ?? 500, {
				error: apiError.detail ?? 'Failed to send message'
			});
		}

		redirect(303, `/chat/${params.id}`);
	},
	delete: async ({ params, cookies, request }) => {
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { error: 'Sign in to delete messages' });
		}

		const formData = await request.formData();
		const messageID = Number(formData.get('message_id'));
		if (!Number.isSafeInteger(messageID) || messageID <= 0) {
			return fail(400, { error: 'Invalid message' });
		}

		const { error: apiError } = await api.DELETE(
			'/api/chat/conversations/{id}/messages/{messageID}',
			{
				headers: { Cookie: `session=${sessionToken}` },
				params: { path: { id: params.id, messageID } }
			}
		);

		if (apiError) {
			return fail(apiError.status ?? 500, {
				error: apiError.detail ?? 'Failed to delete message'
			});
		}

		redirect(303, `/chat/${params.id}`);
	}
};
