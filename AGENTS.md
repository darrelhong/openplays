# AGENTS

Command reference for the OpenPlays monorepo. `server/` is the Go API +
Telegram listener; `web/` is the SvelteKit frontend. See `web/AGENTS.md` for
frontend-specific conventions.

## Server (`cd server`)

Go 1.26, Chi + Huma, SQLite via sqlc + goose.

```bash
go run ./cmd/api/          # run the API server (:8080)
go run ./cmd/listener/     # run the Telegram listener + worker
go test ./...              # run all server tests
go build ./...             # compile everything

go run ./tools/seeddev/    # seed local dev users (seed-host, seed-li, ...)
```

### sqlc — generated DB layer

Queries live in `db/queries/`, schema/migrations in `db/migrations/`, config in
`sqlc.yaml`. After editing either, regenerate the typed Go code in `internal/db`:

```bash
sqlc generate
```

Then build/test to pick up the new types.

## Web (`cd web`)

TypeScript, SvelteKit, pnpm. Talks to the Go API via the typed client in
`$lib/api/client.ts`.

```bash
pnpm dev                   # dev server
pnpm check                 # svelte-check / typecheck
pnpm lint                  # prettier --check + eslint (no changes)
pnpm format                # prettier --write (auto-fix formatting)
pnpm exec eslint . --fix   # eslint auto-fix

pnpm test:unit             # Vitest unit/component tests
pnpm test:e2e              # Playwright e2e (mocks the backend; no API needed)
pnpm test:e2e:ui           # Playwright UI mode
```

Run `pnpm format` and `pnpm lint` after changing frontend code.

### gen:types — API types from OpenAPI

The frontend's request/response types (`$lib/api/types.gen.ts`) are generated
from the Go API's OpenAPI spec. After adding/changing an API endpoint:

```bash
# 1. start the Go API so the spec is served at :8080/openapi.json
cd ../server && go run ./cmd/api/
# 2. regenerate (in web/)
pnpm gen:types
```

## Typical change flows

- **DB change**: edit `db/queries` or `db/migrations` → `sqlc generate` → `go test ./...`
- **API change**: edit handler → run API → `pnpm gen:types` → use the new types in `web/`
- **Frontend change**: edit → `pnpm format` → `pnpm lint` → `pnpm check`
