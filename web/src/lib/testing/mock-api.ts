import { createServer, type IncomingMessage, type Server, type ServerResponse } from 'node:http';

/**
 * Configurable in-process mock of the OpenPlays backend for e2e specs.
 *
 * The web app is SSR + server form actions: the `/api/me/` probe in
 * hooks.server.ts, each page `load`, and the join/leave/accept/remove actions
 * all run in the SvelteKit server process — never the browser. So a browser
 * service worker can't intercept them; we mock at the network boundary the
 * SvelteKit server talks to by listening on the baked `API_BASE_URL` port
 * (8080 in CI via .env.example).
 *
 * Design: the mock holds NO business logic. Each test declares what its
 * endpoints return — the current user (`setUser`), the play served by `load`
 * (`setPlay`), and what an action does (`action`, which optionally transitions
 * the play for the post-action reload). Only the shared fixtures and the two
 * read endpoints every test needs are centralised here. Register one instance
 * per spec in `beforeAll`, `reset()` in `beforeEach`, `close()` in `afterAll`.
 */

export type Participant = {
	id: number;
	display_name: string;
	username?: string;
	rating_code: string | null;
	is_guest: boolean;
	is_host: boolean;
};

export type Play = {
	id: string;
	created_by: string | null;
	can_manage: boolean;
	viewer_state: string;
	confirmed_participants: Participant[];
	added_participants: Participant[];
	waitlist: Participant[];
	confirmed_count: number;
	added_count: number;
	waitlist_count: number;
	slots_left: number;
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

// ---------------------------------------------------------------------------
// Shared fixtures (centralised — reused across specs)
// ---------------------------------------------------------------------------

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

/** The standard host roster row used by `makePlay`. */
export const HOST: Participant = {
	id: 39,
	display_name: 'Seed Host',
	username: 'seedhost',
	rating_code: 'HB',
	is_guest: false,
	is_host: true
};

let nextParticipantId = 100;

/** Build a participant row from a seed user (badminton rating). */
export function participantFor(user: SeedUser, overrides: Partial<Participant> = {}): Participant {
	return {
		id: nextParticipantId++,
		display_name: user.display_name,
		username: user.username,
		rating_code: user.sports_profile.badminton?.level ?? null,
		is_guest: false,
		is_host: false,
		...overrides
	};
}

/**
 * A user-created play. Defaults: hosted by Seed Host, LB–A, open slots, viewer
 * not joined. Counts are derived from the roster arrays so callers only set the
 * arrays. Override anything (e.g. `viewer_state`, `can_manage`, `waitlist`).
 */
export function makePlay(overrides: Partial<Play> = {}): Play {
	const play: Play = {
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
		visibility: 'public',
		contacts: null,
		meta: null,
		source: 'user',
		created_by: 'seed-host',
		creator_display_name: 'Seed Host',
		creator_username: 'seedhost',
		can_manage: false,
		viewer_state: 'not_joined',
		confirmed_participants: [HOST],
		added_participants: [],
		waitlist: [],
		confirmed_count: 0,
		added_count: 0,
		waitlist_count: 0,
		...overrides
	};
	play.confirmed_count = play.confirmed_participants.length;
	play.added_count = play.added_participants.length;
	play.waitlist_count = play.waitlist.length;
	return play;
}

// ---------------------------------------------------------------------------
// Configurable server
// ---------------------------------------------------------------------------

export type MockResponse = { status?: number; json?: unknown };
export type RequestCtx = { params: Record<string, string>; url: URL; req: IncomingMessage };
export type Responder =
	| MockResponse
	| ((ctx: RequestCtx) => MockResponse | void | Promise<MockResponse | void>);

export type MockApi = {
	/** Register/override a route. Patterns support `:param` segments. Last match wins. */
	on(method: string, path: string, responder: Responder): MockApi;
	/** What `GET /api/me/` returns (null → 401, i.e. signed out). */
	setUser(user: SeedUser | null): MockApi;
	/** What `GET /api/plays/:id` returns for the current load/reload. */
	setPlay(play: Play | null): MockApi;
	/**
	 * Register an action endpoint. Responds `status` (default 200); if `then` is
	 * given, the served play transitions to it so the post-action reload reflects it.
	 */
	action(method: string, path: string, opts?: { status?: number; then?: Play }): MockApi;
	/** Clear per-test overrides and reset served user/play. Call in `beforeEach`. */
	reset(): MockApi;
	close(): Promise<void>;
};

type Route = { method: string; path: string; responder: Responder };

/** Match `/api/plays/:id` against a concrete path, capturing params; null if no match. */
function matchPath(pattern: string, pathname: string): Record<string, string> | null {
	const pat = pattern.split('/');
	const seg = pathname.split('/');
	if (pat.length !== seg.length) return null;
	const params: Record<string, string> = {};
	for (let i = 0; i < pat.length; i++) {
		const p = pat[i]!;
		const s = seg[i]!;
		if (p.startsWith(':')) params[p.slice(1)] = s;
		else if (p !== s) return null;
	}
	return params;
}

export function mockApiPort() {
	if (process.env.MOCK_API_PORT) return Number(process.env.MOCK_API_PORT);
	if (process.env.API_BASE_URL) {
		const url = new URL(process.env.API_BASE_URL);
		return Number(url.port || (url.protocol === 'https:' ? 443 : 80));
	}
	return 8080;
}

export function startMockApi(port = mockApiPort()): Promise<MockApi> {
	const state: { user: SeedUser | null; play: Play | null } = { user: null, play: null };
	const overrides: Route[] = [];

	// The two reads every spec needs; lowest priority, overridable via `on`.
	const defaults: Route[] = [
		{
			method: 'GET',
			path: '/api/me/',
			responder: () => (state.user ? { json: state.user } : { status: 401 })
		},
		{
			method: 'GET',
			path: '/api/plays/:id',
			responder: () => (state.play ? { json: state.play } : { status: 404 })
		}
	];

	function resolve(
		method: string,
		pathname: string
	): { responder: Responder; params: Record<string, string> } | null {
		for (let i = overrides.length - 1; i >= 0; i--) {
			const route = overrides[i]!;
			if (route.method !== method) continue;
			const params = matchPath(route.path, pathname);
			if (params) return { responder: route.responder, params };
		}
		for (const route of defaults) {
			if (route.method !== method) continue;
			const params = matchPath(route.path, pathname);
			if (params) return { responder: route.responder, params };
		}
		return null;
	}

	const server: Server = createServer(async (req, res) => {
		const url = new URL(req.url ?? '', `http://localhost:${port}`);
		const match = resolve(req.method ?? 'GET', url.pathname);
		if (!match)
			return send(res, 404, { detail: `no mock route for ${req.method} ${url.pathname}` });
		try {
			const out =
				typeof match.responder === 'function'
					? await match.responder({ params: match.params, url, req })
					: match.responder;
			const response = out ?? {};
			send(res, response.status ?? 200, response.json ?? {});
		} catch (error) {
			send(res, 500, {
				detail: error instanceof Error ? error.message : 'mock api responder failed'
			});
		}
	});

	function send(res: ServerResponse, status: number, json: unknown) {
		res.writeHead(status, { 'content-type': 'application/json' });
		res.end(JSON.stringify(json));
	}

	const api: MockApi = {
		on(method, path, responder) {
			overrides.push({ method, path, responder });
			return api;
		},
		setUser(user) {
			state.user = user;
			return api;
		},
		setPlay(play) {
			state.play = play;
			return api;
		},
		action(method, path, opts = {}) {
			return api.on(method, path, () => {
				if (opts.then !== undefined) state.play = opts.then;
				return { status: opts.status ?? 200 };
			});
		},
		reset() {
			overrides.length = 0;
			state.user = null;
			state.play = null;
			return api;
		},
		close: () => new Promise<void>((res, rej) => server.close((err) => (err ? rej(err) : res())))
	};

	return new Promise<MockApi>((resolve, reject) => {
		server.once('error', reject);
		server.listen(port, () => resolve(api));
	});
}
