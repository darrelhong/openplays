import { expect, test } from '@playwright/test';
import { mockApiPort } from '$lib/testing/mock-api';
import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';

const ianaTimezonePattern = /^(UTC|[A-Za-z_]+(?:\/[A-Za-z0-9_+-]+)+)$/;
let mockApi: ReturnType<typeof createServer>;
let playsRequestTimezones: Array<string | null> = [];

function respondJSON(res: ServerResponse, payload: unknown) {
	res.statusCode = 200;
	res.setHeader('content-type', 'application/json');
	res.end(JSON.stringify(payload));
}

function handleMockApi(req: IncomingMessage, res: ServerResponse) {
	const port = mockApiPort();
	const path = req.url ? new URL(req.url, `http://localhost:${port}`).pathname : '';
	if (req.method === 'GET' && path === '/api/plays/') {
		playsRequestTimezones.push(
			new URL(req.url ?? '', `http://localhost:${port}`).searchParams.get('timezone')
		);
		respondJSON(res, {
			items: [],
			total: 0,
			next_cursor: null
		});
		return;
	}

	if (req.method === 'GET' && path === '/api/venues/') {
		respondJSON(res, {
			// Indices 1 and 12 used to collide: 1 + '23' === 12 + '3'.
			items: Array.from({ length: 13 }, (_, index) => ({
				id: index === 1 ? 23 : index === 12 ? 3 : 100 + index,
				name: `Mock Venue ${index}`,
				latitude: 1.3 + index / 1000,
				longitude: 103.8
			}))
		});
		return;
	}

	res.statusCode = 404;
	res.end('not found');
}

test.beforeAll(async () => {
	mockApi = createServer(handleMockApi);
	await new Promise<void>((resolve, reject) => {
		mockApi.once('error', reject);
		mockApi.listen(mockApiPort(), resolve);
	});
});

test.afterAll(async () => {
	if (!mockApi) return;
	await new Promise<void>((resolve, reject) => {
		mockApi.close((err) => {
			if (err) reject(err);
			else resolve();
		});
	});
});

test('venue dropdown renders distinct IDs without key collisions and leaves filters working', async ({
	page
}) => {
	const errors: string[] = [];
	page.on('pageerror', (error) => errors.push(error.message));
	await page.goto('/');
	await page.getByRole('combobox', { name: 'Sort near venue' }).click();
	await expect(page.getByRole('option')).toHaveCount(13);
	await page.getByRole('option', { name: 'Mock Venue 1', exact: true }).click();
	await expect(page).toHaveURL(/lat=1\.301/);
	await page.getByRole('button', { name: 'Sport', exact: true }).click();
	await page.getByRole('option', { name: 'Tennis', exact: true }).click();
	await expect(page).toHaveURL(/sport=tennis/);
	expect(errors).toEqual([]);
});

test('clear filters preserves timezone and removes date bounds', async ({ page }) => {
	playsRequestTimezones = [];
	await page.goto('/?starts_after=2026-04-10&starts_before=2026-04-11');
	await expect(page.getByRole('button', { name: 'Clear filters' })).toBeVisible();

	const browserTimezone = await page.evaluate(
		() => Intl.DateTimeFormat().resolvedOptions().timeZone
	);

	await page.getByRole('button', { name: 'Clear filters' }).click();
	await expect(page).toHaveURL(/timezone=/);
	await expect.poll(() => playsRequestTimezones.at(-1) ?? null).toBe(browserTimezone);

	const timezone = playsRequestTimezones.at(-1);
	if (!timezone) {
		throw new Error('expected plays request timezone to be captured');
	}

	const nextURL = new URL(page.url());
	expect(nextURL.searchParams.get('starts_after')).toBeNull();
	expect(nextURL.searchParams.get('starts_before')).toBeNull();
	expect(timezone).toBe(browserTimezone);
	expect(timezone).toMatch(ianaTimezonePattern);
	expect(nextURL.searchParams.get('timezone')).toBe(timezone);
});
