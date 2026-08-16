# 0012 - User follows

- Status: Accepted
- Date: 2026-08-11
- Amends: [0004 - Notifications and Web Push](0004-notifications-and-web-push.md)

## Context

OpenPlays has no persistent positive relationship between users. Blocks provide a negative,
mutual-hide edge, and shared play membership can identify past co-players for suggested-player
search, but users cannot ask to be notified about another user's activity. Follow adds that
relationship and the notification fan-out built on it.

The relationship needs to support both immediate follows and profiles that require approval. It
also needs to preserve the existing privacy boundary around user-to-user lists and the existing
mutual-block behavior without coupling follow rows to block-row writes.

## Decision

1. Store follows in one `user_follows` table keyed by `(follower_id, followee_id)`. The table has a
   `status` column whose current values are `accepted` and `pending`, but the database does not
   constrain the status set. A `CHECK (follower_id <> followee_id)` permanently rejects
   self-follows.

2. Following is one-way and immediate by default. A `PUT` creates an `accepted` row unless the
   followee has enabled restricted follows.

3. Add `users.restrict_follow`, a non-null boolean defaulting to false. When it is enabled on the
   followee, new follows create `pending` rows and the followee must accept or decline them.
   The flag is read when the follow is created and belongs to the followee, not the follower.

4. Enabling restricted follows is not retroactive. Existing `accepted` rows remain accepted; only
   follows created after the setting changes are pending. There is no bulk conversion of existing
   relationships.

5. Declining a follow request deletes the row. The same delete operation serves unfollowing and
   cancelling the requester's own pending follow. Keeping a declined row would make the follow
   endpoint's idempotent existing-row behavior permanently prevent a later follow request.

6. Follower and following counts are public in signed-in profile payloads. The actual lists are
   visible only to the profile owner or an accepted follower of that profile. Pending requesters
   and strangers receive `403 Forbidden` from both list endpoints.

7. Add five notification kinds, all delivered to both the in-app feed and push:

   - `follow.started` to the followee when an unrestricted follow is accepted;
   - `follow.requested` to the followee when a restricted follow is created;
   - `follow.accepted` to the requester when a pending follow is accepted;
   - `follow.play_hosted` to accepted followers when someone they follow creates a play;
   - `follow.play_joined` to accepted followers when someone they follow enters a public roster.

   All five use the `follow.` namespace because the follow edge is the reason for delivery.

8. Follower notification fan-out runs off the request path and inserts all recipients with one
   SQL `INSERT ... SELECT`. Play creation and roster joins must not perform one synchronous insert
   per follower.

9. Unlisted plays never produce `follow.play_hosted` or `follow.play_joined` notifications. A
   direct unlisted URL is already a weak privacy boundary; broadcasting it to followers would
   turn that boundary into an active disclosure.

10. Mutual blocks are filtered when follow rows are read or fan-out recipients are selected. No
    follow rows are deleted as a side effect of creating a block; filtering remains correct for
    blocks created by any future mechanism as well as the current data model.

## Rationale

### Why one table with a status instead of separate follows and requests tables

An accepted follow and a pending follow are states of the same directed relationship. One primary
key prevents duplicate edges, lets `PUT` remain idempotent, and makes acceptance a status
transition rather than a cross-table move. Leaving the status unconstrained preserves room for
future states without another migration.

### Why restrict follows belongs to the followee

The followee controls whether new followers need approval. Reading the setting at follow time makes
the rule predictable and preserves existing accepted relationships when the setting changes.

### Why decline deletes the row

Decline is a reversible decision, not a permanent relationship state. Deletion allows the requester
to try again later and keeps the follow endpoint's existing-row idempotency from blocking a normal
future follow.

### Why counts are public but lists are gated

Counts provide useful profile context without exposing a social graph to strangers. List access is
limited to the profile owner and accepted followers, so a user must be part of the relationship to
inspect it. Restricted profiles therefore require acceptance before a requester can see the
profile's lists.

### Why unlisted plays are excluded from fan-out

Unlisted means the host chose not to broadcast the play through discovery. Sending it to followers
would bypass that choice. The existing direct-link behavior remains unchanged, but follow fan-out
does not widen the audience.

### Why blocks filter instead of cascade

The current application has no HTTP block endpoint, and deleting follow rows during block creation
would make relationship cleanup depend on a write path that does not exist. Read-time mutual-block
filters consistently hide the edge and avoid coupling two independent relationship tables.

## Consequences

### Positive

- Users can receive activity notifications from people they choose to follow.
- Restricted profiles can approve new followers without introducing a separate request table.
- Counts remain useful to all signed-in profile viewers while follower identities stay private.
- One-statement fan-out keeps notification cost bounded by database work rather than request-time
  loops.
- Existing accepted relationships survive a later change to follow restrictions.

### Trade-offs

- The follow lists are intentionally asymmetric: a stranger cannot inspect them, and a pending
  requester cannot inspect a restricted profile's lists until accepted.
- `follow.play_joined` is intentionally noisy: it fires for every public roster entry by someone a
  user follows, not only for plays that person hosts. Since there is no per-kind preference system,
  unfollowing is the only escape hatch.
- Fan-out is detached, so a successful play or roster mutation can briefly precede its follower
  notifications.
- Follow rows are not removed when a block is created; read and fan-out filters hide blocked pairs
  instead.
- There is no expiry or bulk cleanup for pending requests.

## Implementation notes

- Migration `20260810000000_user_follows.sql` creates `user_follows`, its followee/status and
  pending indexes, and `users.restrict_follow`.
- Follow mutations are SvelteKit form actions backed by `PUT`/`DELETE /users/{username}/follow`.
- Profile payloads expose follower/following counts, restriction state, and the viewer's follow
  state; list endpoints enforce the owner-or-accepted-follower gate.
- Notification fan-out uses a single `INSERT ... SELECT` over accepted followers and excludes
  mutually blocked pairs and inactive recipients.
