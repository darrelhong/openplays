import { expect, test, type BrowserContext } from '@playwright/test';
import { SEED_USERS, startMockApi, type MockApi, type SeedUser } from '$lib/testing/mock-api';

let mock: MockApi;

test.beforeAll(async () => {
	mock = await startMockApi();
});

test.afterAll(async () => {
	await mock?.close();
});

test.beforeEach(() => {
	mock.reset();
});

async function signIn(context: BrowserContext, user: SeedUser) {
	await context.addCookies([{ name: 'session', value: user.id, domain: 'localhost', path: '/' }]);
	mock.setUser(user);
}

test('uploads and removes a custom profile photo', async ({ page, context }) => {
	const oauthUser: SeedUser = {
		...SEED_USERS['seed-host']!,
		photo_url: 'https://provider.example/avatar.jpg',
		has_custom_avatar: false
	};
	const customUser: SeedUser = {
		...oauthUser,
		photo_url: 'https://images.example/avatars/seed-host/custom.jpg',
		has_custom_avatar: true
	};
	await signIn(context, oauthUser);
	mock.on('PUT', '/api/me/avatar', () => {
		mock.setUser(customUser);
		return { json: customUser };
	});
	mock.on('DELETE', '/api/me/avatar', () => {
		mock.setUser(oauthUser);
		return { json: oauthUser };
	});

	await page.goto('/profile');
	await expect(page.getByRole('button', { name: 'Remove', exact: true })).toHaveCount(0);
	await page.locator('input[name="avatar"]').setInputFiles({
		name: 'avatar.png',
		mimeType: 'image/png',
		buffer: Buffer.from('mock image')
	});
	await expect(page.getByText('Profile photo updated')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Remove', exact: true })).toBeVisible();

	await page.getByRole('button', { name: 'Remove', exact: true }).click();
	await expect(page.getByText('Profile photo removed')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Remove', exact: true })).toHaveCount(0);
});

test('hides a stale upload error while retrying', async ({ page, context }) => {
	const user = SEED_USERS['seed-host']!;
	await signIn(context, user);

	let attempts = 0;
	let finishRetry!: () => void;
	const retryCanFinish = new Promise<void>((resolve) => {
		finishRetry = resolve;
	});
	let markRetryStarted!: () => void;
	const retryStarted = new Promise<void>((resolve) => {
		markRetryStarted = resolve;
	});
	mock.on('PUT', '/api/me/avatar', async () => {
		attempts += 1;
		if (attempts === 1) {
			return { status: 422, json: { status: 422, detail: 'The first upload failed' } };
		}
		markRetryStarted();
		await retryCanFinish;
		return { json: { ...user, has_custom_avatar: true } };
	});

	await page.goto('/profile');
	const input = page.locator('input[name="avatar"]');
	await input.setInputFiles({
		name: 'first.png',
		mimeType: 'image/png',
		buffer: Buffer.from('first mock image')
	});
	await expect(page.getByText('The first upload failed')).toBeVisible();

	await input.setInputFiles({
		name: 'retry.png',
		mimeType: 'image/png',
		buffer: Buffer.from('retry mock image')
	});
	await retryStarted;
	await expect(page.getByText('The first upload failed')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Uploading…' })).toBeVisible();

	finishRetry();
	await expect(page.getByText('Profile photo updated')).toBeVisible();
});
