import { dev } from '$app/environment';
import { env } from '$env/dynamic/private';
import { error } from '@sveltejs/kit';
import { api } from '$lib/api/client';
import type { PageServerLoad } from './$types';

function assertDevAuthEnabled() {
	if (!dev || env.DEV_AUTH_ENABLED !== 'true') {
		error(404, 'Not found');
	}
}

export const load: PageServerLoad = async () => {
	assertDevAuthEnabled();

	const { data } = await api.GET('/api/dev/email-otp-preview');
	if (!data) {
		error(502, 'Email preview is unavailable');
	}

	return data;
};
