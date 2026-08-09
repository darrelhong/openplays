# 0011 - Host-invited players

- Status: Accepted
- Date: 2026-08-08
- Amends: [0002 - Roster capacity, waitlist privacy, and host management](0002-roster-capacity-waitlist-host-management.md); [0007 - Require-waitlist join mode (host-reviewed requests)](0007-require-waitlist-join-mode.md)

## Context

The roster currently supports self-service joins and host decisions about participant rows, but it
has no inbound edge for a host to ask another registered user to join a play. Creating a separate
invite table would duplicate the participant lifecycle and create a second source of truth for
pending roster relationships.

The existing `requested`, `waitlisted`, and `added` statuses already distinguish pending intent,
host parking, and a host-held confirmed spot. An invitation has a different meaning: the host has
selected a person who has not yet accepted the invitation. It therefore needs its own participant
status and privacy rules while continuing to use the existing roster and event machinery.

## Decision

1. Add `invited` as the fifth `play_participants.status` value. Do not add a `play_invites` table.
   Store the inviting host in nullable `invited_by_user_id` on the participant row so
   participant-initiated invitations can be added later without another migration.

2. An `invited` row never reserves capacity. Reserved capacity remains the count of participants
   whose status is `confirmed` or `added`; `requested`, `waitlisted`, and `invited` do not affect
   `slots_left`.

3. An invited user has no play-chat access. Chat membership continues to be limited to
   `confirmed` and `added` participants.

4. Only a play host may send an invitation. Authorization uses the existing `requirePlayHost`
   permission boundary. Participant-initiated invitations, guest invitations, invitations to
   users without accounts, tokenized or revocable share links, and email delivery are out of
   scope.

5. A host invitation is accepted into one of three placements:

   - On a `require_waitlist` play, it becomes `requested`.
   - Otherwise, if reserved capacity is at least `max_players`, it becomes `waitlisted`.
   - Otherwise, it becomes `confirmed`.

   The acceptance decision does not perform a level check. The host deliberately selected the
   invitee, so the host's judgment overrides the self-service level filter just as host acceptance
   does under ADR 0002; capacity is never overridden. On a require-waitlist play, capacity is
   irrelevant because `requested` does not reserve a spot.

   Accepting an invitation goes directly to `confirmed`, rather than through `added`. The host's
   invitation is the host opt-in and the player's acceptance is the player's explicit opt-in, so
   the double-opt-in requirement is preserved. This intentionally narrows ADR 0007's general
   `added`-before-confirmation flow for this invitation path.

6. Invitation identities are host-only. The public payload includes `invited_count`, but not the
   names or previews of invited users. An invitee may see their own invited row and
   `viewer_state: "invited"`. This reuses the host-only pending-queue privacy rule from ADR 0002
   and ADR 0007: an invited person has not agreed to participate yet, unlike an `added` participant
   whose row reserves a public spot.

7. Declining an invitation deletes the participant row. The existing self-serve join endpoint is
   idempotent for any existing participant row; retaining a `declined` row would permanently stop
   that user from joining the play independently. Deletion allows a later re-invitation or a normal
   self-service join. The host-only activity feed records `participant.invite_declined`, and a host
   may invite the same person again after a decline.

8. A play may have at most `max_players` outstanding invitations. Sending another invitation past
   that cap returns `409 Conflict`. A nil `max_players` follows the existing join behavior and is
   treated as a server error rather than receiving an invented fallback. The cap limits pending
   push notifications while scaling with the size of the game.

9. Sending an invitation is forbidden when either user has blocked the other. The existing
   mutual-block query is reused and returns `403 Forbidden`.

10. Invitations use the existing participant lifecycle and event history. Sending records
    `participant.invited` with the host as actor and invitee as subject. Acceptance records the
    event according to its resulting meaning:

    - `confirmed` → `participant.invite_accepted`, visible to the participant;
    - `waitlisted` → existing `participant.moved_to_waitlist`, host-only;
    - `requested` → existing `participant.join_requested`, host-only.

    Declining records `participant.invite_declined`. Hosts cannot cancel an outstanding invitation
    through roster removal; the invitee must decline it. Pending invitation events remain host-only. Only
    `participant.invite_accepted` is added to participant-visible event queries because it means
    the player actually joined the roster.

11. Invitation notifications are feed and push notifications. Sending an invitation notifies the
    invitee with `play.invited`; acceptance and decline notify the hosts with their corresponding
    invitation notification kinds. Notification failures are discarded and never fail the roster
    request.

12. The existing play detail endpoint's visibility behavior is unchanged. In particular,
    `GET /api/plays/{id}` does not check `visibility`, so an unlisted play remains a guessable UUID
    without revocation. The share button is only a client-side share of the existing play URL and
    does not introduce a new access or revocation model.

## Rationale

### Why a participant status instead of an invite table

An invitation already has the identity, rating snapshot, play relationship, lifecycle transitions,
privacy rules, and activity history of a participant row. Reusing that row avoids duplicate state
and lets acceptance and decline compose with existing roster controls.

### Why invitations do not reserve capacity

An invitation is a request for a player to decide, not an occupied or host-held spot. Reserving
capacity before acceptance would let unanswered invitations block players who are ready to join and
would make the public capacity calculation misleading.

### Why acceptance can be confirmed immediately

The invitation itself is the host's explicit selection and acceptance is the invitee's explicit
confirmation. Adding an `added` hop would ask the player to confirm a second time without adding a
new consent boundary. The direct `confirmed` landing preserves double opt-in while keeping the
special invitation flow distinct from a host accepting an uninvited request or waitlist row.

### Why the require-waitlist branch remains pending

Require-waitlist plays deliberately put every inbound player through host review. An invitation
should notify and prompt the selected player to request entry, not bypass that play-level policy.

### Why level checks are omitted

Level matching is a self-service admission heuristic. A host invitation is a deliberate host
choice, so applying the heuristic again would contradict the host-management rule that allows
hosts to override level mismatch while still enforcing capacity.

### Why invited identities are private

An invitee has not accepted and therefore does not yet belong to the roster. Showing only the
aggregate count gives viewers useful demand information while keeping pending identities within
the same host-only queue privacy boundary as requests and waitlisted users. The invitee's own row
is necessary for them to act on the invitation.

### Why decline deletes the row

The existing join operation treats any participant row as an idempotent existing membership. A
persisted declined status would turn a temporary decision into a permanent block on self-service
joining. Deletion preserves the ability to join normally and leaves the decline in the host audit
feed.

### Why cap outstanding invitations

The cap bounds the size of a host's pending queue and the number of invitation notifications a play
can generate. Tying it to `max_players` keeps the limit proportional to the game's intended size.

### Why pending invitation events stay host-only

Pending-queue events carry identities that are private in the roster payload. Exposing them through
the activity feed would bypass that privacy boundary, so only the event meaning that confirms an
actual roster membership is participant-visible.

## Consequences

### Positive

- Hosts can invite known registered users without introducing a second roster system.
- Acceptance preserves explicit consent for both the host and the invitee.
- Capacity, chat access, status transitions, and activity history remain based on existing roster
  primitives.
- Invitation identities remain private while hosts and invitees have the context they need.
- A declined invite does not prevent a later self-service join or re-invitation.
- Existing share URLs continue to work without adding a new token or revocation subsystem.

### Trade-offs

- Hosts can only invite registered users, and the invitee must act before the invitation changes the
  roster.
- There is no expiry or host cancellation notification; an outstanding invitation remains pending
  until the invitee accepts or declines it.
- The invitation cap and count-then-insert pattern are intentionally subject to the same
  concurrency limitations as existing roster mutations.
- A require-waitlist invite still needs host review after the invitee accepts.
- Unlisted play URLs remain directly accessible by guessable UUID, as they were before sharing was
  added.

## Implementation notes

- Add `invited_by_user_id` to `play_participants`; no status CHECK migration is needed because the
  status column is unconstrained.
- Add `ParticipantInvited` and the three invitation event types to both the Go constants and the
  Huma schema enums so the OpenAPI contract includes them.
- Keep `CountReservedPlayParticipants` and `IsPlayChatMember` unchanged: both already use only
  `confirmed` and `added`.
- Add host-only invitation creation at `POST /api/plays/{id}/invites` and self-service acceptance
  at `POST /api/plays/{id}/invites/me/accept`; reuse the existing leave path for decline and reject
  host removal of invited rows.
- Include invited rows only for hosts and the invitee in play details, while always exposing the
  aggregate invited count.
- Existing My Games membership queries already include any participant row and pass through its
  status, so invited plays require no SQL change there.
