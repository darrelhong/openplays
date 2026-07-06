import { expect, test, type BrowserContext } from '@playwright/test';
import type { IncomingMessage } from 'node:http';
import {
	HOST,
	makePlay,
	participantFor,
	SEED_USERS,
	startMockApi,
	type MockApi
} from '$lib/testing/mock-api';
import { dismissPushPrompt } from '$lib/testing/e2e';

/**
 * The per-player review page (/play/[id]/review/[username]): anonymous star
 * rating, prop chips (max 2, host props only when the reviewee hosted), and a
 * shoutout. Entry point: "Give props" buttons on an ended play's roster.
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

async function readJsonBody(req: IncomingMessage): Promise<Record<string, unknown>> {
	const chunks: Buffer[] = [];
	for await (const chunk of req) {
		chunks.push(chunk as Buffer);
	}
	return JSON.parse(Buffer.concat(chunks).toString());
}

function endedPlay(overrides: Parameters<typeof makePlay>[0] = {}) {
	const endsAt = new Date(Date.now() - 2 * 60 * 60 * 1000);
	return makePlay({
		starts_at: new Date(endsAt.getTime() - 2 * 60 * 60 * 1000).toISOString(),
		ends_at: endsAt.toISOString(),
		...overrides
	});
}

const openWindow = {
	state: 'open',
	closes_at: new Date(Date.now() + 13 * 24 * 60 * 60 * 1000).toISOString()
};

const PEER_PROPS = ['great_sport', 'chill_vibes', 'humble', 'punctual', 'powerful_smash'];
const HOST_PROPS = ['well_organized', 'quick_replies', 'clear_communication'];

const REVIEWEES = [
	{ user_id: 'seed-host', display_name: 'Seed Host', username: 'seedhost', is_host: true },
	{
		user_id: 'seed-li',
		display_name: 'Seed Low Intermediate',
		username: 'seedli',
		is_host: false
	}
];

function serveReviewSheet(reviewees: unknown[] = REVIEWEES, window: unknown = openWindow) {
	mock.on('GET', '/api/plays/play-1/reviews', {
		json: { window, peer_props: PEER_PROPS, host_props: HOST_PROPS, reviewees }
	});
}

test('reviewer fills stars, props, and a shoutout for the host', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(endedPlay());
	serveReviewSheet();

	let putBody: Record<string, unknown> | null = null;
	let putPath = '';
	mock.on('PUT', '/api/plays/play-1/reviews/:userId', async ({ req, params }) => {
		putBody = await readJsonBody(req);
		putPath = params.userId!;
		return { json: putBody };
	});

	await page.goto('/play/play-1/review/seedhost');
	await dismissPushPrompt(page);

	await expect(
		page.getByRole('heading', { name: 'How was your game with Seed Host?' })
	).toBeVisible();

	// Props reveal after rating; the host gets the hosting prop group
	await page.getByRole('radio', { name: '4 stars' }).check({ force: true });
	await expect(page.getByText('For hosting')).toBeVisible();
	await page.getByRole('checkbox', { name: 'Well organized' }).check({ force: true });
	await page.getByRole('checkbox', { name: 'Great sport' }).check({ force: true });
	// The two-prop cap disables the rest
	await expect(page.getByRole('checkbox', { name: 'Chill vibes' })).toBeDisabled();

	await page.getByPlaceholder('Give Seed Host a public shoutout').fill('Ran a great session');
	await page.getByRole('button', { name: 'Submit' }).click();

	// Saving sends the reviewer back to the game
	await expect(page).toHaveURL(/\/play\/play-1$/);
	expect(putPath).toBe('seed-host');
	expect(putBody).toEqual({
		rating: 4,
		props: ['great_sport', 'well_organized'],
		shoutout: 'Ran a great session'
	});
});

test('non-host reviewee gets no hosting props', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(endedPlay());
	serveReviewSheet();

	await page.goto('/play/play-1/review/seedli');
	await dismissPushPrompt(page);

	await expect(
		page.getByRole('heading', { name: 'How was your game with Seed Low Intermediate?' })
	).toBeVisible();

	// Props stay hidden (space reserved) until a rating is picked
	await expect(page.getByText('Give props')).not.toBeVisible();
	await page.getByRole('radio', { name: '5 stars' }).check({ force: true });
	await expect(page.getByText('Give props')).toBeVisible();

	await expect(page.getByText('For hosting')).toHaveCount(0);
	await expect(page.getByRole('checkbox', { name: 'Well organized' })).toHaveCount(0);
});

test('a submitted review locks and is shown read-only', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(endedPlay());
	serveReviewSheet([
		{
			user_id: 'seed-li',
			display_name: 'Seed Low Intermediate',
			username: 'seedli',
			is_host: false,
			my_review: { rating: 5, props: ['chill_vibes'], shoutout: 'Great rallies' }
		}
	]);

	await page.goto('/play/play-1/review/seedli');
	await dismissPushPrompt(page);

	await expect(page.getByText('You rated 5/5')).toBeVisible();
	await expect(page.getByText('Chill vibes')).toBeVisible();
	await expect(page.getByText('“Great rallies”')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Submit' })).toHaveCount(0);
});

test('closed window shows the review read-only', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(endedPlay());
	serveReviewSheet(
		[
			{
				user_id: 'seed-li',
				display_name: 'Seed Low Intermediate',
				username: 'seedli',
				is_host: false,
				my_review: { rating: 3, props: [], shoutout: null }
			}
		],
		{ state: 'closed', closes_at: new Date(Date.now() - 1000).toISOString() }
	);

	await page.goto('/play/play-1/review/seedli');
	await dismissPushPrompt(page);

	await expect(page.getByText('The review window has closed.')).toBeVisible();
	await expect(page.getByText('You rated 3/5')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Submit' })).toHaveCount(0);
});

test('someone outside the game roster is not reviewable', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(endedPlay());
	serveReviewSheet();

	const response = await page.goto('/play/play-1/review/notinthisgame');
	expect(response?.status()).toBe(404);
});

test('ended play roster swaps confirmed badges for Give props buttons', async ({
	page,
	context
}) => {
	await signIn(context, 'seed-advanced');
	const advanced = SEED_USERS['seed-advanced']!;
	const li = SEED_USERS['seed-li']!;
	mock.setPlay(
		endedPlay({
			viewer_state: 'confirmed',
			confirmed_participants: [
				HOST,
				participantFor(advanced, { is_viewer: true }),
				participantFor(li)
			]
		})
	);
	// The viewer already reviewed seedli: their Give props button drops
	serveReviewSheet([
		REVIEWEES[0]!,
		{ ...REVIEWEES[1]!, my_review: { rating: 4, props: [], shoutout: null } }
	]);

	await page.goto('/play/play-1');
	await dismissPushPrompt(page);

	// Unreviewed co-players get a Give props link to their review page
	await expect(page.getByRole('link', { name: 'Give Seed Host props' })).toHaveAttribute(
		'href',
		'/play/play-1/review/seedhost'
	);
	// Already reviewed, yourself: no button; and no Confirmed badges remain
	await expect(page.getByRole('link', { name: /Give Seed Low Intermediate/ })).toHaveCount(0);
	await expect(page.getByRole('link', { name: /Give Seed Advanced/ })).toHaveCount(0);
	await expect(page.locator('table, ul').getByText('Confirmed')).toHaveCount(0);
});
