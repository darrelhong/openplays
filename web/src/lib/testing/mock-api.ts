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
 * memory and mutated by the action endpoints, so a join is reflected on the
 * next page `load`, exercising the full UI round-trip.
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
	[key: string]: unknown;
	confirmed_participants: Participant[];
	added_participants: Participant[];
	waitlist: Participant[];
	confirmed_count: number;
	added_count: number;
	waitlist_count: number;
	slots_left: number;
	viewer_state: string;
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

export type MockApi = {
	/** Mutable play store, keyed by id. Seed/inspect directly in tests. */
	plays: Map<string, Play>;
	close: () => Promise<void>;
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

/** Project a stored play for a viewer: creators can manage; others keep stored state. */
function viewerProjection(play: Play, userId: string | null): Play {
	const view = structuredClone(play);
	if (userId && userId === play.created_by) {
		view.can_manage = true;
		view.viewer_state = 'creator';
	}
	return view;
}

let nextParticipantId = 100;

export function startMockApi(port = Number(process.env.MOCK_API_PORT ?? 8080)): Promise<MockApi> {
	const plays = new Map<string, Play>();

	const server: Server = createServer((req, res) => {
		const { pathname } = new URL(req.url ?? '', `http://localhost:${port}`);
		const userId = cookieUserId(req);

		if (req.method === 'GET' && pathname === '/api/me/') {
			const user = userId ? SEED_USERS[userId] : null;
			return user ? sendJSON(res, 200, user) : problem(res, 401, 'unauthenticated');
		}

		const playId = pathname.match(/^\/api\/plays\/([^/]+)$/)?.[1];
		if (req.method === 'GET' && playId) {
			const play = plays.get(playId);
			return play
				? sendJSON(res, 200, viewerProjection(play, userId))
				: problem(res, 404, 'play not found');
		}

		const joinId = pathname.match(/^\/api\/plays\/([^/]+)\/join$/)?.[1];
		if (req.method === 'POST' && joinId) {
			const play = plays.get(joinId);
			if (!play) return problem(res, 404, 'play not found');
			const user = userId ? SEED_USERS[userId] : null;
			if (!user) return problem(res, 401, 'unauthenticated');

			play.confirmed_participants.push({
				id: nextParticipantId++,
				display_name: user.display_name,
				rating_code: user.sports_profile?.badminton?.level ?? null,
				is_guest: false,
				is_host: false
			});
			play.confirmed_count += 1;
			play.slots_left = Math.max(play.slots_left - 1, 0);
			play.viewer_state = 'confirmed';
			return sendJSON(res, 200, {});
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
