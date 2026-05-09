import { redirect } from '@sveltejs/kit';
import { api } from '$lib/api/client';
import { COOKIE_SECURE } from '$env/static/private';
import { env } from '$env/dynamic/private';
import type { Actions, PageServerLoad } from './$types';

function isLoginFeatureEnabled(): boolean {
	return env.FEATURE_LOGIN === 'true';
}

// Redirect to home if already logged in
export const load: PageServerLoad = async ({ locals }) => {
	if (!isLoginFeatureEnabled()) {
		redirect(303, '/');
	}

	if (locals.user) {
		redirect(303, '/');
	}
};

// Auth flow:
// 1. Google GIS button (client-side) opens popup, user signs in with Google
// 2. Google returns an ID token (JWT) to the client via callback
// 3. Client submits token here via hidden form (requestSubmit)
// 4. This action forwards it to the Go API which verifies the JWT and creates a session
// 5. Go API returns session_token in body (its Set-Cookie is lost since SvelteKit is the caller)
// 6. We set the cookie on the browser here — browser never talks to Go directly
export const actions: Actions = {
	google: async ({ request, cookies }) => {
		if (!isLoginFeatureEnabled()) {
			redirect(303, '/');
		}

		const formData = await request.formData();
		const credential = formData.get('credential') as string;

		if (!credential) {
			return { error: 'No credential provided' };
		}

		// Forward Google ID token to Go API for verification + user upsert
		const { data, error } = await api.POST('/api/auth/google', {
			body: { credential }
		});

		if (error) {
			return { error: error.detail ?? 'Login failed' };
		}

		// Set session cookie on the browser (Go API's Set-Cookie doesn't reach browser
		// because SvelteKit server is the HTTP client, not the browser)
		cookies.set('session', data.session_token, {
			path: '/',
			httpOnly: true,
			secure: COOKIE_SECURE !== 'false',
			sameSite: 'lax',
			maxAge: 30 * 24 * 60 * 60 // 30 days
		});

		redirect(303, '/');
	}
};
