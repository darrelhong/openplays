# sv

Everything you need to build a Svelte project, powered by [`sv`](https://github.com/sveltejs/cli).

## Creating a project

If you're seeing this, you've probably already done this step. Congrats!

```sh
# create a new project
npx sv create my-app
```

To recreate this project with the same configuration:

```sh
# recreate this project
pnpm dlx sv@0.14.1 create --template minimal --types ts --add prettier eslint vitest="usages:component,unit" playwright mcp="ide:opencode,vscode+setup:local" --install pnpm web
```

## Developing

Once you've installed dependencies with `pnpm install`, start a development server:

```sh
pnpm dev
```

### Local Dev Login

For OAuth-free local login, enable the matching server and web flags:

```sh
DEV_AUTH_ENABLED=true
```

Seed users from the server package:

```sh
cd ../server
go run ./tools/seeddev/
```

Then start the API and web dev servers and open `/__dev/login`. The page only loads in SvelteKit dev mode when `DEV_AUTH_ENABLED=true`; the backend `/api/dev/*` routes are only registered when the same flag is enabled on the API server.

## API Types

TypeScript types are auto-generated from the Go API's OpenAPI spec. The generated file is `src/lib/api/types.gen.ts`.

To regenerate after changing the API:

```sh
# 1. Start the Go API server (in the server/ directory)
cd ../server && go run ./cmd/api/

# 2. Generate types (in the web/ directory)
pnpm gen:types
```

This fetches `http://localhost:8080/openapi.json` and generates typed request/response interfaces. The config is in `redocly.yaml`.

## Testing

Unit tests (Vitest):

```sh
pnpm test:unit
```

End-to-end tests (Playwright). First install the browser once:

```sh
pnpm exec playwright install chromium
```

The e2e specs do **not** need the Go API running — they mock the backend
in-process. Each spec starts a small `node:http` server (see
`src/lib/testing/mock-api.ts`) on the same port the SvelteKit build talks to,
serving `/api/me/`, play details, and the join/leave/roster actions. Auth is a
cookie whose value is the seed user id. Specs live next to the route they cover
as `*.e2e.ts` (e.g. `src/routes/play/[id]/page.roster.e2e.ts`).

```sh
pnpm test:e2e          # headless run (build + preview happens automatically)
pnpm test:e2e:ui       # interactive UI mode (watch, step, time-travel)
```

`API_BASE_URL` is baked at build time (defaults to `http://localhost:8080`), and
the mock listens on that same port. **If your Go API is already running on
8080**, point both at a free port so they don't collide:

```sh
API_BASE_URL=http://localhost:8099 COOKIE_SECURE=false MOCK_API_PORT=8099 pnpm test:e2e:ui
```

Add `--headed` to watch the browser, or scope to a file by name, e.g.
`pnpm test:e2e page.roster`.

## Building

To create a production version of your app:

```sh
npm run build
```

You can preview the production build with `npm run preview`.

> To deploy your app, you may need to install an [adapter](https://svelte.dev/docs/kit/adapters) for your target environment.
