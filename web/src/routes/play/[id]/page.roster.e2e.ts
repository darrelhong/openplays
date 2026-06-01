import { expect, test, type BrowserContext, type Page } from '@playwright/test';
import { makePlay, participantFor, SEED_USERS, startMockApi, type MockApi } from '$lib/testing/mock-api';

/**
 * End-to-end coverage of the play-detail roster flows in
 * `play-details-content.svelte`: join (direct + waitlist), confirm an added
 * spot, leave, and host-side add/remove from the waitlist.
 *
 * Each test seeds a play in the in-process mock backend, plants a `session`
 * cookie (value = seed user id), drives the real UI, and asserts the state the
 * server action produced on reload. See `$lib/testing/mock-api`.
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
});

async function loginAs(context: BrowserContext, userId: keyof typeof SEED_USERS) {
	await context.addCookies([{ name: 'session', value: userId, domain: 'localhost', path: '/' }]);
}

/** Open the ActionConfirmDialog via its trigger, then click its confirm button. */
async function confirmAction(page: Page, triggerName: string | RegExp, confirmName: string) {
	await page.getByRole('button', { name: triggerName }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog).toBeVisible();
	await dialog.getByRole('button', { name: confirmName }).click();
}

test('eligible user joins a game directly', async ({ page, context }) => {
	mock.plays.set('play-1', makePlay());
	await loginAs(context, 'seed-advanced');

	await page.goto('/play/play-1');

	// Seed Advanced (A) is within LB–A, so the affordance is direct join.
	await expect(page.getByRole('button', { name: 'Join game' })).toBeVisible();
	await confirmAction(page, 'Join game', 'Join game');

	await expect(page.getByRole('button', { name: 'Leave game' })).toBeVisible();
	await expect(page.getByText('Confirmed').first()).toBeVisible();
});

test('ineligible user joins the waitlist', async ({ page, context }) => {
	// Seed Low Intermediate (LI) is below the play's minimum (HI) → waitlist only.
	mock.plays.set('play-1', makePlay({ level_min: 'HI', level_max: 'A' }));
	await loginAs(context, 'seed-li');

	await page.goto('/play/play-1');

	await expect(page.getByRole('button', { name: 'Join waitlist' })).toBeVisible();
	await confirmAction(page, 'Join waitlist', 'Join waitlist');

	await expect(page.getByRole('button', { name: 'Leave waitlist' })).toBeVisible();
	await expect(page.getByText('On waitlist').first()).toBeVisible();
});

test('added player confirms their spot', async ({ page, context }) => {
	const advanced = SEED_USERS['seed-advanced']!;
	mock.plays.set(
		'play-1',
		makePlay({ added_participants: [participantFor(advanced)], viewer_state: 'added' })
	);
	await loginAs(context, 'seed-advanced');

	await page.goto('/play/play-1');

	await confirmAction(page, 'Confirm spot', 'Confirm spot');

	await expect(page.getByRole('button', { name: 'Leave game' })).toBeVisible();
});

test('added player declines their spot', async ({ page, context }) => {
	const advanced = SEED_USERS['seed-advanced']!;
	mock.plays.set(
		'play-1',
		makePlay({ added_participants: [participantFor(advanced)], viewer_state: 'added' })
	);
	await loginAs(context, 'seed-advanced');

	await page.goto('/play/play-1');

	// "Decline" posts to ?/leave, removing the viewer from the roster.
	await confirmAction(page, 'Decline', 'Decline');

	// Back to not-joined, and (still eligible) can join directly again.
	await expect(page.getByRole('button', { name: 'Join game' })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Confirm spot' })).toHaveCount(0);
});

test('confirmed user leaves the game', async ({ page, context }) => {
	const advanced = SEED_USERS['seed-advanced']!;
	mock.plays.set(
		'play-1',
		makePlay({
			slots_left: 4,
			confirmed_participants: [
				{ id: 39, display_name: 'Seed Host', rating_code: 'HB', is_guest: false, is_host: true },
				participantFor(advanced)
			],
			viewer_state: 'confirmed'
		})
	);
	await loginAs(context, 'seed-advanced');

	await page.goto('/play/play-1');

	await confirmAction(page, 'Leave game', 'Leave game');

	// Back to not-joined, and (still eligible) can join directly again.
	await expect(page.getByRole('button', { name: 'Join game' })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Leave game' })).toHaveCount(0);
});

test('host adds a waitlisted player into the game', async ({ page, context }) => {
	const li = SEED_USERS['seed-li']!;
	mock.plays.set('play-1', makePlay({ waitlist: [participantFor(li)] }));
	await loginAs(context, 'seed-host');

	await page.goto('/play/play-1');

	await confirmAction(page, 'Add Seed Low Intermediate', 'Add player');

	// Left the waitlist (Add only renders on waitlist rows) but is still on the roster.
	await expect(page.getByRole('button', { name: 'Add Seed Low Intermediate' })).toHaveCount(0);
	await expect(page.getByText('Seed Low Intermediate')).toBeVisible();
});

test('host removes a waitlisted player', async ({ page, context }) => {
	const li = SEED_USERS['seed-li']!;
	mock.plays.set('play-1', makePlay({ waitlist: [participantFor(li)] }));
	await loginAs(context, 'seed-host');

	await page.goto('/play/play-1');

	await confirmAction(page, 'Remove Seed Low Intermediate', 'Remove player');

	await expect(page.getByText('Seed Low Intermediate')).toHaveCount(0);
});

test('host cancels the game from the edit page', async ({ page, context }) => {
	mock.plays.set('play-1', makePlay());
	await loginAs(context, 'seed-host');

	// Cancel lives on the edit route (?/cancelPlay → DELETE the play).
	await page.goto('/play/play-1/edit');
	await confirmAction(page, 'Cancel game', 'Cancel game');

	// Redirects back to the play, now cancelled and no longer joinable.
	await expect(page).toHaveURL(/\/play\/play-1$/);
	await expect(page.getByText('Cancelled').first()).toBeVisible();
	await expect(page.getByRole('button', { name: 'Join game' })).toHaveCount(0);
});
