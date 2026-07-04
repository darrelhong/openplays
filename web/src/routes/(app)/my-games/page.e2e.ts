import { expect, test, type BrowserContext } from '@playwright/test';
import { makePlay, SEED_USERS, startMockApi, type MockApi } from '$lib/testing/mock-api';
import { dismissPushPrompt } from '$lib/testing/e2e';

/**
 * My Games tabs: Upcoming (default) vs Past. Past includes ended games and
 * cancelled games (with their badge), served by `/api/me/plays/past`.
 */
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

async function signIn(context: BrowserContext, userId: keyof typeof SEED_USERS) {
	await context.addCookies([{ name: 'session', value: userId, domain: 'localhost', path: '/' }]);
	mock.setUser(SEED_USERS[userId]!);
}

test('my games defaults to upcoming and switches to past', async ({ page, context }) => {
	await signIn(context, 'seed-host');

	const endedAt = new Date(Date.now() - 3 * 24 * 60 * 60 * 1000);
	mock.on('GET', '/api/me/plays', {
		json: {
			items: [makePlay({ id: 'up-1', name: 'Tuesday Doubles', viewer_state: 'creator' })],
			total: 1,
			has_more: false,
			next_cursor: null
		}
	});
	mock.on('GET', '/api/me/plays/past', {
		json: {
			items: [
				makePlay({
					id: 'past-1',
					name: 'Last Week Session',
					viewer_state: 'creator',
					starts_at: new Date(endedAt.getTime() - 2 * 60 * 60 * 1000).toISOString(),
					ends_at: endedAt.toISOString()
				}),
				makePlay({
					id: 'past-2',
					name: 'Rained Out Game',
					viewer_state: 'creator',
					cancelled_at: endedAt.toISOString()
				})
			],
			total: 2,
			has_more: false,
			next_cursor: null
		}
	});

	await page.goto('/my-games');
	await dismissPushPrompt(page);

	// Names render in both the mobile grid and the desktop table; assert
	// within the table, the visible variant at the test viewport
	const table = page.locator('table');
	await expect(page.getByText('Showing 1 upcoming game')).toBeVisible();
	await expect(table.getByText('Tuesday Doubles')).toBeVisible();

	await page.getByRole('link', { name: 'Past' }).click();
	await expect(page).toHaveURL(/\/my-games\/past$/);
	await expect(page.getByText('Showing 2 past games')).toBeVisible();
	await expect(table.getByText('Last Week Session')).toBeVisible();
	await expect(table.getByText('Rained Out Game')).toBeVisible();
});
