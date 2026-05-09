import type { LayoutServerLoad } from './$types';
import { env } from '$env/dynamic/private';

export const load: LayoutServerLoad = async ({ locals }) => {
	const showLoginButton = env.FEATURE_LOGIN === 'true';

	return {
		user: locals.user,
		showLoginButton
	};
};
