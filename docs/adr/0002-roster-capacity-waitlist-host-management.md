# 0002 - Roster capacity, waitlist privacy, and host management

- Status: Accepted
- Date: 2026-05-27

## Context

User-created plays have a roster with confirmed and waitlisted participants. Imported plays can still
show player and slot details from the source message, but they do not have an OpenPlays host who can
manage a roster.

Roster behavior needs to be predictable for players and hosts:

- Players need to know whether they are confirmed or waitlisted.
- Hosts need deliberate controls for accepting or removing people.
- Capacity must not be bypassed by waitlist acceptance.
- Waitlist identities are more sensitive than aggregate waitlist demand.

## Decision

1. `max_players` is the capacity for confirmed participants only.
2. `slots_left` is derived from `max_players - confirmed_count` and is never negative.
3. Joining a user-created play auto-confirms only when the user has a matching level and an open
   slot. Otherwise the user is waitlisted.
4. Hosts may accept a waitlisted participant only when an open slot already exists.
5. Host acceptance intentionally overrides level mismatch checks, but never overrides capacity.
6. Removing a confirmed participant frees a slot but does not automatically promote anyone from the
   waitlist.
7. Removing a waitlisted participant only removes that waitlist row.
8. Hosts may remove confirmed or waitlisted participants, but may not remove their own creator
   participant row.
9. Only the creator of a user-created play can manage its roster.
10. Imported plays cannot be joined or managed through roster controls.
11. Confirmed participant previews may be shown on public play details and cards.
12. Waitlist identities are visible only to the host. Other viewers may see waitlist count and their
    own viewer state.

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

### Why keep waitlist identities private

Waitlist demand is useful public information, but names and photos of people who did not make the
confirmed roster are less necessary for non-hosts. The host needs that information to manage the
roster; other viewers do not.

## Consequences

### Positive

- Capacity rules are consistent across joining, accepting, leaving, and removal.
- Hosts stay in control of waitlist movement.
- Waitlist privacy is preserved while still exposing useful aggregate demand.
- Future player-count editing can compose with the same capacity rule.

### Trade-offs

- A freed slot remains open until the host accepts someone or another player joins.
- Waitlisted players may not move strictly first-in-first-out because host acceptance is explicit.
- Hosts cannot remove themselves from the roster until a cancel, delete, or transfer-host flow
  exists.

## Implementation notes

- `play_participants.status` stores `confirmed` or `waitlisted`.
- The backend recomputes `slots_left` from confirmed participant count after roster mutations.
- The play detail endpoint returns waitlist rows only when `can_manage` is true.
- Host roster management uses participant IDs, not user IDs, so it can support registered and guest
  rows if guest management is expanded later.
