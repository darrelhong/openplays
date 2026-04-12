import { browser } from '$app/environment';
import posthog from 'posthog-js';
import { PUBLIC_POSTHOG_PROJECT_TOKEN } from '$env/static/public';

export const load = async () => {
	if (browser) {
		posthog.init(PUBLIC_POSTHOG_PROJECT_TOKEN, {
			api_host: 'https://us.i.posthog.com',
			defaults: '2026-01-30'
		});
	}

	return;
};
