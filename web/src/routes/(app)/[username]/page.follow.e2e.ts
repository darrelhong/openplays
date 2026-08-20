import { expect, test, type BrowserContext } from '@playwright/test';
import { SEED_USERS, startMockApi, type MockApi } from '$lib/testing/mock-api';
import { dismissPushPrompt } from '$lib/testing/e2e';
import type { components } from '$lib/api/types.gen';

type PublicUserProfile = components['schemas']['PublicUserProfile'];
type UserSummary = components['schemas']['UserSummary'];

/**
 * Follow flow: the profile follow button, the follow-request inbox, and the
 * follower-list privacy gate.
 *
 * Each test declares its own mock responses below; mock-api.ts holds only
 * shapes. The `profile` / `inbox` variables are the spec's own state — the
 * follow/accept/decline responders mutate them so the post-action reload (and
 * a viewer switch) reflects the new follow state.
 */
let mock: MockApi;

/** The profile served by `GET /api/users/:username`. */
let profile: PublicUserProfile;
/** The followee's pending inbox served by `GET /api/me/follow-requests`. */
let inbox: UserSummary[];
/** The currently signed-in user (what `PUT .../follow` enqueues when restricted). */
let viewer: keyof typeof SEED_USERS;

test.beforeAll(async () => {
	mock = await startMockApi();
});

test.afterAll(async () => {
	await mock?.close();
});

test.beforeEach(() => {
	mock.reset();
	profile = {} as PublicUserProfile;
	inbox = [];
});

async function signIn(context: BrowserContext, userId: keyof typeof SEED_USERS) {
	viewer = userId;
	await context.addCookies([{ name: 'session', value: userId, domain: 'localhost', path: '/' }]);
	mock.setUser(SEED_USERS[userId]!);
}

function setProfile(overrides: Partial<PublicUserProfile> = {}) {
	profile = {
		id: 'seed-host',
		display_name: 'Seed Host',
		username: 'seedhost',
		props: [],
		sports: [],
		rostered_play_count: 0,
		follower_count: 0,
		following_count: 0,
		restrict_follow: false,
		viewer_follow_state: 'none',
		...overrides
	};
}

function summaryFor(userId: keyof typeof SEED_USERS): UserSummary {
	const user = SEED_USERS[userId]!;
	return { id: user.id, display_name: user.display_name, username: user.username };
}

/** Profile + follow/unfollow routes. Mutates `profile` so the reload reflects it. */
function registerProfileRoutes() {
	mock.on('GET', '/api/users/:username', () => ({ json: profile }));
	mock.on('GET', '/api/users/:username/shoutouts', () => ({
		json: { items: [], total: 0, has_more: false, next_cursor: null }
	}));
	mock.on('PUT', '/api/users/:username/follow', () => {
		if (profile.restrict_follow) {
			profile.viewer_follow_state = 'pending';
			inbox = [summaryFor(viewer), ...inbox];
		} else {
			profile.viewer_follow_state = 'following';
		}
		return { json: { status: profile.restrict_follow ? 'pending' : 'accepted' } };
	});
	mock.on('DELETE', '/api/users/:username/follow', () => {
		profile.viewer_follow_state = 'none';
		return { status: 204 };
	});
}

/** Follow-request inbox + accept/decline routes. Mutates `inbox`. */
function registerInboxRoutes() {
	mock.on('GET', '/api/me/follow-requests', () => ({
		json: { items: inbox, total: inbox.length, has_more: false, next_cursor: null }
	}));
	mock.on('POST', '/api/me/follow-requests/:user_id/accept', ({ params }) => {
		inbox = inbox.filter((u) => u.id !== params.user_id);
		return { status: 200 };
	});
	mock.on('POST', '/api/me/follow-requests/:user_id/decline', ({ params }) => {
		inbox = inbox.filter((u) => u.id !== params.user_id);
		return { status: 200 };
	});
}

test('following an unrestricted user takes effect immediately', async ({ page, context }) => {
	await signIn(context, 'seed-li');
	setProfile({ restrict_follow: false, viewer_follow_state: 'none' });
	registerProfileRoutes();

	await page.goto('/seedhost');
	await dismissPushPrompt(page);

	await expect(page.getByRole('button', { name: 'Follow', exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'Follow', exact: true }).click();

	await expect(page.getByRole('button', { name: 'Following', exact: true })).toBeVisible();
});

test('following a restricted user shows Requested and lands in the inbox', async ({
	page,
	context
}) => {
	await signIn(context, 'seed-li');
	setProfile({ restrict_follow: true, viewer_follow_state: 'none' });
	registerProfileRoutes();
	registerInboxRoutes();

	await page.goto('/seedhost');
	await dismissPushPrompt(page);

	await expect(page.getByRole('button', { name: 'Request to follow', exact: true })).toBeVisible();
	await page.getByRole('button', { name: 'Request to follow', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Requested', exact: true })).toBeVisible();

	// The followee's inbox now holds the request.
	await signIn(context, 'seed-host');
	await page.goto('/profile/follow-requests');
	await dismissPushPrompt(page);
	await expect(page.getByText('Seed Low Intermediate')).toBeVisible();
});

test('accepting a request flips the requester to Following', async ({ page, context }) => {
	await signIn(context, 'seed-host');
	inbox = [summaryFor('seed-li')];
	registerInboxRoutes();

	await page.goto('/profile/follow-requests');
	await dismissPushPrompt(page);
	await expect(page.getByText('Seed Low Intermediate')).toBeVisible();

	await page.getByRole('button', { name: 'Accept', exact: true }).click();
	await expect(page.getByText('No follow requests')).toBeVisible();

	// The requester is now following.
	await signIn(context, 'seed-li');
	setProfile({ restrict_follow: true, viewer_follow_state: 'following' });
	registerProfileRoutes();
	await page.goto('/seedhost');
	await dismissPushPrompt(page);
	await expect(page.getByRole('button', { name: 'Following', exact: true })).toBeVisible();
});

test('declining removes the request and the requester can request again', async ({
	page,
	context
}) => {
	await signIn(context, 'seed-host');
	inbox = [summaryFor('seed-li')];
	registerInboxRoutes();

	await page.goto('/profile/follow-requests');
	await dismissPushPrompt(page);
	await expect(page.getByText('Seed Low Intermediate')).toBeVisible();

	await page.getByRole('button', { name: 'Decline', exact: true }).click();
	await expect(page.getByText('No follow requests')).toBeVisible();

	// The requester can request again.
	await signIn(context, 'seed-li');
	setProfile({ restrict_follow: true, viewer_follow_state: 'none' });
	registerProfileRoutes();
	await page.goto('/seedhost');
	await dismissPushPrompt(page);
	await expect(page.getByRole('button', { name: 'Request to follow', exact: true })).toBeVisible();
});

test('a stranger sees follower counts but gets the locked list state', async ({
	page,
	context
}) => {
	await signIn(context, 'seed-li');
	setProfile({ viewer_follow_state: 'none', follower_count: 42, following_count: 7 });
	registerProfileRoutes();
	mock.on('GET', '/api/users/:username/followers', () => ({
		status: 403,
		json: { status: 403, detail: 'followers are private' }
	}));
	mock.on('GET', '/api/users/:username/following', () => ({
		status: 403,
		json: { status: 403, detail: 'following is private' }
	}));

	await page.goto('/seedhost');
	await dismissPushPrompt(page);

	// Counts render as plain text, not links.
	await expect(page.getByText('42', { exact: true })).toBeVisible();
	await expect(page.getByText('7', { exact: true })).toBeVisible();
	await expect(page.locator('a[href="/seedhost/followers"]')).toHaveCount(0);
	await expect(page.locator('a[href="/seedhost/following"]')).toHaveCount(0);

	// Direct nav to the list shows the locked state instead of a raw error.
	await page.goto('/seedhost/followers');
	await dismissPushPrompt(page);
	await expect(page.getByText('This list is private')).toBeVisible();
});
