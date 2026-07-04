import { expect, test, type BrowserContext, type Page } from '@playwright/test';
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
 * End-to-end coverage of the play-detail roster flows in
 * `play-details-content.svelte`.
 *
 * Classic plays: direct join (added → self-confirm), join requests when
 * full/level-mismatched, withdraw, confirm/decline an added spot, leave,
 * host add/remove from the request queue, roster visibility, and cancel.
 * Require-waitlist plays: request to join, host triage (add to game / add to
 * waitlist / remove), parked players leaving the waitlist.
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

test('eligible user joins directly and still confirms their spot', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(makePlay({ viewer_state: 'not_joined' }));
	// A direct join reserves the spot as "added"; the player then confirms.
	mock.action('POST', '/api/plays/play-1/join', {
		then: makePlay({
			viewer_state: 'added',
			slots_left: 4,
			added_participants: [participantFor(advanced, { is_viewer: true })]
		})
	});
	mock.action('POST', '/api/plays/play-1/participants/me/confirm', {
		then: makePlay({
			viewer_state: 'confirmed',
			slots_left: 4,
			confirmed_participants: [HOST, participantFor(advanced)]
		})
	});

	await page.goto('/play/play-1');
	await dismissPushPrompt(page);

	await expect(page.getByRole('button', { name: 'Join game' })).toBeVisible();
	await confirmAction(page, 'Join game', 'Join game');

	// Joined but not on the roster yet: the confirm flow still applies
	await expect(page.getByText('Added').first()).toBeVisible();
	await expect(page.getByRole('button', { name: 'Leave game' })).toHaveCount(0);
	await confirmAction(page, 'Confirm spot', 'Confirm spot');

	await expect(page.getByRole('button', { name: 'Leave game' })).toBeVisible();
	await expect(page.getByText('Confirmed').first()).toBeVisible();
});

test('ineligible user requests to join', async ({ page, context }) => {
	await signIn(context, 'seed-li');
	// Seed Low Intermediate is below the play's minimum, so the UI offers a join request.
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
	await dismissPushPrompt(page);

	await expect(page.getByRole('button', { name: 'Request to join' })).toBeVisible();
	await confirmAction(page, 'Request to join', 'Request to join');

	await expect(page.getByRole('button', { name: 'Withdraw request' })).toBeVisible();
	await expect(page.getByText('Requested').first()).toBeVisible();
});

test('added player confirms their spot', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(
		makePlay({
			viewer_state: 'added',
			added_participants: [participantFor(advanced, { is_viewer: true })]
		})
	);
	mock.action('POST', '/api/plays/play-1/participants/me/confirm', {
		then: makePlay({
			viewer_state: 'confirmed',
			slots_left: 4,
			confirmed_participants: [HOST, participantFor(advanced)]
		})
	});

	await page.goto('/play/play-1');
	await dismissPushPrompt(page);

	// The added player sees their own pending state, addressed to them
	await expect(page.getByText('Awaiting your confirmation')).toBeVisible();
	await expect(page.getByText('Awaiting player confirmation')).toHaveCount(0);
	await confirmAction(page, 'Confirm spot', 'Confirm spot');

	await expect(page.getByRole('button', { name: 'Leave game' })).toBeVisible();
});

test('added player declines their spot', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(
		makePlay({
			viewer_state: 'added',
			added_participants: [participantFor(advanced, { is_viewer: true })]
		})
	);
	// "Decline" posts to ?/leave, dropping the viewer back to not-joined.
	mock.action('DELETE', '/api/plays/play-1/participants/me', {
		then: makePlay({ viewer_state: 'not_joined' })
	});

	await page.goto('/play/play-1');
	await dismissPushPrompt(page);

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
	await dismissPushPrompt(page);

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
	await dismissPushPrompt(page);

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
	await dismissPushPrompt(page);

	await confirmAction(page, 'Remove Seed Low Intermediate', 'Remove player');

	await expect(page.getByText('Seed Low Intermediate')).toHaveCount(0);
});

test('full game offers a join request even to eligible players', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(makePlay({ slots_left: 0, viewer_state: 'not_joined' }));

	await page.goto('/play/play-1');
	await dismissPushPrompt(page);

	await expect(page.getByRole('button', { name: 'Request to join' })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Join game' })).toHaveCount(0);
});

test('requester withdraws their request', async ({ page, context }) => {
	await signIn(context, 'seed-li');
	mock.setPlay(
		makePlay({
			level_min: 'HI',
			level_max: 'A',
			viewer_state: 'waitlisted',
			waitlist: [participantFor(li, { is_viewer: true })]
		})
	);
	mock.action('DELETE', '/api/plays/play-1/participants/me', {
		then: makePlay({ level_min: 'HI', level_max: 'A', viewer_state: 'not_joined' })
	});

	await page.goto('/play/play-1');
	await dismissPushPrompt(page);

	await expect(page.getByText('Requested').first()).toBeVisible();
	await confirmAction(page, 'Withdraw request', 'Withdraw request');

	await expect(page.getByRole('button', { name: 'Request to join' })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Withdraw request' })).toHaveCount(0);
});

test('non-hosts see no empty pending sections', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(makePlay({ viewer_state: 'not_joined' }));

	await page.goto('/play/play-1');
	await dismissPushPrompt(page);

	await expect(page.getByRole('button', { name: 'Join game' })).toBeVisible();
	await expect(page.getByText('None yet')).toHaveCount(0);
	await expect(page.getByRole('heading', { name: 'Requests' })).toHaveCount(0);
	await expect(page.getByRole('heading', { name: 'Waitlist' })).toHaveCount(0);
});

test('non-hosts see added players without self-serve actions', async ({ page, context }) => {
	await signIn(context, 'seed-li');
	// Someone else holds an added spot; the viewer has not joined.
	mock.setPlay(
		makePlay({ viewer_state: 'not_joined', added_participants: [participantFor(advanced)] })
	);

	await page.goto('/play/play-1');
	await dismissPushPrompt(page);

	await expect(page.getByText('Seed Advanced')).toBeVisible();
	// The pending-confirmation detail is only shown to the host and the added player
	await expect(page.getByText('Awaiting player confirmation')).toHaveCount(0);
	await expect(page.getByText('Awaiting your confirmation')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Confirm spot' })).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Decline' })).toHaveCount(0);
});

test('require-waitlist play makes eligible players request to join', async ({ page, context }) => {
	await signIn(context, 'seed-advanced');
	mock.setPlay(makePlay({ require_waitlist: true, viewer_state: 'not_joined' }));
	mock.action('POST', '/api/plays/play-1/join', {
		then: makePlay({
			require_waitlist: true,
			viewer_state: 'requested',
			requests: [participantFor(advanced, { is_viewer: true })]
		})
	});

	await page.goto('/play/play-1');
	await dismissPushPrompt(page);

	await expect(page.getByRole('button', { name: 'Join game' })).toHaveCount(0);
	await confirmAction(page, 'Request to join', 'Request to join');

	await expect(page.getByText('Requested').first()).toBeVisible();
	await expect(page.getByRole('button', { name: 'Withdraw request' })).toBeVisible();
});

test('host accepts a join request into the game', async ({ page, context }) => {
	await signIn(context, 'seed-host');
	const requester = participantFor(li);
	mock.setPlay(
		makePlay({
			require_waitlist: true,
			can_manage: true,
			viewer_state: 'creator',
			requests: [requester]
		})
	);
	mock.action('POST', '/api/plays/play-1/participants/:pid/accept', {
		then: makePlay({
			require_waitlist: true,
			can_manage: true,
			viewer_state: 'creator',
			added_participants: [requester]
		})
	});

	await page.goto('/play/play-1');
	await dismissPushPrompt(page);

	await expect(page.getByRole('heading', { name: 'Requests' })).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Add Seed Low Intermediate to the waitlist' })
	).toBeVisible();
	await confirmAction(page, /^Add Seed Low Intermediate$/, 'Add player');

	await expect(page.getByText('Awaiting player confirmation')).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Add Seed Low Intermediate to the waitlist' })
	).toHaveCount(0);
});

test('host parks a join request on the waitlist', async ({ page, context }) => {
	await signIn(context, 'seed-host');
	const requester = participantFor(li);
	mock.setPlay(
		makePlay({
			require_waitlist: true,
			can_manage: true,
			viewer_state: 'creator',
			requests: [requester]
		})
	);
	mock.action('POST', '/api/plays/play-1/participants/:pid/waitlist', {
		then: makePlay({
			require_waitlist: true,
			can_manage: true,
			viewer_state: 'creator',
			waitlist: [requester]
		})
	});

	await page.goto('/play/play-1');
	await dismissPushPrompt(page);

	// Hosts of require-waitlist plays see the waitlist above the requests
	const headings = await page.getByRole('heading').allTextContents();
	expect(headings.indexOf('Waitlist')).toBeLessThan(headings.indexOf('Requests'));

	await confirmAction(page, 'Add Seed Low Intermediate to the waitlist', 'Add to waitlist');

	// Now under Waitlist with the game-add CTA, no longer parkable
	await expect(page.getByText('Seed Low Intermediate')).toBeVisible();
	await expect(
		page.getByRole('button', { name: 'Add Seed Low Intermediate to the waitlist' })
	).toHaveCount(0);
	await expect(page.getByRole('button', { name: /^Add Seed Low Intermediate$/ })).toBeVisible();
});

test('parked player sees the waitlist state and can leave it', async ({ page, context }) => {
	await signIn(context, 'seed-li');
	mock.setPlay(
		makePlay({
			require_waitlist: true,
			viewer_state: 'waitlisted',
			waitlist: [participantFor(li, { is_viewer: true })]
		})
	);
	mock.action('DELETE', '/api/plays/play-1/participants/me', {
		then: makePlay({ require_waitlist: true, viewer_state: 'not_joined' })
	});

	await page.goto('/play/play-1');
	await dismissPushPrompt(page);

	await expect(page.getByText('On waitlist').first()).toBeVisible();
	await confirmAction(page, 'Leave waitlist', 'Leave waitlist');

	await expect(page.getByRole('button', { name: 'Request to join' })).toBeVisible();
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
	await dismissPushPrompt(page);
	await confirmAction(page, 'Cancel game', 'Cancel game');

	await expect(page).toHaveURL(/\/play\/play-1$/);
	await expect(page.getByText('Cancelled').first()).toBeVisible();
	await expect(page.getByRole('button', { name: 'Join game' })).toHaveCount(0);
});
