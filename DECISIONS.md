## Server

### General
- Go backend uses sqlc for type-safe SQL, goose for migrations, huma + chi for API with auto-generated OpenAPI spec
- Minimal dependencies — lightweight
- File-based routing pattern for Go API: internal/api/routes/api/{resource}/ with router.go at each level
- Use sqlc.narg pattern for optional SQL filters in SQLite

### Time & Timestamps
- RFC3339 for all API-facing timestamps; SQLite internal format (YYYY-MM-DD HH:MM:SS+00:00) only at DB boundary
- All times stored as UTC in the DB; timezone stored separately so local display time can be reconstructed
- SQLite timezone gotcha: +08:00 suffix breaks string comparison against CURRENT_TIMESTAMP — all stored times must be UTC with Z suffix

### Ingestion & Parsing
- Create new games based on ingested messages, update existing games if they already exist
- Parse messages using LLM to extract game details
- LLM output handling: accepts both array and single-object JSON responses, strips markdown fences LLMs sometimes wrap around output
- raw_messages table serves as a job queue (not just a log) — has status, retry_count, next_retry_at, last_error for async processing
- Go workers orchestrate async message processing; on startup, drain leftover pending/failed jobs from previous runs
- Retry backoff schedule: 30s, 1m, 2m, 5m, 15m (capped)
- Resilience preference: if dedup check fails (DB error), prefer potential duplicate over dropping a message

### Deduplication
- Two-tier dedup: SHA256 exact-match hash + trigram Jaccard similarity for fuzzy matching
- Jaccard threshold at 0.85 — empirically tuned against real Telegram badminton group messages; catches reposts with minor edits (slot count changes, session removals) while keeping different-host messages with similar structure as separate

### Venue Resolution
- 5-step resolution pipeline: exact alias lookup → expanded alias lookup → fuzzy word overlap (60% threshold) → geocoder fallback → unresolved
- Each abbreviation expands to exactly one word to prevent composability bugs (e.g. "sec sch" → "secondary school")
- Resolved raw strings are saved as aliases so future lookups skip expansion
- Initialisms (SBH, TPCC, BV CC) and nicknames cannot be resolved by expansion or fuzzy matching — these need manual aliases via the venuefill tool
- Geocoder is optional — system degrades gracefully to alias/fuzzy matching only
- Two geocoder providers: Google Places (5,000 free requests/month, requires API key) or OneMap (Singapore government API, free, requires email/password)
- Venues with postal codes upsert on postal_code; generic locations (e.g. "Simei") without postal codes always insert new rows

### Pagination
- Cursor-based pagination (forward-only, composite keyset). Default sort uses (starts_at, id), distance sort uses (distance_km, id)
- Fetch page_size + 1 rows; extra row determines has_more flag without needing a separate query (a separate COUNT query is still issued for the total)
- Separate SQL queries per sort mode (time, distance) since sqlc is static — can't dynamically change JOIN type or ORDER BY
- Distance-sorted queries use INNER JOIN venues (excludes plays without a resolved venue); time-sorted uses LEFT JOIN

### Data Model
- Level ordinals use gaps of 10 for future insertability (LB=10, MB=20, etc.); tennis uses NTRP rating x10 directly (e.g. 35 for 3.5)
- Meta field is a flexible JSON object stored as TEXT for sport-specific attributes: shuttle brand, air-con, gendered pricing/levels, pitch type, etc. — avoids schema changes per sport
- Contacts stored as JSON array in TEXT column, implements sql Scanner/Valuer for transparent serialization
- ParsedPlayCandidate is an ephemeral type (LLM parser output) — converted to db.Play or db.UpsertPlayParams before storage; two conversion paths exist due to different nullability semantics

### Testing
- Integration tests use in-memory SQLite with real goose migrations (not mocks) — tests run against the real schema
- Spy test doubles record all calls for behaviour verification rather than implementation detail testing

## Web

### General
- SvelteKit + Bits UI + UnoCSS
- Prioritise server-side rendering first and try not to expose APIs directly
- Stone color palette throughout dark-themed frontend (bg-stone-950, text-stone-100, etc.)
- pnpm gen:types in web/ to regenerate TS types from running Go API's /openapi.json

### Components & Styling
- Use CSS transitions with data-[state] and data-[starting-style] attributes for animations instead of tailwindcss-animate
- Components should follow Bits UI recommended patterns (composable primitives with a batteries-included wrapper) shadcn style
- Reusable helpers should live in $lib/utils/ and be tested with vitest

### Display Conventions
- Time display: omit minutes when :00 (e.g. "7 pm" vs "7:30 pm")
- Fee display: whole dollar amounts omit decimals ($10 vs $10.50)
- Find nearby activities using existing venues and their coordinates so as to not require separate call to expensive geo APIs
