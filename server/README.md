# OpenPlays Server

Listens to Telegram group messages, extracts structured sports session listings using a local LLM, and outputs parsed play data.

## Prerequisites

- Go 1.26+
- [LM Studio](https://lmstudio.ai/) (or any OpenAI-compatible API endpoint)
- Telegram API credentials from [my.telegram.org](https://my.telegram.org)

## Setup

```bash
cp .env.example .env
```

Fill in `.env`:

`LLM_MODEL` can be left empty to use whatever model is loaded in LM Studio. Set `LLM_API_KEY` if using a cloud provider like OpenAI.

## Listener

Connects to Telegram and parses incoming messages in real-time.

### Deduplication

Before a message is queued for LLM parsing, it is checked against the last 24 hours of messages using trigram Jaccard similarity. Messages with >= 85% similarity to an existing message are skipped and never inserted into `raw_messages`. This catches reposts with minor edits (e.g. slot count changes, emoji tweaks) without blocking genuinely distinct listings.

Text is normalized before comparison: lowercased, emoji stripped, formatting characters removed, whitespace collapsed. A SHA-256 hash of the normalized text is also stored for potential exact-match lookups. If the dedup query fails, the message is still inserted (fail-open — better a duplicate than a lost message).

```bash
go run ./cmd/listener/
```

Log to file while watching output:

```bash
go run ./cmd/listener/ 2>&1 | tee -a listener_output.log
```

On first run, Telegram will prompt for a verification code.

### Worker

The listener runs a background worker that processes messages asynchronously:

1. **New messages** are inserted into `raw_messages` with status `pending` and processed immediately via `Notify()`.
2. **Failed jobs** (e.g. LLM rate limits, timeouts) are retried with exponential backoff (30s, 1m, 2m, 5m, 15m cap). A ticker checks for due retries every 5 minutes.
3. **Pending and retry queues are separate** — new messages are never blocked by retries, and retries happen on schedule even if no new messages arrive.
4. On startup, both queues are drained before entering the main loop.

Rate limiting is configured via `LLM_MAX_REQ_PER_MIN` in `.env`. The limiter lives in the LLM extractor, so all callers (worker, parsetest) are throttled.

Parsed plays are upserted on `(host_name, starts_at, ends_at, sport, venue_postal_code)` — if the same host reposts a session with updated info (e.g. fewer slots, price change), the existing play is updated rather than duplicated.

### Venue Normalization

After LLM extraction, the worker resolves each venue to a canonical entry in the `venues` table using Singapore postal codes. This enables location-based filtering and prevents duplicates caused by venue name variations (e.g. "Hougang CC" vs "Hougang Community Club").

Resolution flow:

1. **Exact alias match** — lowercase the raw venue string and look up `venue_aliases`. If found, use the cached postal code.
2. **Abbreviation expansion** — expand common abbreviations (`sec` → `secondary`, `cc` → `community club`, `sh` → `sport hall`, etc.) and try alias lookup again. `"sports hall"` is normalised to `"sport hall"` everywhere.
3. **Fuzzy matching** — compare the expanded input against all venue names in the database using word overlap scoring. If ≥60% of the input words appear in a candidate venue name, it's a match. Catches cases like `"Canberra Sport Hall"` matching `"Bukit Canberra Sports Hall"`.
4. **Geocoder fallback** — query Google Places (or OneMap) with the raw venue name. If a result is returned, upsert into `venues` and store the raw string as a new alias.
5. **Unresolved** — if all steps fail, the play is inserted with `venue_postal_code = NULL` for manual resolution later.

Every successful resolution (via any step) automatically saves the raw input as an alias, so the same string resolves instantly next time. The alias table grows organically from real messages.

Initialisms and nicknames (SBH, TPCC, BV CC) can't be resolved by expansion or fuzzy matching — these need manual aliases via `venuefill`.

Venue resolution is optional — if no geocoder credentials are set in `.env`, it still resolves via aliases and fuzzy matching against existing venues.

## Test parsing

Pipe a message through the LLM pipeline without needing Telegram. Useful for testing different LLM providers or prompt changes.

```bash

# From file with sender name
SENDER_NAME="Daniel" go run ./tools/parsetest/ < example_messages.txt

## With a different LLM provider
LLM_BASE_URL=https://api.openai.com/v1 \
LLM_MODEL=gpt-4o-mini \
LLM_API_KEY=sk-... \
go run ./tools/parsetest/ < example_messages.txt
```

## Test OneMap search

Query the OneMap API directly to verify venue resolution. Requires `ONEMAP_EMAIL` and `ONEMAP_PASSWORD` in `.env`.

```bash
go run ./tools/onemaptest/ "Hougang CC"
go run ./tools/onemaptest/ "Singapore Badminton Hall"
```

## Manage venues

Populate the `venues` and `venue_aliases` tables using the `venuefill` tool. Supports both Google Places (default) and OneMap (`--onemap`) as geocoding providers.

```bash
# Search Google Places and create a venue + aliases
go run ./tools/venuefill/ search "Hougang Community Club" "hougang cc" "hg cc"

# Search OneMap instead
go run ./tools/venuefill/ search --onemap "Hougang Community Club" "hougang cc"

# Add aliases to an existing venue by ID (no API call)
go run ./tools/venuefill/ alias 1 "hougang community club" "hougang cc"

# List all venues and aliases
go run ./tools/venuefill/ list
```

The `search` subcommand queries the geocoding provider, upserts the venue, and adds the lowercased search term plus any extra arguments as aliases. The `alias` subcommand maps aliases to a venue that already exists in the database by its ID — useful for abbreviations and colloquial names that neither API can resolve (e.g. "sbh" for Singapore Badminton Hall). Run `list` to see venue IDs.

Aliases are upserted — re-running `alias` with the same alias pointing to a different venue ID will override the existing mapping. This makes it easy to fix wrong venue resolutions:

```bash
# Google resolved "zhenghua cc" to the wrong place — fix it via OneMap:
go run ./tools/venuefill/ search --onemap "Zhenghua CC"
# Note the venue ID from the output, then re-point the alias:
go run ./tools/venuefill/ alias <venue_id> "zhenghua cc"
```

## Models & Database

Database models are generated by [sqlc](https://sqlc.dev/) from the SQL schema. Migrations are managed by [goose](https://github.com/pressly/goose).

```bash
# Install tools
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

### Run migrations

```bash
goose -dir db/migrations sqlite3 openplays.db up
```

### Regenerate models after schema or query changes

```bash
sqlc generate
```

This reads `db/migrations/*.sql` (schema) and `db/queries/*.sql` (queries), then generates Go code in `internal/db/`. The generated `db.Play` struct uses custom types from `internal/model/` via column overrides in `sqlc.yaml` — enums (`Sport`, `ListingType`, `GameType`, `GenderPref`) and JSON types (`Contacts`, `Meta`) map directly, no manual conversion needed.

**Do not edit files in `internal/db/`** — they are overwritten on every `sqlc generate`.
