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
