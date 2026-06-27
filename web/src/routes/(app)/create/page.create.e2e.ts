import { expect, test, type BrowserContext, type Page } from '@playwright/test';
import { HOST, makePlay, SEED_USERS, startMockApi, type MockApi } from '$lib/testing/mock-api';
import { dismissPushPrompt } from '$lib/testing/e2e';
import type { IncomingMessage } from 'node:http';

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

async function signIn(context: BrowserContext) {
	await context.addCookies([
		{ name: 'session', value: 'seed-host', domain: 'localhost', path: '/' }
	]);
	mock.setUser(SEED_USERS['seed-host']!);
}

async function setFormValue(page: Page, selector: string, value: string) {
	await page.locator(selector).evaluate((element, nextValue) => {
		const input = element as HTMLInputElement;
		input.value = nextValue;
		input.dispatchEvent(new Event('input', { bubbles: true }));
		input.dispatchEvent(new Event('change', { bubbles: true }));
	}, value);
}

async function selectOption(page: Page, label: string, option: string) {
	await page.getByLabel(label).click();
	await page.getByRole('option', { name: option, exact: true }).click();
}

async function readJSON(req: IncomingMessage) {
	const chunks: Buffer[] = [];
	for await (const chunk of req) {
		chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
	}
	return JSON.parse(Buffer.concat(chunks).toString('utf8') || '{}') as Record<string, unknown>;
}

test('creates a game and shows submitted fields on the detail page', async ({ page, context }) => {
	await signIn(context);
	mock.on('GET', '/api/venues/', { json: { items: [] } });
	mock.on('GET', '/api/venues/search', ({ url }) => {
		expect(url.searchParams.get('q')).toBe('Test Sports Hall');
		return {
			json: {
				items: [
					{
						google_place_id: 'ChIJ77',
						name: 'Test Sports Hall',
						address: '123 Test Road, Singapore'
					}
				]
			}
		};
	});
	mock.on('POST', '/api/venues/resolve-google', async ({ req }) => {
		const body = await readJSON(req);
		expect(body).toMatchObject({
			google_place_id: 'ChIJ77',
			query: 'Test Sports Hall'
		});
		return {
			json: {
				id: 77,
				name: 'Test Sports Hall',
				address: '123 Test Road, Singapore',
				postal_code: '123456',
				latitude: 1.3,
				longitude: 103.8,
				google_place_id: 'ChIJ77'
			}
		};
	});

	let createdBody: Record<string, unknown> | null = null;
	mock.on('POST', '/api/plays/', async ({ req }) => {
		createdBody = await readJSON(req);
		const startsAt = new Date(String(createdBody.starts_at));
		const durationMinutes = Number(createdBody.duration_minutes ?? 120);
		const endsAt = new Date(startsAt.getTime() + durationMinutes * 60_000);
		const maxPlayers = Number(createdBody.max_players);
		const slotsLeft = Number(createdBody.slots_left);
		const courts = Number(createdBody.courts);
		const play = makePlay({
			id: 'created-play',
			name: createdBody.name,
			description: createdBody.description,
			visibility: createdBody.visibility,
			venue: createdBody.venue,
			venue_name: createdBody.venue,
			venue_id: createdBody.venue_id,
			venue_google_place_id: 'ChIJ77',
			sport: createdBody.sport,
			game_type: createdBody.game_type,
			starts_at: startsAt.toISOString(),
			ends_at: endsAt.toISOString(),
			fee: createdBody.fee,
			max_players: maxPlayers,
			slots_left: slotsLeft,
			courts,
			can_manage: true,
			viewer_state: 'creator',
			confirmed_participants: [HOST]
		});
		mock.setPlay(play);
		return { json: play };
	});

	await page.goto('/create');
	await dismissPushPrompt(page);

	await selectOption(page, 'Sport', 'Badminton');
	await page.getByLabel('Name').fill('Friday Friendly');
	await page.getByLabel('Description').fill('Bring water and shuttles.');
	await page.getByRole('checkbox', { name: /Set visibility as unlisted/ }).click();
	await page.getByLabel('Venue').fill('Test Sports Hall');
	const venueOption = page.getByRole('option', { name: /Test Sports Hall/ });
	await expect(venueOption).toBeVisible();
	await venueOption.click();
	await expect(page.locator('input[name="venue_id"]')).toHaveValue('77');
	await page.getByLabel('Start Time').fill('19:30');
	await selectOption(page, 'Game Type', 'Doubles');
	await page.getByLabel('Fee ($)').fill('12.50');
	await page.getByLabel('Max Players').fill('4');
	await page.getByLabel('Courts').fill('2');

	await setFormValue(page, 'input[name="date"]', '2026-07-10');
	await setFormValue(page, 'input[name="starts_at"]', '2026-07-10T19:30:00+08:00');

	const createForm = page.locator('form').filter({
		has: page.getByRole('button', { name: 'Create Game' })
	});
	const submittedValues = await createForm.evaluate((form) =>
		Object.fromEntries(new FormData(form as HTMLFormElement).entries())
	);
	expect(submittedValues).toMatchObject({
		sport: 'badminton',
		game_type: 'doubles',
		name: 'Friday Friendly',
		description: 'Bring water and shuttles.',
		visibility: 'unlisted',
		venue: 'Test Sports Hall',
		venue_id: '77',
		date: '2026-07-10',
		start_time: '19:30',
		starts_at: '2026-07-10T19:30:00+08:00',
		fee: '12.50',
		max_players: '4',
		courts: '2'
	});
	const invalidFields = await createForm.evaluate((form) =>
		Array.from(form.querySelectorAll(':invalid')).map((element) => ({
			id: element.id,
			name: element.getAttribute('name'),
			message: (element as HTMLInputElement).validationMessage
		}))
	);
	expect(invalidFields).toEqual([]);

	const actionRequest = page.waitForRequest(
		(request) => request.method() === 'POST' && new URL(request.url()).pathname === '/create'
	);
	await Promise.all([actionRequest, page.getByRole('button', { name: 'Create Game' }).click()]);

	await expect.poll(() => createdBody).not.toBeNull();
	await expect(page).toHaveURL(/\/play\/created-play$/);
	expect(createdBody).toMatchObject({
		sport: 'badminton',
		game_type: 'doubles',
		name: 'Friday Friendly',
		description: 'Bring water and shuttles.',
		visibility: 'unlisted',
		venue: 'Test Sports Hall',
		venue_id: 77,
		fee: 1250,
		max_players: 4,
		slots_left: 3,
		courts: 2
	});
	await expect(page.getByRole('heading', { name: 'Friday Friendly' })).toBeVisible();
	await expect(page.getByText('Bring water and shuttles.')).toBeVisible();
	await expect(page.getByText('Unlisted')).toBeVisible();
	await expect(page.getByText('Test Sports Hall')).toBeVisible();
	await expect(page.getByText('Doubles')).toBeVisible();
	await expect(page.getByText('$12.50')).toBeVisible();
});
