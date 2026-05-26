import { dev } from '$app/environment';
import { env } from '$env/dynamic/private';
import { API_BASE_URL, COOKIE_SECURE } from '$env/static/private';
import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

type SeedUser = {
	id: string;
	displayName: string;
	description: string;
};

const seedUsers: SeedUser[] = [
	{ id: 'seed-host', displayName: 'Seed Host', description: 'Badminton HB, Tennis 3.5' },
	{ id: 'seed-li', displayName: 'Seed Low Intermediate', description: 'Badminton LI' },
	{ id: 'seed-mi', displayName: 'Seed Mid Intermediate', description: 'Badminton MI' },
	{ id: 'seed-advanced', displayName: 'Seed Advanced', description: 'Badminton A' },
	{ id: 'seed-tennis', displayName: 'Seed Tennis 4.0', description: 'Tennis 4.0' },
	{ id: 'seed-norating', displayName: 'Seed No Rating', description: 'No sports profile' }
];

function assertDevAuthEnabled() {
	if (!dev || env.DEV_AUTH_ENABLED !== 'true') {
		error(404, 'Not found');
	}
}

export const load: PageServerLoad = async () => {
	assertDevAuthEnabled();
	return { seedUsers };
};

export const actions: Actions = {
	login: async ({ request, cookies }) => {
		assertDevAuthEnabled();

		const formData = await request.formData();
		const userID = String(formData.get('user_id') ?? '').trim();
		if (!seedUsers.some((user) => user.id === userID)) {
			return fail(400, { error: 'Unknown seed user', userID });
		}

		const response = await fetch(`${API_BASE_URL}/api/dev/login`, {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ user_id: userID })
		});
		if (!response.ok) {
			return fail(response.status, { error: await devLoginError(response), userID });
		}

		const body = (await response.json()) as { session_token?: string };
		if (!body.session_token) {
			return fail(500, { error: 'Dev login response did not include a session token', userID });
		}

		cookies.set('session', body.session_token, {
			path: '/',
			httpOnly: true,
			secure: COOKIE_SECURE !== 'false',
			sameSite: 'lax',
			maxAge: 30 * 24 * 60 * 60
		});

		redirect(303, '/');
	}
};

async function devLoginError(response: Response): Promise<string> {
	try {
		const body = (await response.json()) as { detail?: string; title?: string };
		return body.detail ?? body.title ?? 'Dev login failed';
	} catch {
		return 'Dev login failed';
	}
}
