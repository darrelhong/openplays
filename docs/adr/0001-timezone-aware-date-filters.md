# 0001 - Timezone-aware date filters

- Status: Accepted
- Date: 2026-05-08

## Context

The system stores play timestamps in UTC in SQLite and returns API timestamps in RFC3339.
That keeps storage and transport consistent, but date-only filters such as `starts_after=2026-04-10`
are ambiguous without a timezone.

This ambiguity matters because users think in local calendar days, not UTC boundaries.
For example, a user in Singapore filtering for `2026-04-10` expects plays on April 10 in
Singapore time, not plays after `2026-04-10T00:00:00Z`.

We also needed a predictable rule for upper bounds. A `starts_before` date should behave like
an inclusive day picker in the UI, but SQL range filtering is simpler and safer with an exclusive
upper bound.

## Decision

We standardize date filtering as follows:

1. The API accepts an optional `timezone` query parameter using an IANA timezone name.
   Example: `Asia/Singapore`.
2. If `timezone` is omitted or invalid, the backend falls back to `UTC`.
3. `starts_after` is interpreted as the start of the provided calendar day in the requested
   timezone, then converted to UTC for database filtering.
4. `starts_before` is interpreted as an inclusive calendar day in the UI, but implemented as the
   start of the next calendar day in the requested timezone, converted to UTC, and used as an
   exclusive SQL upper bound.
5. The frontend should preserve and send the browser timezone when it mutates date-related filters
   so the backend can interpret date-only query parameters using the user's local calendar.

## Rationale

### Why IANA timezone names

IANA timezone names are what Go's `time.LoadLocation` expects and what browsers return from
`Intl.DateTimeFormat().resolvedOptions().timeZone`.

This gives us one shared format across frontend and backend without maintaining our own mapping.

### Why default to UTC

Defaulting to UTC preserves existing API behavior for callers that do not yet send `timezone`.
It also gives a deterministic fallback when a timezone is missing or invalid.

### Why use exclusive `starts_before`

Exclusive upper bounds avoid off-by-one and end-of-day precision bugs. Internally, the SQL rule is:

`starts_at >= starts_after_utc AND starts_at < starts_before_utc`

That is easier to reason about than trying to synthesize `23:59:59.999...` boundaries.

## Consequences

### Positive

- Users get date filtering based on their local calendar day.
- Frontend and backend use the same timezone format.
- SQL filtering remains simple and index-friendly.
- Existing clients continue to work because UTC remains the fallback.

### Trade-offs

- Date-only filters now depend on the caller supplying a meaningful timezone for local-day behavior.
- Different callers may see different results for the same date string if they send different
  timezones.
- Invalid timezone values do not fail the request; they silently fall back to UTC.

## Implementation notes

- Database timestamps remain stored in UTC.
- API timestamps remain RFC3339.
- SQLite query parameters use UTC datetime strings at the DB boundary.
- Frontend filter interactions preserve a `timezone` query param derived from the browser timezone.
