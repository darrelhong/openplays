import { expect, test } from '@playwright/test';
import { makePlay, startMockApi, type MockApi } from '$lib/testing/mock-api';

/**
 * Spike: a user eligible for the play's level joins a game directly (no waitlist).
 *
 * `seed-advanced` (badminton A) sits within the play's LB–A range, so
 * `canDirectJoin` is true and the button reads "Join game". Auth is planted as
 * a cookie whose value is the seed user id; the mock resolves `/api/me/` from it.
 */
let mock: MockApi;

test.beforeAll(async () => {
	mock = await startMockApi();
});

test.afterAll(async () => {
	await mock?.close();
});

test.beforeEach(() => {
	mock.plays.clear();
	mock.plays.set('play-1', makePlay());
});

test('eligible user joins a game directly', async ({ page, context }) => {
	await context.addCookies([
		{ name: 'session', value: 'seed-advanced', domain: 'localhost', path: '/' }
	]);

	await page.goto('/play/play-1');

	// Direct-join affordance (not the waitlist variant).
	const joinButton = page.getByRole('button', { name: 'Join game' });
	await expect(joinButton).toBeVisible();
	await joinButton.click();

	// Confirmation dialog from ActionConfirmDialog.
	const dialog = page.getByRole('dialog');
	await expect(dialog).toBeVisible();
	await dialog.getByRole('button', { name: 'Join game' }).click();

	// After the server action + reload, the viewer is confirmed and can leave.
	await expect(page.getByRole('button', { name: 'Leave game' })).toBeVisible();
	await expect(page.getByText('Confirmed').first()).toBeVisible();
});
