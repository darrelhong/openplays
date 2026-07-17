import { api } from '$lib/api/client';
import { profileLinksFromFormData, profileLinksToForm } from '$lib/utils/profile-links';
import { sportsProfileFromFormData, sportsProfileToForm } from '$lib/utils/sports-profile';
import { fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ parent }) => {
	const { user } = await parent();
	return {
		sportsProfileForm: sportsProfileToForm(user.sports_profile),
		bio: user.bio ?? '',
		profileLinksForm: profileLinksToForm(user.profile_links)
	};
};

export const actions: Actions = {
	avatar: async ({ request, cookies }) => {
		const formData = await request.formData();
		const avatar = formData.get('avatar');
		if (!(avatar instanceof File) || avatar.size === 0) {
			return fail(400, { avatarError: 'Choose a profile photo' });
		}
		if (!['image/jpeg', 'image/png'].includes(avatar.type)) {
			return fail(422, { avatarError: 'Profile photo must be JPEG or PNG' });
		}
		if (avatar.size > 5 * 1024 * 1024) {
			return fail(413, { avatarError: 'Profile photo must be 5 MB or smaller' });
		}
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { avatarError: 'Not authenticated' });
		}
		const upload = new FormData();
		upload.set('avatar', avatar);
		const { data, error } = await api.PUT('/api/me/avatar', {
			headers: { Cookie: `session=${sessionToken}` },
			// The generated schema models OpenAPI binary data as string; openapi-fetch
			// passes FormData through at runtime and lets it set the multipart boundary.
			body: upload as unknown as { avatar: string }
		});
		if (error || !data) {
			return fail(error?.status ?? 500, {
				avatarError: error?.detail ?? 'Failed to update profile photo'
			});
		}
		return { avatarSuccess: 'Profile photo updated', user: data };
	},
	removeAvatar: async ({ cookies }) => {
		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { avatarError: 'Not authenticated' });
		}
		const { data, error } = await api.DELETE('/api/me/avatar', {
			headers: { Cookie: `session=${sessionToken}` }
		});
		if (error || !data) {
			return fail(error?.status ?? 500, {
				avatarError: error?.detail ?? 'Failed to remove profile photo'
			});
		}
		return { avatarSuccess: 'Profile photo removed', user: data };
	},
	update: async ({ request, cookies }) => {
		const formData = await request.formData();
		const displayName = (formData.get('display_name') as string)?.trim();
		const username = (formData.get('username') as string)?.trim();
		const sportsProfile = sportsProfileFromFormData(formData);
		const bio = String(formData.get('bio') ?? '').trim();
		const profileLinks = profileLinksFromFormData(formData);
		const profileForm = {
			bio,
			profileLinksForm: profileLinksToForm(profileLinks)
		};

		if (!displayName) {
			return fail(400, { error: 'Display name is required', profileForm });
		}

		if (username === '') {
			return fail(400, { error: 'Username cannot be empty', profileForm });
		}

		if ([...bio].length > 500) {
			return fail(422, { error: 'Bio must be 500 characters or fewer', profileForm });
		}

		const sessionToken = cookies.get('session');
		if (!sessionToken) {
			return fail(401, { error: 'Not authenticated', profileForm });
		}

		const { data, error } = await api.PATCH('/api/me/', {
			headers: { Cookie: `session=${sessionToken}` },
			body: {
				display_name: displayName,
				username: username || undefined,
				sports_profile: sportsProfile,
				bio,
				profile_links: profileLinks
			}
		});

		if (error) {
			return fail(error.status ?? 500, {
				error: error.detail ?? 'Failed to update profile',
				profileForm
			});
		}

		if (!data) {
			return fail(500, { error: 'Failed to update profile', profileForm });
		}

		return {
			success: true,
			user: data,
			sportsProfileForm: sportsProfileToForm(data.sports_profile),
			profileForm: {
				bio: data.bio ?? '',
				profileLinksForm: profileLinksToForm(data.profile_links)
			}
		};
	}
};
