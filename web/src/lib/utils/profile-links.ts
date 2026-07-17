import type { components } from '$lib/api/types.gen';

export type ProfileLinks = components['schemas']['ProfileLinks'];
export type ProfileLinkKey = keyof ProfileLinks;
export type ProfileLinksFormValues = Record<ProfileLinkKey, string>;

type ProfileLinkProvider = {
	key: ProfileLinkKey;
	label: string;
	displayLabel: string;
	displayPrefix: '@' | '';
	placeholder: string;
	maxLength: number;
	pattern: string;
	inputMode?: 'numeric';
	buildUrl: (identifier: string) => string;
};

const profileUrl = (base: string, identifier: string) => `${base}${encodeURIComponent(identifier)}`;

export const PROFILE_LINK_PROVIDERS: ReadonlyArray<ProfileLinkProvider> = [
	{
		key: 'rovo',
		label: 'Rovo username',
		displayLabel: 'Rovo',
		displayPrefix: '@',
		placeholder: 'username',
		maxLength: 64,
		pattern: '[A-Za-z0-9._-]+',
		buildUrl: (identifier) => profileUrl('https://rovo.co/', identifier)
	},
	{
		key: 'reclub',
		label: 'Reclub username',
		displayLabel: 'Reclub',
		displayPrefix: '@',
		placeholder: 'username',
		maxLength: 64,
		pattern: '[A-Za-z0-9._-]+',
		buildUrl: (identifier) => profileUrl('https://reclub.co/players/@', identifier)
	},
	{
		key: 'telegram',
		label: 'Telegram',
		displayLabel: 'Telegram',
		displayPrefix: '@',
		placeholder: 'username',
		maxLength: 32,
		pattern: '[A-Za-z0-9_]+',
		buildUrl: (identifier) => profileUrl('https://t.me/', identifier)
	},
	{
		key: 'instagram',
		label: 'Instagram',
		displayLabel: 'Instagram',
		displayPrefix: '@',
		placeholder: 'username',
		maxLength: 30,
		pattern: '[A-Za-z0-9._]+',
		buildUrl: (identifier) => profileUrl('https://www.instagram.com/', identifier)
	},
	{
		key: 'facebook',
		label: 'Facebook',
		displayLabel: 'Facebook',
		displayPrefix: '@',
		placeholder: 'username',
		maxLength: 64,
		pattern: '[A-Za-z0-9._-]+',
		buildUrl: (identifier) => profileUrl('https://www.facebook.com/', identifier)
	},
	{
		key: 'x',
		label: 'X (Twitter)',
		displayLabel: 'X',
		displayPrefix: '@',
		placeholder: 'username',
		maxLength: 15,
		pattern: '[A-Za-z0-9_]+',
		buildUrl: (identifier) => profileUrl('https://x.com/', identifier)
	},
	{
		key: 'strava_athlete_id',
		label: 'Strava ID',
		displayLabel: 'Strava',
		displayPrefix: '',
		placeholder: 'athlete ID',
		maxLength: 20,
		pattern: '[0-9]+',
		inputMode: 'numeric',
		buildUrl: (identifier) => profileUrl('https://www.strava.com/athletes/', identifier)
	}
];

export function profileLinksToForm(links: ProfileLinks | null | undefined): ProfileLinksFormValues {
	return {
		rovo: links?.rovo ?? '',
		reclub: links?.reclub ?? '',
		telegram: links?.telegram ?? '',
		instagram: links?.instagram ?? '',
		facebook: links?.facebook ?? '',
		x: links?.x ?? '',
		strava_athlete_id: links?.strava_athlete_id ?? ''
	};
}

export function profileLinksFromFormData(formData: FormData): ProfileLinks {
	const links: ProfileLinks = {};
	for (const provider of PROFILE_LINK_PROVIDERS) {
		const identifier = stringFormValue(formData.get(`profile_link_${provider.key}`))
			.trim()
			.replace(/^@/, '');
		if (identifier) {
			links[provider.key] = identifier;
		}
	}
	return links;
}

export function profileLinkUrl(provider: ProfileLinkProvider, identifier: string): string {
	return provider.buildUrl(identifier);
}

function stringFormValue(value: FormDataEntryValue | null): string {
	return typeof value === 'string' ? value : '';
}
