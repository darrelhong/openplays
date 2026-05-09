import { api } from '$lib/api/client';
import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params }) => {
	const id = Number(params.id);

	if (!Number.isFinite(id) || id <= 0) {
		error(404, 'Play not found');
	}

	const selectedPlayResponse = await api
		.GET('/api/plays/{id}', {
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
		play: selectedPlayResponse.data
	};
};
