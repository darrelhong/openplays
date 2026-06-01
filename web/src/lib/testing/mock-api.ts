import { createServer, type IncomingMessage, type Server, type ServerResponse } from 'node:http';

/**
 * Reusable in-process mock of the OpenPlays backend for e2e specs.
 *
 * The web app is SSR + server form actions: the `/api/me/` probe in
 * hooks.server.ts, each page `load`, and the join/leave/accept/remove actions
 * all run in the SvelteKit server process — never the browser. So a browser
 * service worker can't intercept them; we mock at the network boundary the
 * SvelteKit server talks to by listening on the baked `API_BASE_URL` port
 * (8080 in CI via .env.example).
 *
 * Each spec starts its own instance in `beforeAll` and closes it in `afterAll`,
 * mirroring the pattern in `src/routes/page.timezone.e2e.ts`. State is held in
 * memory and mutated by the action endpoints, so a join/leave/accept is
 * reflected on the next page `load`, exercising the full UI round-trip.
 *
 * Auth convention: the `session` cookie value IS the seed user id
 * (e.g. `session=seed-advanced`); `/api/me/` resolves the user from it.
 */

export type Participant = {
	id: number;
	display_name: string;
	rating_code: string | null;
	is_guest: boolean;
	is_host: boolean;
};

export type Play = {
	id: string;
	created_by: string | null;
	confirmed_participants: Participant[];
	added_participants: Participant[];
	waitlist: Participant[];
	confirmed_count: number;
	added_count: number;
	waitlist_count: number;
	slots_left: number;
	viewer_state: string;
	[key: string]: unknown;
};

export type SeedUser = {
	id: string;
	email: string;
	username: string;
	display_name: string;
	status: string;
	sports_profile: Record<string, { level: string }>;
	created_at: string;
	updated_at: string;
};

export const SEED_USERS: Record<string, SeedUser> = {
	'seed-advanced': {
		id: 'seed-advanced',
		email: 'seed-advanced@example.test',
		username: 'seedadvanced',
		display_name: 'Seed Advanced',
		status: 'active',
		sports_profile: { badminton: { level: 'A' } },
		created_at: '2026-05-26T09:02:17Z',
		updated_at: '2026-05-26T09:02:17Z'
	},
	'seed-host': {
		id: 'seed-host',
		email: 'seed-host@example.test',
		username: 'seedhost',
		display_name: 'Seed Host',
		status: 'active',
		sports_profile: { badminton: { level: 'HB' }, tennis: { level: '3.5' } },
		created_at: '2026-05-26T09:02:17Z',
		updated_at: '2026-05-26T09:02:17Z'
	},
	'seed-li': {
		id: 'seed-li',
		email: 'seed-li@example.test',
		username: 'seedli',
		display_name: 'Seed Low Intermediate',
		status: 'active',
		sports_profile: { badminton: { level: 'LI' } },
		created_at: '2026-05-26T09:02:17Z',
		updated_at: '2026-05-26T09:02:17Z'
	}
};

let nextParticipantId = 100;

/** A user-created play. Defaults are eligible for a direct join by Seed Advanced (LB–A). */
export function makePlay(overrides: Partial<Play> = {}): Play {
	return {
		id: 'play-1',
		listing_type: 'play',
		sport: 'badminton',
		game_type: 'doubles',
		host_name: 'Seed Host',
		starts_at: '2026-07-01T04:00:00Z',
		ends_at: '2026-07-01T06:00:00Z',
		timezone: 'Asia/Singapore',
		venue: 'Test Hall',
		venue_name: 'Test Hall',
		level_min: 'LB',
		level_max: 'A',
		fee: 900,
		currency: 'SGD',
		max_players: 6,
		slots_left: 5,
		courts: 2,
		contacts: null,
		meta: null,
		source: 'user',
		created_by: 'seed-host',
		creator_display_name: 'Seed Host',
		creator_username: 'seedhost',
		confirmed_participants: [
			{ id: 39, display_name: 'Seed Host', rating_code: 'HB', is_guest: false, is_host: true }
		],
		added_participants: [],
		waitlist: [],
		viewer_state: 'not_joined',
		can_manage: false,
		confirmed_count: 1,
		added_count: 0,
		waitlist_count: 0,
		...overrides
	};
}

/** Build a participant row from a seed user (badminton rating). */
export function participantFor(user: SeedUser, overrides: Partial<Participant> = {}): Participant {
	return {
		id: nextParticipantId++,
		display_name: user.display_name,
		rating_code: user.sports_profile.badminton?.level ?? null,
		is_guest: false,
		is_host: false,
		...overrides
	};
}

export type MockApi = {
	/** Mutable play store, keyed by id. Seed/inspect directly in tests. */
	plays: Map<string, Play>;
	close: () => Promise<void>;
};

// Badminton skill ladder, mirrors the frontend's canDirectJoin ordering.
const BADMINTON_ORD: Record<string, number> = {
	LB: 10,
	MB: 20,
	HB: 30,
	LI: 40,
	MI: 50,
	HI: 60,
	A: 70
};

function cookieUserId(req: IncomingMessage): string | null {
	const value = (req.headers.cookie ?? '').match(/(?:^|;\s*)session=([^;]+)/)?.[1];
	return value ? decodeURIComponent(value) : null;
}

function sendJSON(res: ServerResponse, status: number, body: unknown) {
	res.writeHead(status, { 'content-type': 'application/json' });
	res.end(body === undefined ? '' : JSON.stringify(body));
}

function problem(res: ServerResponse, status: number, detail: string) {
	res.writeHead(status, { 'content-type': 'application/problem+json' });
	res.end(JSON.stringify({ status, detail, title: detail }));
}

/** Whether the user can join directly (badminton only — the sport used in tests). */
function canDirectJoin(play: Play, user: SeedUser): boolean {
	if (play.slots_left <= 0) return false;
	const level = user.sports_profile.badminton?.level;
	const ord = level ? BADMINTON_ORD[level] : undefined;
	if (ord == null) return false;
	const min = typeof play.level_min === 'string' ? BADMINTON_ORD[play.level_min] : undefined;
	if (min != null && ord < min) return false;
	const max = typeof play.level_max === 'string' ? BADMINTON_ORD[play.level_max] : undefined;
	if (max != null && ord > max) return false;
	return true;
}

/** Project a stored play for a viewer: derive counts, and mark creators as managers. */
function viewerProjection(play: Play, userId: string | null): Play {
	const view = structuredClone(play);
	view.confirmed_count = view.confirmed_participants.length;
	view.added_count = view.added_participants.length;
	view.waitlist_count = view.waitlist.length;
	if (userId && userId === play.created_by) {
		view.can_manage = true;
		view.viewer_state = 'creator';
	}
	return view;
}

/** Remove a participant by id from every roster bucket; returns the bucket it was in. */
function removeParticipantById(
	play: Play,
	id: number
): 'confirmed_participants' | 'added_participants' | 'waitlist' | null {
	for (const bucket of ['confirmed_participants', 'added_participants', 'waitlist'] as const) {
		const idx = play[bucket].findIndex((p) => p.id === id);
		if (idx >= 0) {
			play[bucket].splice(idx, 1);
			return bucket;
		}
	}
	return null;
}

export function startMockApi(port = Number(process.env.MOCK_API_PORT ?? 8080)): Promise<MockApi> {
	const plays = new Map<string, Play>();

	const server: Server = createServer((req, res) => {
		const { pathname } = new URL(req.url ?? '', `http://localhost:${port}`);
		const userId = cookieUserId(req);
		const user = userId ? SEED_USERS[userId] : null;

		if (req.method === 'GET' && pathname === '/api/me/') {
			return user ? sendJSON(res, 200, user) : problem(res, 401, 'unauthenticated');
		}

		// Current-user roster actions ----------------------------------------
		const confirmMe = pathname.match(/^\/api\/plays\/([^/]+)\/participants\/me\/confirm$/);
		if (req.method === 'POST' && confirmMe) {
			const play = plays.get(confirmMe[1]!);
			if (!play) return problem(res, 404, 'play not found');
			if (!user) return problem(res, 401, 'unauthenticated');
			const idx = play.added_participants.findIndex((p) => p.display_name === user.display_name);
			if (idx < 0) return problem(res, 404, 'not an added participant');
			const [participant] = play.added_participants.splice(idx, 1);
			play.confirmed_participants.push(participant!);
			play.slots_left = Math.max(play.slots_left - 1, 0);
			play.viewer_state = 'confirmed';
			return sendJSON(res, 200, {});
		}

		const leaveMe = pathname.match(/^\/api\/plays\/([^/]+)\/participants\/me$/);
		if (req.method === 'DELETE' && leaveMe) {
			const play = plays.get(leaveMe[1]!);
			if (!play) return problem(res, 404, 'play not found');
			if (!user) return problem(res, 401, 'unauthenticated');
			const wasConfirmed = play.confirmed_participants.some(
				(p) => p.display_name === user.display_name
			);
			for (const bucket of ['confirmed_participants', 'added_participants', 'waitlist'] as const) {
				play[bucket] = play[bucket].filter((p) => p.display_name !== user.display_name);
			}
			if (wasConfirmed) play.slots_left += 1;
			play.viewer_state = 'not_joined';
			return sendJSON(res, 200, {});
		}

		// Host roster management ----------------------------------------------
		const accept = pathname.match(/^\/api\/plays\/([^/]+)\/participants\/(\d+)\/accept$/);
		if (req.method === 'POST' && accept) {
			const play = plays.get(accept[1]!);
			if (!play) return problem(res, 404, 'play not found');
			const id = Number(accept[2]);
			const idx = play.waitlist.findIndex((p) => p.id === id);
			if (idx < 0) return problem(res, 404, 'not on waitlist');
			const [participant] = play.waitlist.splice(idx, 1);
			play.added_participants.push(participant!);
			return sendJSON(res, 200, {});
		}

		const removeById = pathname.match(/^\/api\/plays\/([^/]+)\/participants\/(\d+)$/);
		if (req.method === 'DELETE' && removeById) {
			const play = plays.get(removeById[1]!);
			if (!play) return problem(res, 404, 'play not found');
			const bucket = removeParticipantById(play, Number(removeById[2]));
			if (!bucket) return problem(res, 404, 'participant not found');
			if (bucket === 'confirmed_participants') play.slots_left += 1;
			return sendJSON(res, 200, {});
		}

		// Join (direct or waitlist, decided by eligibility) -------------------
		const join = pathname.match(/^\/api\/plays\/([^/]+)\/join$/);
		if (req.method === 'POST' && join) {
			const play = plays.get(join[1]!);
			if (!play) return problem(res, 404, 'play not found');
			if (!user) return problem(res, 401, 'unauthenticated');
			if (canDirectJoin(play, user)) {
				play.confirmed_participants.push(participantFor(user));
				play.slots_left = Math.max(play.slots_left - 1, 0);
				play.viewer_state = 'confirmed';
			} else {
				play.waitlist.push(participantFor(user));
				play.viewer_state = 'waitlisted';
			}
			return sendJSON(res, 200, {});
		}

		// Cancel a play (host) ------------------------------------------------
		const cancel = pathname.match(/^\/api\/plays\/([^/]+)$/);
		if (req.method === 'DELETE' && cancel) {
			const play = plays.get(cancel[1]!);
			if (!play) return problem(res, 404, 'play not found');
			play.cancelled_at = '2026-06-01T00:00:00Z';
			return sendJSON(res, 200, {});
		}

		// Play details --------------------------------------------------------
		const playId = pathname.match(/^\/api\/plays\/([^/]+)$/)?.[1];
		if (req.method === 'GET' && playId) {
			const play = plays.get(playId);
			return play
				? sendJSON(res, 200, viewerProjection(play, userId))
				: problem(res, 404, 'play not found');
		}

		return problem(res, 404, `no mock route for ${req.method} ${pathname}`);
	});

	return new Promise<MockApi>((resolve, reject) => {
		server.once('error', reject);
		server.listen(port, () => {
			resolve({
				plays,
				close: () =>
					new Promise<void>((res, rej) => server.close((err) => (err ? rej(err) : res())))
			});
		});
	});
}
