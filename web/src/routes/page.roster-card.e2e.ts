import { expect, test } from '@playwright/test';
import { HOST, makePlay, startMockApi, type MockApi } from '$lib/testing/mock-api';

/**
 * Home-page card occupancy. `slots_left` is the server's slot accounting
 * (confirmed and added players both reserve slots), so a game filled by
 * 'added' players — the default direct-join outcome — must render as full
 * even when the participant preview omits reserved players.
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

test('play card shows a full roster when reserved players hold the slots', async ({ page }) => {
	// Full game: only the confirmed host is in the preview, the other three
	// slots are reserved by 'added' players.
	const play = makePlay({
		slots_left: 0,
		max_players: 4,
		participant_preview: [HOST]
	});
	mock.on('GET', '/api/plays/', {
		json: { items: [play], total: 1, next_cursor: null }
	});
	mock.on('GET', '/api/venues/', { json: { items: [] } });

	await page.goto('/');

	// The desktop table and mobile grid both render the card; assert on the
	// desktop variant (default Playwright viewport)
	const table = page.locator('table');
	await expect(table.getByLabel('4/4 joined')).toBeVisible();
	await expect(table.getByText('Reserved spot 2')).toBeAttached();
	await expect(table.getByText('Open slot 1')).toHaveCount(0);
});

test('play card shows open slots when the game has room', async ({ page }) => {
	const play = makePlay({
		slots_left: 3,
		max_players: 4,
		participant_preview: [HOST]
	});
	mock.on('GET', '/api/plays/', {
		json: { items: [play], total: 1, next_cursor: null }
	});
	mock.on('GET', '/api/venues/', { json: { items: [] } });

	await page.goto('/');

	const table = page.locator('table');
	await expect(table.getByLabel('1/4 joined')).toBeVisible();
	await expect(table.getByText('Open slot 3')).toBeAttached();
	await expect(table.getByText('Reserved spot 2')).toHaveCount(0);
});
