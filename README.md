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

## Structure

```
server/     Go API + migrations + tools
web/        SvelteKit frontend
```
