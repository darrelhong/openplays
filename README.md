# OpenPlays

Simple platform to discover and organise sports sessions and activities.

## Features

- **Telegram ingestion** -- listens to group messages, extracts session details (sport, date/time, venue, fee, skill level, slots) via LLM
- **Venue normalization** -- geocoding and venue database for consistent location data
- **Distance sorting** -- sort sessions by proximity to any venue using Haversine distance
- **Dynamic filters** -- filter sessions by sport, date, venue, fee, skill level, slots

## Stack

| Layer | Tech |
|-------|------|
| Backend | Go, Chi, Huma, SQLite, sqlc, goose |
| Frontend | SvelteKit, Bits UI, UnoCSS |
| Ingestion | Telegram Bot API, LLM parsing, Go worker |

### Version Management

[asdf](https://asdf-vm.com/) is used for version management.

Add plugins for golang, nodejs, and pnpm

`asdf install`

## Structure

```
server/     Go API + Telegram listener + migrations + tools
web/        SvelteKit frontend
deploy/     Systemd units, Caddyfile, setup/deploy scripts
```

## Local object storage

The development Compose stack runs MinIO, creates the `openplays-local` bucket,
and makes its objects publicly readable. The Go API and SvelteKit app still run
directly on the host.

On macOS, start a Docker-compatible VM with Colima, then start the stack:

```bash
colima start --cpu 2 --memory 2 --disk 10
docker-compose -f compose.dev.yml up -d
```

Copy `server/.env.example` to `server/.env` if needed. Its object-store values
already match this stack. MinIO's API is available at
`http://localhost:9000`; its console is at `http://localhost:9001` with username
`openplays` and password `openplays-local-secret`.

Stop the services while preserving uploaded files:

```bash
docker-compose -f compose.dev.yml down
```

Add `--volumes` to that command when you want to delete all local objects and
recreate the bucket from scratch. If Docker's Compose CLI plugin is configured,
the equivalent `docker compose` commands work too.

## CI/CD

GitHub Actions runs on every push/PR:

- **test.yml** -- Go tests + SvelteKit unit tests + `svelte-check` (PRs and deploys)
- **deploy.yml** -- runs tests, builds, rsyncs to VPS, migrates DB, restarts services (push to main)

## Deployment

Single VPS running all services behind Caddy (auto-HTTPS):

```
Cloudflare → Caddy (:443) → SvelteKit (:3000)
                           → Go API (:8080)
                           Go Listener (background)
                           SQLite (file on disk)
```

First-time setup:

```bash
ssh root@VPS 'bash -s' < deploy/setup.sh   # provision server
# edit /opt/openplays/server/.env            # add secrets
# edit /opt/openplays/web/.env               # set ORIGIN
```

Manual deploy (or push to main for CI deploy):

```bash
./deploy/deploy.sh
```

## Migrations

Deploys automatically back up the DB before migrating. If a migration fails, restore from backup:

```bash
cp /opt/openplays/data/openplays.db.bak.TIMESTAMP /opt/openplays/data/openplays.db
sudo systemctl restart openplays-api openplays-listener
```

Prefer **additive migrations** (new columns as nullable, new tables, new indexes). For breaking changes, deploy in phases: new code that handles both schemas → migration → remove old schema support.
