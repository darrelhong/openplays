# 0007 - Require-waitlist join mode (host-reviewed requests)

- Status: Accepted
- Date: 2026-07-03
- Amends: [0002 - Roster capacity, waitlist privacy, and host management](0002-roster-capacity-waitlist-host-management.md)

## Context

The join flow from ADR 0002 (auto-confirm on level match + open slot, waitlist otherwise) suits
tennis-style games. Badminton games commonly require pre-payment: the host wants to review every
joiner and grant spots only after payment is received. The app cannot observe payments, so the
feature must be payment-agnostic — copy speaks of hosts confirming players, never of payment.
Alternatives considered and rejected: a Telegram-style "host approval" mode that skipped the
player's confirm-participation step (loses the existing double-opt-in), and a single relabeled
pending queue (conflated "awaiting host review" with "waiting for a slot").

## Decision

1. Add a per-play boolean `require_waitlist` (create/edit checkbox "Require waitlist"), default
   off. Classic (off) plays keep ADR 0002's flow with two adjustments:
   - **Joins never confirm instantly.** A direct join (level match + open slot) reserves the spot
     as `added` and the player still confirms participation — `confirmed` always means the player
     explicitly confirmed, regardless of entry path (event `participant.joined`).
   - **The pending queue is presented as requests.** A classic play's `waitlisted` rows read as
     "Request to join" / "Requested" / a "Requests" section, with "{name} requested to join"
     notifications/history and withdrawal events — it functionally is a request queue (join → host
     adds → player confirms). The "waitlist" label is reserved for the host-parked list that only
     exists on require-waitlist plays.
2. On require-waitlist plays, joining never auto-confirms: every join creates a participant with
   the new status `requested`, regardless of open slots or level match. The join CTA reads
   "Request to join". Rating is still recorded so hosts see the requester's level.
3. `requested` is a fourth participant status alongside `confirmed`/`added`/`waitlisted`. It does
   not reserve capacity. No participants-table migration is needed (status has no CHECK
   constraint); ordering and counts treat it as its own bucket.
4. Hosts see a "Requests" section with two placement CTAs per request plus Remove:
   - **Add** — the existing accept endpoint, extended to accept `requested` as well as
     `waitlisted`; still requires an open slot; still transitions to `added`, after which the
     player confirms participation as in ADR 0002.
   - **Waitlist** — a new endpoint (`POST /{id}/participants/{participantID}/waitlist`) parking a
     `requested` participant as `waitlisted` (409 for any other status). Capacity unchanged.
5. Requesters can withdraw via the existing leave endpoint ("Withdraw request").
6. Privacy mirrors the waitlist: request identities are host-only; a requester sees their own row
   and the aggregate `requested_count`. `viewer_state` gains a `requested` value.
7. Notifications: every pending join — classic or require-waitlist — fires one kind,
   `play.join_requested` ("{name} requested to join"), to hosts; `play.moved_to_waitlist`
   ("You were added to the waitlist") goes to the player when parked; both feed+push. Direct
   joins fire `play.player_joined`; accept keeps `play.player_added`. Withdrawals are silent.
   The legacy `play.waitlist_joined` kind is no longer emitted.
8. Play activity events are stamped unambiguously at write time so each type has exactly one
   meaning and one copy forever: `participant.joined` (direct join, added pending
   self-confirmation), `participant.join_requested` (any pending join, both modes),
   `participant.moved_to_waitlist` (host-redacted like `participant.added`), and
   `participant.request_withdrawn`. The legacy `participant.joined_confirmed` and
   `participant.joined_waitlist` types are no longer written. The participant-visible activity
   feed carries roster events only; pending-queue events (requests, waitlist moves, withdrawals)
   are host-only, matching the queue's identity privacy.
9. Toggling the flag on edit needs no data migration: existing rows keep their statuses; only new
   joins change behavior. Requests created while the flag was on remain manageable after toggling
   it off (accept/waitlist/remove still work on `requested` rows).

## Rationale

### Why a per-play option instead of a universal request flow

Replacing auto-join everywhere would add a host round-trip to every tennis game that today joins
in one tap. The option scopes the friction to games that need vetting; the "Require waitlist" name
tells hosts what they're opting into, and joiners learn the flow implicitly from the "Request to
join" CTA and their "Requested" state.

### Why requests are a distinct status rather than a relabeled waitlist

The two pending meanings differ: a request is "the host hasn't decided", the waitlist is "the host
decided to keep them in line". Keeping them separate gives hosts a triage inbox (requests) and a
holding list (waitlist), lets the host communicate a decision without granting a spot, and avoids
presentation-overloading `waitlisted` based on a play flag. Since `play_participants.status` has
no DB constraint, the new value costs no schema change.

### Why host placement reuses accept → added → player confirms

The player's confirm-participation step is deliberate double-opt-in (ADR 0002 #20-23): plans
change between requesting and being granted a spot. Reusing the accept machinery means zero new
state transitions for granting spots, the capacity rule ("accept requires an open slot, overrides
level but never capacity") applies unchanged, and an accepted player gains `added` status —
which already carries game-chat access for coordinating payment details.

### Why requests never reserve capacity

Reserving on request would let a slow host's inbox block confirmed players, and would make "full"
ambiguous. Requests are intent, not occupancy — identical to the waitlist's existing semantics, so
`slots_left` accounting (`reserved = confirmed + added`) is untouched.

## Consequences

### Positive

- Payment-gated (or any vetted) games work without the app modeling payments.
- Classic games are byte-for-byte unaffected; the flag only changes join placement and adds one
  endpoint.
- Capacity, privacy, no-auto-promotion, and double-opt-in rules from ADR 0002 all carry over.
- The host's request inbox is explicit; the activity feed distinguishes requests from waitlist
  joins.

### Trade-offs

- Hosts of require-waitlist games must act on every joiner; an unattended game accumulates
  requests (no expiry yet, mirroring the existing `added`-row trade-off in 0002).
- Requesters cannot access the play chat until accepted (`added`), so payment instructions must
  live in the play description/contacts.
- A requester has no visibility into queue position or how many others requested beyond the count.
- Level filtering is advisory-only in this mode: everyone can request; the host sees the rating
  and decides.

### ADR 0002 clauses amended (require-waitlist plays only)

- #3/#18/#19 (join auto-confirms on level+slot): joins always become `requested`.
- #13/#14: `added` participants are now publicly visible like confirmed players (they hold
  reserved spots), with an `is_viewer` flag scoping self-serve actions; waitlist privacy extends
  to requests.
- Reaffirmed: #4/#5 (accept requires slot, overrides level, never capacity), #6 (no
  auto-promotion), #20-23 (`added` semantics and player confirmation).

## Implementation notes

- Migration `20260703010000_play_require_waitlist.sql`: `plays.require_waitlist BOOLEAN NOT NULL
  DEFAULT 0`.
- Go status enum: `model.ParticipantRequested`; event types in `model/play.go`; join placement in
  `resolveJoinStatus` (join.go); accept/waitlist endpoints in `manage_roster.go`; leave event in
  `events.go`; notifications in `notifications/play.go` + `policy.go`.
- Payload: `PlayPublic.require_waitlist`, `requests[]`, `requested_count`, `viewer_state` value
  `requested` (get.go; list paths pass raw participant status through).
- Web: `getPlayJoinLabel`/`canDirectJoin` short-circuit on the flag; checkbox in create and edit
  forms (both, they are not shared); requests section, badges, and dialogs in
  `play-details-content.svelte`; `?/waitlistParticipant` action in `play/[id]/+page.server.ts`;
  "Requested" chip in `play-viewer-state-badge.svelte`.
- Tests: `require_waitlist_test.go` (join/withdraw/accept/park/full-roster/classic-accept), web
  `play-join-label.test.ts`.
