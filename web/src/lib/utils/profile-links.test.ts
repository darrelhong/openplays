import { describe, expect, it } from 'vitest';
import {
	PROFILE_LINK_PROVIDERS,
	profileLinksFromFormData,
	profileLinksToForm,
	profileLinkUrl
} from './profile-links';

describe('profile links', () => {
	it('keeps providers in the requested display order', () => {
		expect(PROFILE_LINK_PROVIDERS.map(({ key }) => key)).toEqual([
			'rovo',
			'reclub',
			'telegram',
			'instagram',
			'facebook',
			'x',
			'strava_athlete_id'
		]);
	});

	it('maps API values into complete form values', () => {
		expect(profileLinksToForm({ telegram: 'openplays_sg' })).toEqual({
			rovo: '',
			reclub: '',
			telegram: 'openplays_sg',
			instagram: '',
			facebook: '',
			x: '',
			strava_athlete_id: ''
		});
	});

	it('trims identifiers, removes @, and omits empty fields', () => {
		const formData = new FormData();
		formData.set('profile_link_rovo', '  @darrel  ');
		formData.set('profile_link_telegram', 'openplays_sg');
		formData.set('profile_link_instagram', '   ');

		expect(profileLinksFromFormData(formData)).toEqual({
			rovo: 'darrel',
			telegram: 'openplays_sg'
		});
	});

	it('builds provider-controlled URLs', () => {
		const reclub = PROFILE_LINK_PROVIDERS.find(({ key }) => key === 'reclub');
		const strava = PROFILE_LINK_PROVIDERS.find(({ key }) => key === 'strava_athlete_id');
		expect(reclub && profileLinkUrl(reclub, 'darrel')).toBe('https://reclub.co/players/@darrel');
		expect(strava && profileLinkUrl(strava, '123456')).toBe(
			'https://www.strava.com/athletes/123456'
		);
	});
});
