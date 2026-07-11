# 0008 - Post-game player reviews (ratings, props, shoutouts)

- Status: Accepted
- Date: 2026-07-06
- Amends: [0002 - Roster capacity, waitlist privacy, and host management](0002-roster-capacity-waitlist-host-management.md)
- Related: [0007 - Require-waitlist join mode](0007-require-waitlist-join-mode.md)

## Context

The app had no post-game feedback loop: profiles showed only self-declared levels and game
counts, giving hosts nothing to screen joiners with and players no earned reputation. Sports and
social apps solve this with branded reaction vocabularies (Strava's kudos being the best-known),
so ours needed its own terminology and shape. Design constraints settled during product
discussion: reviews must be gameplay-scoped (only people who actually played together), star
ratings must not breed retaliation or grade inflation, skill praise should be sport-specific
rather than generic, and all reputation data is for logged-in users only.

## Decision

1. **Vocabulary.** A review of a co-player has three parts, each optional but never all empty:
   an anonymous 1–5 star **rating**, up to two **props** (predefined trait tags), and a
   **shoutout** (attributed free text shown on the reviewee's profile). "Props" and "shoutouts"
   are deliberately our own words rather than the badges/testimonials/kudos other apps use.
2. **Scope and eligibility.** One review per (play, reviewer, reviewee). Eligible reviewers and
   reviewees are the same set, defined by a single SQL query: active registered users holding a
   reserved spot (`confirmed`/`added`) plus the play hosts. Guests (no account) and pending
   queues are out; self-review is blocked (DB CHECK + handler).
3. **Roster freeze (amends 0002 globally).** Once `ends_at` passes, every roster mutation —
   join, leave, player confirm/decline, host add/waitlist/remove — returns 409 and the UI drops
   the CTAs. The final roster is the durable record of who played, which is what review
   eligibility reads; there is no attendance tracking, so status-at-end is the proxy.
4. **Window.** Reviews open at `ends_at` and close 14 days later (Go time comparison is
   authoritative; `DEV_REVIEWS_ALWAYS_OPEN=true` forces the window open for local testing).
   The PUT endpoint is an UPSERT and never deletes; the **UI locks a review after submit** for
   v1 — the edit plumbing is intentionally kept server-side to revisit (e.g. shoutout typos).
5. **Anonymity boundary.** Ratings leave the database only as aggregates: average, count, and a
   1–5 bucket distribution. Props surface as per-sport counts. Reviewer identity appears in
   exactly one place — shoutouts, which are attributed by design and carry no rating. All review
   data is served only through auth-required endpoints.
6. **Props are sport-linked; ratings and shoutouts are universal.** The vocabulary is a small
   universal attitude set (great sport, chill vibes, humble, punctual) plus a skill pack per
   sport, plus host-only props (well organized, quick replies, clear communication) offered only
   when the reviewee hosted that play. Every given prop counts toward the sport of the play it
   was earned in. Slugs are stored; display labels live in a frontend const map guarded by a
   unit test that parses the Go vocabulary, so the two cannot drift.
7. **Storage.** `play_reviews(play_id, reviewer, reviewee, rating, props, shoutout)` with a
   uniqueness key on the triple and CHECKs (rating 1–5 or null, shoutout non-blank when present,
   never all-empty, no self-review). Props are a JSON slug array rather than a join table:
   edits stay a single-row UPSERT, `json_each` covers the only aggregate needed, and the
   vocabulary is code-owned so there is nothing referential to gain.
8. **API.** `GET /plays/{id}/reviews` returns the viewer's review sheet (window state, prop
   vocabulary for the play's sport, co-players with the viewer's own review); 403 for
   non-participants. `PUT /plays/{id}/reviews/{revieweeUserID}` saves one review, window-gated.
   `PublicUserProfile` gains `rating` (average/count/distribution, omitted at zero) and
   sport-linked `props` counts; shoutouts live on their own cursor-paginated endpoint
   (`GET /users/{username}/shoutouts`, newest first), keeping reviewer identity out of the
   profile payload entirely.
9. **UI.** Reviews are written per player at `/play/{id}/review/{username}` (stars first; props
   and shoutout reveal after rating; submit returns to the game). The entry point is the ended
   play's roster: Confirmed badges give way to per-player "Give props" buttons, hidden for
   yourself, guests, and already-reviewed players. Ended games are reachable via a Past tab
   (`/my-games/past`, cancelled games included with their badge). Profiles show the star average
   beside the name (opening a distribution chart) and clickable per-sport cards (opening a sport
   summary with self rating, games, and prop counts).
10. **Prompt.** One notification kind, `play.review_prompt`, nudges every eligible participant
    after their game ends. A `play_review_prompts` marker row is inserted _before_ notifying
    (`INSERT OR IGNORE`), making the prompt at-most-once across rescans, restarts, and multiple
    instances. A prompter goroutine in the API process scans at one minute past five-minute
    clock marks (games end on whole hours, so a 5:00 finish is nudged at 5:01) with a 72-hour
    backstop that also keeps pre-feature plays from prompting; solo rosters are skipped.

## Rationale

### Why anonymous, aggregate-only stars

Peer-to-peer star ratings in a community where players meet again invite retaliation and 4.9
inflation (the Grab model works because riders rarely re-meet drivers). Full anonymity — no
per-rating attribution anywhere in any payload — removes the social cost of honest ratings,
while attributed shoutouts keep the warm, public side of reputation personal.

### Why props are sport-linked

"Powerful smash" praise is meaningful; "skilled player" is filler. Tying every prop to the sport
of the play it was earned in lets profiles read per-sport (matching the existing per-sport
ratings section), lets one slug (e.g. `fast_footwork`) be reused across sports without mixing
counts, and lets the review card offer only chips that make sense for that game. The two-prop
cap keeps them scarce enough to mean something.

### Why a per-player review page instead of a rate-everyone screen

The one-screen batch flow was built first and discarded: per-player pages match the actual
gesture (tap a person on the roster), keep each submit atomic with per-person validation, and
made the roster itself the natural entry point — which in turn let the ended-roster row swap its
status badge for the review CTA rather than growing a separate review hub.

### Why a marker table instead of a notified-at column

The prompt fans out to N users per play; a per-play column cannot represent partial failure
(crash after notifying 3 of 8 → duplicates or silent misses). A per-(play, user) primary key
with insert-before-notify gives exact at-most-once semantics: a crash between insert and notify
loses one nudge, which is strictly better than ever double-pushing.

### Why the roster freezes at game end

Review eligibility reads current roster status because `play_participants` keeps no history.
Freezing all roster mutations at `ends_at` makes that read stable by construction — the eligible
set cannot change after the review window opens — and is independently correct: editing who
"played" after the game misrepresents the record.

## Consequences

### Positive

- Profiles earn trust signals hosts can actually use, with zero payment/attendance modeling.
- The anonymity boundary is enforced structurally (aggregate-only queries), not by filtering.
- Adding sports or props is purely additive: new slugs, no migration, label-map test catches
  omissions.
- The prompt pipeline is idempotent enough to run on multiple instances unchanged.

### Trade-offs

- No editing after submit in the UI (v1): a shoutout typo is permanent until an edit affordance
  ships; the server-side UPSERT + window already support it.
- With one rating the average is fully identifying; the product accepted showing from the first
  rating rather than a minimum-count threshold.
- `user_blocks` are ignored: a blocked user's rating still counts and their shoutout still shows.
- Status-at-end approximates attendance; a no-show who never left the roster can review and be
  reviewed.
- A crashed prompt run can drop a nudge (at-most-once by choice); the review itself remains
  reachable from the Past tab and the play page.

### Future directions

Ideas from Strava-style social sports apps that build on this feature's data, roughly in order
of value per effort:

- **Double-blind shoutouts** (Airbnb-style): hold shoutouts hidden until both sides have
  reviewed or the window closes, removing reciprocity bias. The locked-after-submit model and
  the 14-day window already fit this — it is mostly a visibility rule on the profile query.
- **Most-played-with**: a profile section of frequent co-players with counts; a pure query over
  `play_participants` that compounds the social loop reviews started.
- **Milestone achievements**: system-granted flair distinct from peer props (10th game, first
  hosted game, first prop, all sports played) — fills profiles before peer reputation
  accumulates.
- **Year in review / monthly recap**: games, sports, top venue, props received, most-frequent
  partner — every ingredient already exists.
- **Host flair** (superhost-style): hosts with strong ratings and host props get a badge shown
  on their games at join time, closing the loop on the original goal of helping joiners and
  hosts trust each other.
