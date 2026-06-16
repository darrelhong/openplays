# 0003 - Venue search and Google place persistence

- Status: Accepted
- Date: 2026-06-16

## Context

User-created plays need venue entry that works for both known venues and new locations. The venue
field used to behave like free text, but richer play details, map links, location filtering, and
deduplication all benefit from linking plays to canonical venue rows.

The venue database will grow over time, so the create form should not preload every venue. It should
search local venues first and only use Google Places when local data is not enough. New Google places
should be stored in our database so later searches and plays can reuse the same venue row.

Singapore venue data has a practical wrinkle: postal codes can be building-level and are unique in
our `venues` table. A Google place can resolve to a postal code that already belongs to an existing
manual or OneMap venue row, so Google persistence must handle both `google_place_id` and
`postal_code` uniqueness collisions.

The UI should not expose whether a result came from Google or from our database as product copy.
Provider/source details are implementation details; the user should just see venue options.

## Decision

1. The create form uses an async venue combobox rather than preloading all venues.
2. Venue search is routed through a SvelteKit BFF endpoint for the create page, which calls the Go
   venue API using the typed API client and the current session cookie.
3. The Go venue search endpoint returns local database matches first.
4. Google Places autocomplete is called only when fewer than two local matches exist and the query is
   long enough for Google search.
5. Search results use one neutral shape:
   - saved venue results include `id`
   - unresolved suggestions include `google_place_id`
   - no public `source` field is returned
6. The frontend does not display a provider badge. It infers whether selection needs persistence
   from `id == nil && google_place_id != nil`.
7. Selecting a saved venue only fills the selected venue ID and display name.
8. Selecting an unresolved Google suggestion immediately resolves the Google place, upserts it into
   `venues`, and fills the returned saved venue ID.
9. Google autocomplete requests and place-details resolution share a generated session token. After a
   Google suggestion is resolved, the frontend starts a new token for the next search session.
10. The venue search combobox keeps a lightweight in-memory cache for the current browser/page
    session, keyed by normalized query plus Google session token.
11. Google venue upsert handles uniqueness conflicts by reusing/updating the conflicting venue row
    instead of failing when either `google_place_id` or `postal_code` already exists.
12. Existing venues can have `google_place_id` backfilled with a dry-run-first Go tool.

## Rationale

### Why local-first search

Local venue rows are cheaper, faster, and already canonical for our data. Querying them first keeps
Google usage low and helps repeated venue selection converge on database-backed results.

### Why top up with Google only when local matches are sparse

Some local matches are enough for common venues and aliases. Google is most useful when our own data
does not yet contain enough candidates. The threshold of two local matches keeps the UI useful while
limiting external autocomplete calls.

### Why save on suggestion selection

Persisting immediately after a user chooses a Google suggestion gives the create form a normal
`venue_id` before play submission. That keeps the play-create API simpler because it only needs to
create a play against an already saved venue.

This also lets the UI show the exact saved venue name/address after Google Place Details resolution
instead of relying only on autocomplete text.

### Why hide provider/source in the API response

The user does not need to know which provider produced a suggestion. The only behavior difference the
frontend needs is whether the option is already saved. `id` is enough to represent that: saved rows
have one, unresolved suggestions do not.

### Why keep `google_place_id` visible for unresolved suggestions

The frontend still needs an identifier to resolve the selected suggestion. For now we use
`google_place_id` directly. If we later want to avoid exposing provider-specific identifiers, we can
replace it with an opaque resolve token.

### Why use an in-memory page-session cache

Autocomplete can be noisy while users focus, blur, and retype. A tiny in-memory cache avoids repeated
requests for the same normalized query during the current page session without introducing
cross-page persistence, invalidation policies, or a query library dependency.

### Why resolve postal-code conflicts during Google upsert

Without this, a Google result whose postal code already exists on another venue row can fail with a
unique constraint error. Reusing the existing row is better than creating a duplicate venue or
returning a 500 from the create flow.

## Consequences

### Positive

- The create form scales better as the venue table grows.
- Google Places calls are reduced by local-first search and page-session caching.
- New venues become reusable after a user selects them.
- Play creation continues to submit a concrete `venue_id`.
- The UI presents one neutral venue list instead of provider-labeled results.
- Postal-code collisions no longer break Google venue persistence.

### Trade-offs

- Selecting a Google suggestion can create or update a venue even if the user later abandons the
  create form.
- The frontend still receives `google_place_id` for unresolved suggestions, so provider details are
  hidden from product copy but not fully abstracted from the API payload.
- The page-session cache is intentionally simple and does not persist across refreshes or route
  reloads.
- A Google session token rotates after resolution, so the same query may be fetched again in a later
  search session.

## Future option: save only on play creation

An alternative is to defer Google place resolution until the user submits the play-create form:

1. Search returns saved venues and unresolved suggestions.
2. Selecting an unresolved suggestion fills hidden `google_place_id` and session-token fields but does
   not save a venue yet.
3. The Go play-create endpoint resolves/upserts the venue and creates the play in one backend flow.

This would avoid storing venues from abandoned forms and tie the venue side effect to an actual play
creation. The trade-off is that play creation becomes responsible for Google resolution, venue
upsert, and error handling. If we revisit this, the save should happen in the Go create endpoint
rather than only in the SvelteKit action so venue persistence and play creation stay coupled on the
backend.

## Implementation notes

- `GET /api/venues/search` searches local venues first and optionally appends Google suggestions.
- `POST /api/venues/resolve-google` resolves a selected Google place and stores it as a venue.
- The create page uses `/create/venues` as a SvelteKit BFF route in front of the Go venue endpoints.
- `VenueSearchItem` does not include a public provider/source field.
- `google_place_id` is stored on `venues` and has a unique index.
- Google venue upsert handles both `google_place_id` and `postal_code` uniqueness conflicts.
- `server/tools/venueplaceids` backfills `google_place_id` for existing venues and dry-runs by
  default.
