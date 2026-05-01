import { api } from '$lib/api/client';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ parent }) => {
	const { user } = await parent();
	return { user: user! };
};

export const actions: Actions = {
	update: async ({ request, cookies }) => {
		const formData = await request.formData();
		const displayName = (formData.get('display_name') as string)?.trim();
		const username = (formData.get('username') as string)?.trim();

		if (!displayName) {
			return { error: 'Display name is required' };
		}

		if (username === '') {
			return { error: 'Username cannot be empty' };
		}

		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return { error: 'Not authenticated' };
		}

		const { data, error } = await api.PATCH('/api/me/', {
			headers: { Cookie: `session=${sessionToken}` },
			body: {
				display_name: displayName,
				username: username || undefined
			}
		});

		if (error) {
			return { error: error.detail ?? 'Failed to update profile' };
		}

		return { success: true, user: data };
	}
};
