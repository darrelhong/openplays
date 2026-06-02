import { expect, test, type BrowserContext, type Page } from '@playwright/test';
import {
	HOST,
	makePlay,
	participantFor,
	SEED_USERS,
	startMockApi,
	type MockApi
} from '$lib/testing/mock-api';

/**
 * End-to-end coverage of the play-detail roster flows in
 * `play-details-content.svelte`: join (direct + waitlist), confirm/decline an
 * added spot, leave, host add/remove from the waitlist, and cancel.
 *
 * Each test declares what the backend returns: the play served to `load`
 * (`setPlay`), and what each action responds plus the play it transitions to
 * for the reload (`action({ then })`). The mock holds no logic — see
 * `$lib/testing/mock-api`.
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

/** Plant the session cookie (so hooks fires) and set who `/api/me/` returns. */
async function signIn(context: BrowserContext, userId: keyof typeof SEED_USERS) {
	await context.addCookies([{ name: 'session', value: userId, domain: 'localhost', path: '/' }]);
	mock.setUser(SEED_USERS[userId]!);
}

/** Open an ActionConfirmDialog via its trigger, then click its confirm button. */
async function confirmAction(page: Page, triggerName: string | RegExp, confirmName: string) {
	await page.getByRole('button', { name: triggerName }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog).toBeVisible();
	await dialog.getByRole('button', { name: confirmName }).click();
}

const advanced = SEED_USERS['seed-advanced']!;
const li = SEED_USERS['seed-li']!;

test('eligible user joins a game directly', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(makePlay({ viewer_state: 'not_joined' }));
	mock.action('POST', '/api/plays/play-1/join', {
		then: makePlay({
			viewer_state: 'confirmed',
			slots_left: 4,
			confirmed_participants: [HOST, participantFor(advanced)]
		})
	});

	await page.goto('/play/play-1');

	await expect(page.getByRole('button', { name: 'Join game' })).toBeVisible();
	await confirmAction(page, 'Join game', 'Join game');

	await expect(page.getByRole('button', { name: 'Leave game' })).toBeVisible();
	await expect(page.getByText('Confirmed').first()).toBeVisible();
});

test('ineligible user joins the waitlist', async ({ page, context }) => {
	await signIn(context, 'seed-li');
	// Seed Low Intermediate is below the play's minimum, so the UI offers waitlist.
	mock.setPlay(makePlay({ level_min: 'HI', level_max: 'A', viewer_state: 'not_joined' }));
	mock.action('POST', '/api/plays/play-1/join', {
		then: makePlay({
			level_min: 'HI',
			level_max: 'A',
			viewer_state: 'waitlisted',
			waitlist: [participantFor(li)]
		})
	});

	await page.goto('/play/play-1');

	await expect(page.getByRole('button', { name: 'Join waitlist' })).toBeVisible();
	await confirmAction(page, 'Join waitlist', 'Join waitlist');

	await expect(page.getByRole('button', { name: 'Leave waitlist' })).toBeVisible();
	await expect(page.getByText('On waitlist').first()).toBeVisible();
});

test('added player confirms their spot', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(makePlay({ viewer_state: 'added', added_participants: [participantFor(advanced)] }));
	mock.action('POST', '/api/plays/play-1/participants/me/confirm', {
		then: makePlay({
			viewer_state: 'confirmed',
			slots_left: 4,
			confirmed_participants: [HOST, participantFor(advanced)]
		})
	});

	await page.goto('/play/play-1');

	await confirmAction(page, 'Confirm spot', 'Confirm spot');

	await expect(page.getByRole('button', { name: 'Leave game' })).toBeVisible();
});

test('added player declines their spot', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(makePlay({ viewer_state: 'added', added_participants: [participantFor(advanced)] }));
	// "Decline" posts to ?/leave, dropping the viewer back to not-joined.
	mock.action('DELETE', '/api/plays/play-1/participants/me', {
		then: makePlay({ viewer_state: 'not_joined' })
	});

	await page.goto('/play/play-1');

	await confirmAction(page, 'Decline', 'Decline');

	await expect(page.getByRole('button', { name: 'Join game' })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Confirm spot' })).toHaveCount(0);
});

test('confirmed user leaves the game', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(
		makePlay({
			viewer_state: 'confirmed',
			slots_left: 4,
			confirmed_participants: [HOST, participantFor(advanced)]
		})
	);
	mock.action('DELETE', '/api/plays/play-1/participants/me', {
		then: makePlay({ viewer_state: 'not_joined' })
	});

	await page.goto('/play/play-1');

	await confirmAction(page, 'Leave game', 'Leave game');

	await expect(page.getByRole('button', { name: 'Join game' })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Leave game' })).toHaveCount(0);
});

test('host adds a waitlisted player into the game', async ({ page, context }) => {
	await signIn(context, 'seed-host');
	const waitlisted = participantFor(li);
	mock.setPlay(makePlay({ can_manage: true, viewer_state: 'creator', waitlist: [waitlisted] }));
	mock.action('POST', '/api/plays/play-1/participants/:pid/accept', {
		then: makePlay({ can_manage: true, viewer_state: 'creator', added_participants: [waitlisted] })
	});

	await page.goto('/play/play-1');

	await confirmAction(page, 'Add Seed Low Intermediate', 'Add player');

	// Left the waitlist (Add only renders on waitlist rows) but is still on the roster.
	await expect(page.getByRole('button', { name: 'Add Seed Low Intermediate' })).toHaveCount(0);
	await expect(page.getByText('Seed Low Intermediate')).toBeVisible();
});

test('host removes a waitlisted player', async ({ page, context }) => {
	await signIn(context, 'seed-host');
	mock.setPlay(
		makePlay({ can_manage: true, viewer_state: 'creator', waitlist: [participantFor(li)] })
	);
	mock.action('DELETE', '/api/plays/play-1/participants/:pid', {
		then: makePlay({ can_manage: true, viewer_state: 'creator' })
	});

	await page.goto('/play/play-1');

	await confirmAction(page, 'Remove Seed Low Intermediate', 'Remove player');

	await expect(page.getByText('Seed Low Intermediate')).toHaveCount(0);
});

test('host cancels the game from the edit page', async ({ page, context }) => {
	await signIn(context, 'seed-host');
	// The edit route's load requires a manageable, non-cancelled play.
	mock.setPlay(makePlay({ can_manage: true, viewer_state: 'creator' }));
	mock.action('DELETE', '/api/plays/play-1', {
		then: makePlay({
			can_manage: true,
			viewer_state: 'creator',
			cancelled_at: '2026-06-01T00:00:00Z'
		})
	});

	await page.goto('/play/play-1/edit');
	await confirmAction(page, 'Cancel game', 'Cancel game');

	await expect(page).toHaveURL(/\/play\/play-1$/);
	await expect(page.getByText('Cancelled').first()).toBeVisible();
	await expect(page.getByRole('button', { name: 'Join game' })).toHaveCount(0);
});
