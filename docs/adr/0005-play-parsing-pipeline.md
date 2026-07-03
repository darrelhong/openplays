# 0005 - Play parsing pipeline

- Status: Accepted
- Date: 2026-07-03

## Context

Plays are sourced from Telegram badminton groups where hosts post free-form
listings. The text is highly irregular: one message may contain many listings
(multiple dates, time slots, or level/price tiers), venue names are
abbreviated or misspelled, and the same listing is reposted with minor edits
as slots fill up. Group traffic (~250 messages/day) also includes chatter,
coaching ads, ticket resales, and equipment sales that are not play sessions.

Turning this into structured, queryable plays requires natural-language
extraction. We use a paid LLM behind an OpenAI-compatible API, which makes
every extraction call a real cost, so the architecture must control how often
the LLM is invoked, not just what it extracts.

Production data (May–July 2026, ~14,500 messages) shaped several decisions
recorded here: unbounded retries of deterministically-failing messages had
consumed ~half of all LLM calls ever made, and small output quirks (the LLM
quoting a number) were failing entire messages.

## Decision

Parsing is a staged pipeline: ingest dedup → durable job queue → LLM
extraction → tolerant parse → per-candidate step chain → play upsert.

### 1. Ingest and message-level dedup

The listener (a Telegram user session via gotgproto) receives every text
message in the configured supergroup. Before enqueueing, the message is
compared against the last 24 hours of stored messages using two tiers:
SHA256 exact hash, then trigram Jaccard similarity with a 0.85 threshold.
Matches are dropped without any LLM involvement. If the dedup check itself
fails (DB error), the message is enqueued anyway — a potential duplicate is
preferred over a lost message.

### 2. Durable job queue with bounded retries

`raw_messages` is the job queue, not just a log. Statuses: `pending`,
`processing`, `done`, `failed`, `skipped`, `dead`. A worker goroutine drains
pending jobs on a notify signal and re-checks failed jobs every 5 minutes.

- Failures back off on a schedule (30s, 1m, 2m, 5m, 15m).
- After `maxAttempts` total attempts (currently 3) a job is marked `dead`
  with its `last_error` preserved, and is never retried again.
- On startup, jobs stuck in `processing` (service killed mid-LLM-call, e.g.
  by a deploy) are requeued to `pending`.

### 3. One LLM call per message

Each job is sent to the LLM once per attempt: system prompt + sender name +
reference date + full message text, temperature 0. The system prompt encodes
the domain rules rather than code doing post-hoc fixes:

- Two listing types: `play` (per-person fee) and `sell_booking` (court
  let-go, total fee, no level/shuttle/max players).
- Explosion rule: each unique (date × venue × time slot × level) combination
  is a separate listing, so one message yields N candidates.
- Skip rules: coaching/training ads, "anywhere" venues, generic recurring
  schedules — the LLM returns an empty array for pure-noise messages.
- Field normalization: 24h times, dates resolved against the message
  timestamp, venue names stripped of area/detail text, level codes (LB…A),
  dollar-to-cents fees, gendered pricing rules, contact extraction.

There is no pre-LLM content filter: measured on prod data, only ~8% of
messages produce zero listings, and those are textually too similar to real
listings (prices, venues, sport keywords) for a safe keyword gate.

### 4. Tolerant response parsing

The response is parsed defensively before reaching typed code:

- Markdown code fences are stripped (LLMs sometimes wrap JSON despite
  instructions).
- Both a JSON array and a bare single object are accepted.
- Numeric fields (`courts`, `fee_cents`, `max_players`, `slots_left`,
  gendered fees) use `FlexFloat`/`FlexInt`, which accept a JSON number or a
  string with a numeric prefix (`2`, `"1200"`, `"3.5 courts"`). Non-numeric
  strings (`"$10"`) still fail: a dead message that can be inspected is
  better than silently storing a wrong price.

Parsed candidates are `ParsedPlayCandidate` — an ephemeral type owned by the
parser, converted to DB params before storage.

### 5. Per-candidate step chain

Each candidate independently runs convert → validate → resolve-venue →
upsert. A step returns `ErrSkip` (expected rejection, logged as a warning) or
an error; either way only that candidate is dropped — the other candidates
from the same message still persist, and the message is still marked `done`.

- **Convert**: candidate → `UpsertPlayParams` (UTC timestamps from local
  date/time + group timezone, level codes → ordinals, fractional courts
  floored with the original noted in `meta`, sport-specific attributes into
  the JSON `meta` field).
- **Validate**: sanity checks — duration must be positive and ≤ 5 hours.
- **Resolve venue**: raw venue text → canonical venue via the 5-step
  resolver (alias → expanded alias → fuzzy word overlap → geocoder →
  unresolved). Unresolved venues are kept, not dropped: the play stores the
  raw venue string and a NULL `venue_id`.
- **Upsert**: insert or update on the natural key
  `(host_name, starts_at, sport, COALESCE(venue_id, 0))`.

### 6. Play-level upsert as second dedup tier

Reposts older than the 24h message-dedup window, or edited beyond the
Jaccard threshold, still reach the LLM — but the upsert key collapses them
into the same play row, updating slots/fee/level instead of duplicating.
This is what makes "update existing games" work without any explicit
edit-detection logic.

## Rationale

### Why an LLM instead of rules/regex

The message formats are adversarially diverse (emoji-delimited dates, CJK
numerals, multi-tier pricing, gendered levels). The explosion rule alone —
7 listings from one message with overlapping date/tier groups — is beyond
maintainable regex. The prompt is the parser, and refining it is cheaper
than maintaining a rule engine.

### Why a queue between ingest and extraction

Telegram delivery is real-time but the LLM is slow (seconds) and fallible.
The queue decouples them: ingest never blocks on extraction, restarts don't
lose messages, and failures retry with backoff. The worker is a single
goroutine — at current volume (~250 messages/day) throughput is irrelevant;
durability is the point.

### Why retries are bounded

LLM extraction failures are mostly deterministic at temperature 0: the same
message produces the same unparseable output every time. Before the cap,
7 permanently-failing messages had accumulated 14,215 retry calls (~half of
all LLM calls ever made, ~73% of daily spend). The cap bounds the worst case
per message at `maxAttempts` calls; `dead` keeps the message and error
queryable (`SELECT id, last_error FROM raw_messages WHERE status='dead'`)
so new failure patterns can be root-caused. Dead messages still count for
ingest dedup, so a repost of a failing message does not restart the loop.

### Why parsing is tolerant, and where tolerance stops

Every parse failure now costs `maxAttempts` LLM calls and loses the listing,
so known-benign LLM quirks (markdown fences, single object instead of array,
quoted numbers) are absorbed in code. Tolerance stops where it could corrupt
data: non-numeric strings in numeric fields fail the message rather than
guessing. Coercion is hand-rolled (~30 lines) rather than a library —
jsoniter's fuzzy decoders are global and don't handle trailing text like
`"3.5 courts"`; cast/mapstructure would still need the same `UnmarshalJSON`
glue — consistent with the project's minimal-dependencies preference.

### Why dedup runs at two levels

Message-level dedup (cheap, pre-LLM) exists to avoid paying for extraction
of reposts; it is deliberately narrow (24h window, 0.85 similarity) to avoid
false positives. Play-level upsert (post-LLM) is the correctness backstop
that also turns legitimate reposts-with-edits into updates. Neither level
alone suffices: without the first, every repost costs an LLM call; without
the second, near-miss reposts create duplicate plays.

### Why a step chain with per-candidate isolation

One message can yield several candidates of mixed quality (e.g. a valid
listing plus a hallucinated 14-hour session). Isolating failures per
candidate means one bad extraction doesn't discard its siblings. Steps are
small interfaces (`Name()` + `Process()`) so tests use spies and the venue
resolver stays optional (nil resolver = skip resolution, used when no
geocoder is configured).

## Consequences

### Positive

- LLM calls track real message volume (~250/day): duplicates are filtered
  before the LLM, failures are bounded at 3 calls, and noise messages cost
  exactly one call each.
- Reposts and edits converge onto one play row via the upsert key.
- Every raw message is kept with its status, LLM response, and last error —
  the pipeline is fully replayable and debuggable from the DB.
- Prompt changes (new skip rules, new fields) don't require schema or code
  changes unless a new field is persisted.

### Trade-offs

- Extraction quality is bounded by the prompt; systematic misreads require
  prompt iteration and reprocessing.
- A provider outage longer than the retry window (~1.5 min of backoff at
  `maxAttempts = 3`) marks messages dead; recovery is a manual requeue
  (`UPDATE raw_messages SET status='pending' WHERE ...`).
- The 24h dedup window means a repost on day 2 costs an LLM call (absorbed
  by the upsert; accepted to keep slot-count updates flowing).
- The upsert key treats host+time+venue+sport as identity: a host genuinely
  running two different sessions at the same venue and time collapses into
  one play.
- `messages/day × attempts` is the cost ceiling; there is no batching of
  multiple messages per LLM call (rejected: one bad message would poison a
  batch, and per-message retry/attribution would blur).

## Implementation notes

- Ingest/dedup: `internal/listener/handler.go`, `internal/dedupe`.
- Queue/worker/retry cap: `internal/listener/worker.go`.
- Extraction and prompt: `internal/listener/pipeline/extractor.go`.
- Tolerant types: `internal/model/parsed_play_candidate.go`.
- Steps: `internal/listener/pipeline/step_*.go`, wired in `defaults.go`.
- Upsert key: `db/queries/plays.sql` (`ON CONFLICT`).
- Venue resolution details: see ADR 0003 and DECISIONS.md.
