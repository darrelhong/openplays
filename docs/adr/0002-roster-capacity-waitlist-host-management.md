# 0002 - Roster capacity, waitlist privacy, and host management

- Status: Accepted
- Date: 2026-05-27
- Amended by: [0007 - Require-waitlist join mode](0007-require-waitlist-join-mode.md) (join
  placement on require-waitlist plays); [0008 - Post-game player reviews](0008-post-game-player-reviews.md)
  (all roster mutations freeze once the play ends)

## Context

User-created plays have a roster with confirmed and waitlisted participants. Imported plays can still
show player and slot details from the source message, but they do not have OpenPlays host records and
cannot be managed through roster controls.

Host permissions are stored separately from roster membership. The creator is the initial host, but
the host record is intentionally not the same thing as a participant row. This keeps room for future
organizers who can manage a play without taking one of the player slots.

User-created plays can be cancelled by a host. Cancellation is a terminal state for normal roster
actions, but it is not a hard delete because existing participants and hosts still need an auditable
record of what happened.

Roster behavior needs to be predictable for players and hosts:

- Players need to know whether they are confirmed or waitlisted.
- Hosts need deliberate controls for accepting or removing people.
- Capacity must not be bypassed by waitlist acceptance.
- Waitlist identities are more sensitive than aggregate waitlist demand.
- Hosts and future organizers need management permissions without necessarily being players.
- Cancelled plays should disappear from discovery lists while remaining available by direct URL.
- Players need a clear distinction between being automatically confirmed and being invited from the
  waitlist to confirm their spot.

## Decision

1. `max_players` is the roster capacity. Today it caps confirmed participants; once `added` exists,
   it caps confirmed plus added participants because added rows reserve a spot.
2. `slots_left` is never negative. Today it is derived from `max_players - confirmed_count`; once
   `added` exists, it should be derived from `max_players - confirmed_count - added_count`.
3. Joining a user-created play auto-confirms only when the user has a matching level and an open
   slot. Otherwise the user is waitlisted.
4. Hosts may accept a waitlisted participant only when an open slot already exists.
5. Host acceptance intentionally overrides level mismatch checks, but never overrides capacity.
6. Removing a confirmed participant frees a slot but does not automatically promote anyone from the
   waitlist.
7. Removing a waitlisted participant only removes that waitlist row.
8. Hosts may remove confirmed or waitlisted participants, but may not remove a participant row that
   belongs to any host of the play.
9. User-created play management is authorized through `play_hosts`, not through roster membership.
10. The play creator is inserted as the initial host.
11. Additional organizer-style permissions should be represented as additional host records until
    the product has a separate organizer concept.
12. Imported plays cannot be joined or managed through roster controls.
13. Confirmed participant previews may be shown on public play details and cards.
14. Waitlist identities are visible only to the host. Other viewers may see waitlist count and their
    own viewer state.
15. Cancelling a user-created play sets `cancelled_at` and `cancelled_by`; it does not delete the
    play, participants, or host records.
16. Cancelled plays are excluded from list and discovery queries, but direct detail URLs may still
    return the play with cancelled state.
17. Cancelled plays cannot be joined, left, edited, accepted from waitlist, or have roster rows
    removed through normal management actions.
18. Direct joins from players who satisfy level and capacity rules are automatically confirmed.
19. Direct joins that are waitlisted due to level restriction or capacity remain waitlisted.
20. When a host accepts a waitlisted player in the future confirmation flow, the participant should
    move to an `added` state rather than immediately becoming confirmed.
21. The `added` state represents a host-held spot that requires the player to confirm before they
    become a confirmed participant.
22. `added` rows reserve capacity so hosts cannot offer the same open spot to multiple waitlisted
    players at once.
23. Players in the `added` state can confirm into `confirmed` or leave/remove themselves from the
    roster.

## Rationale

### Why require an open slot for host acceptance

Accepting from the waitlist should not silently expand the game. If a host wants more players, that
belongs in a future player-count editing flow. Until that exists, accepting a waitlisted player is a
move into existing capacity.

### Why avoid automatic promotion

Automatic promotion can surprise both hosts and players. Hosts may want to choose who gets the open
slot, especially when level, attendance history, or payment status matters outside the app. Keeping
promotion manual makes host intent explicit.

### Why allow hosts to override level checks

Level matching is a useful default for self-service joining, not a hard policy. A host accepting a
waitlisted player is an explicit judgment call and should be allowed as long as capacity remains
valid.

### Why separate hosts from participants

Hosts are a permission boundary, not necessarily roster members. Keeping `play_hosts` separate from
`play_participants` lets the creator manage the game today and leaves space for future organizers who
help run the game without playing in it.

### Why keep waitlist identities private

Waitlist demand is useful public information, but names and photos of people who did not make the
confirmed roster are less necessary for non-hosts. The host needs that information to manage the
roster; other viewers do not.

### Why cancellation is soft

Cancellation should stop future actions and remove the play from discovery without erasing the
history participants may need to inspect. Keeping the play, participant rows, and host rows also lets
`cancelled_by` identify which host performed the cancellation.

### Why use an added state before confirmation

Host acceptance is different from a player choosing to join directly. A waitlisted player may no
longer be available by the time the host offers them a slot, so the player should explicitly confirm
before they become part of the final roster. Reserving capacity while in `added` prevents hosts from
accidentally offering one slot to multiple people.

## Consequences

### Positive

- Capacity rules are consistent across joining, accepting, leaving, and removal.
- Hosts stay in control of waitlist movement.
- Waitlist privacy is preserved while still exposing useful aggregate demand.
- Host permissions can grow without changing roster semantics.
- Cancelled games remain auditable without appearing in normal discovery.
- Future player-count editing can compose with the same capacity rule.

### Trade-offs

- A freed slot remains open until the host accepts someone or another player joins.
- Waitlisted players may not move strictly first-in-first-out because host acceptance is explicit.
- `added` rows can hold capacity while waiting for player confirmation, so a future expiry or host
  release action may be needed.
- Hosts cannot remove themselves from the roster until a cancel, remove-host, or transfer-host flow
  exists.

## Implementation notes

- `play_participants.status` stores `confirmed` or `waitlisted`.
- The planned confirmation flow will extend `play_participants.status` with `added`.
- Capacity checks for accepting waitlisted participants should count both `confirmed` and `added`
  rows once `added` exists.
- The backend currently recomputes `slots_left` from confirmed participant count after roster
  mutations.
- Once `added` exists, public capacity should distinguish confirmed player count from available
  slots so reserved-but-unconfirmed spots do not look available.
- The play detail endpoint returns waitlist rows only when `can_manage` is true.
- Host roster management uses participant IDs, not user IDs, so it can support registered and guest
  rows if guest management is expanded later.
- `play_hosts` stores users who may manage a user-created play. The creator is currently inserted as
  the initial host, but host membership does not imply participant membership.
- `plays.cancelled_at` records the cancellation time and `plays.cancelled_by` records the host user
  that cancelled the play.
- The web join flow should use browser feedback to tell the player whether they were confirmed,
  waitlisted, or added pending confirmation.
