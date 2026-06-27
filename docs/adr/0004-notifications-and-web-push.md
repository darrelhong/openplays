# 0004 - Notifications and Web Push

- Status: Accepted
- Date: 2026-06-25

## Context

OpenPlays needs user notifications for play activity such as waitlist joins, confirmed joins,
player confirmations, players leaving, and host-added players. Some of these events should also
appear in the play activity history, but notifications are user-targeted: they are addressed to a
recipient and can be read or unread.

Browser push delivery is useful for timely updates, but it is not reliable enough to be the only
record. Users may deny notification permission, close the browser, lose network connectivity, use a
browser that drops a push, or have a stale push subscription. If a push is missed, the notification
should still appear in an in-app feed.

The notification system also needs to be flexible for future delivery channels and feed-like
features. A future following feed may reuse some of the same domain events, but a notification is
still different from a general feed item because it has a recipient, read state, and delivery
attempts.

## Decision

1. Store user notifications durably in SQLite in a `user_notifications` table.
2. Treat the in-app notification feed as the source of truth for delivered notification records.
3. Treat Web Push as a best-effort delivery channel for the same notification payload.
4. Store Web Push subscriptions in SQLite, keyed by push endpoint and associated with the
   authenticated user.
5. Store VAPID keys in SQLite as a singleton row so keys survive process restarts and deployments.
6. Validate push subscription endpoints against known browser push-service hosts before saving them.
7. Use a backend delivery policy, keyed by notification `kind`, to decide whether an event creates a
   feed row, sends Web Push, or both.
8. Create the in-app notification row synchronously on the request context when the delivery policy
   enables feed delivery.
9. Send Web Push network requests asynchronously using a detached context with a timeout when the
   delivery policy enables push delivery, so user actions do not block on third-party push-service
   latency.
10. Clean up stale push subscriptions when a push service returns `404 Not Found` or `410 Gone`.
11. Use `read_at IS NULL` as the unread marker.
12. Set `read_at` once, only when a notification is first marked read, and preserve that first-read
    timestamp.
13. Return notification timestamps as RFC3339 at the API boundary while storing SQLite timestamps in
    the internal UTC text format.
14. Keep one list endpoint for now, returning the current user's latest notifications with a limit
    of 50 and no pagination.
15. Let the backend own notification copy, notification kind names, target-user rules, delivery
    policy, and Web Push payload construction.
16. Keep optional `tag`, `kind`, `play_id`, `url`, and JSON `data` fields so future notification
    grouping, deep links, action rendering, and additional channels can be added without changing
    the core table shape immediately.
17. When a push arrives while the web app is open, have the service worker post the push payload to
    open tabs so the UI can show an optimistic unread row before reconciling with the backend feed.
18. Poll the notification feed every 10 minutes as a fallback; push messages and opening the
    notification popover refresh immediately.
19. If browser notifications are not enabled, show a compact popover prompt asking the user to enable
    them. If permission is denied, show browser-settings guidance instead of a request button.

## Rationale

### Why persist notifications separately from push

Push delivery is inherently best effort. A durable `user_notifications` row gives the product a
recoverable in-app feed even when native push is denied, unavailable, or lost. It also gives the UI a
simple read/unread model independent of push-service behavior.

### Why Web Push is asynchronous

Sending a push requires an outbound HTTP request to a third-party push service such as Mozilla,
Google/FCM, Apple, or Microsoft. Waiting for every host and device subscription inline would make
join, leave, confirm, and roster-management requests depend on external latency. The user-visible
action only needs any enabled feed notification to be recorded synchronously; native push can happen
in the background.

### Why feed and push are policy-controlled separately

Not every notification-kind should interrupt the user through native push. For example, a waitlist
join should appear in the host's notification feed, but it does not need to trigger browser push for
now. A central delivery policy lets us change feed/push behavior per notification kind without
spreading channel decisions across play routes.

### Why validate subscription endpoints

The browser provides the push endpoint URL, and the server later POSTs encrypted payloads to that
URL. Without validation, an authenticated user could register an arbitrary internal URL and turn push
delivery into server-side request forgery. Allowlisting known push-service hosts keeps the backend
from sending push requests to internal infrastructure.

### Why store VAPID keys in the database

VAPID keys identify the application server to browser push services. If keys were regenerated on
every restart, existing subscriptions could become unusable. Storing the singleton key pair in
SQLite keeps local and production behavior stable without introducing separate secret storage yet.

### Why `read_at` instead of a boolean

A timestamp supports both unread checks and future product behavior such as "read since" display,
debugging, or analytics. Updating only rows where `read_at IS NULL` preserves the first time the
notification became read, rather than changing it every time the feed opens.

### Why backend-owned copy and targeting

Notification copy, visibility rules, and delivery policy are part of product behavior, not just
presentation. Keeping them in the backend makes them easier to unit test and keeps clients from
having to reconstruct event semantics. This also prepares the same notification kinds for future
email, push, or other delivery channels.

### Why the service worker messages open tabs

The service worker can receive Web Push while a page is already open, but it does not own the Svelte
application state. Posting the push payload to open tabs lets the UI show immediate feedback, such as
the bell dot and an optimistic row, while the page still fetches `/notifications` to reconcile with
the durable feed.

## Consequences

### Positive

- Users have a notification feed even when native push is unavailable or missed.
- User actions are not blocked by push-service network latency.
- Push subscriptions and VAPID keys survive process restarts.
- Read/unread state is queryable and durable.
- Notification behavior can be tested in backend unit tests.
- Future channels can reuse the same notification creation path.
- Notification kinds can be tuned independently for feed and push delivery.
- Open tabs can update the bell indicator immediately when push is delivered.

### Trade-offs

- In-app notifications and play activity events are separate records, so some domain events may be
  written to both systems.
- The frontend currently derives unread state from the latest 50 fetched notifications; there is no
  dedicated unread-count endpoint yet.
- Feed-only notifications do not update open tabs immediately; they appear on popover open, page
  reload, or the 10-minute fallback poll unless we add SSE/WebSocket-style realtime delivery.
- There is no explicit unsubscribe endpoint yet; stale subscriptions are removed only when a push
  send returns `404` or `410`.
- VAPID keys are stored in the application database, which is simple for now but may move to managed
  secret storage later.
- Web Push service allowlists must be maintained if browser vendors add or change push endpoint
  hostnames.

## Implementation notes

- `web_push_vapid_keys` stores the singleton VAPID key pair.
- `web_push_subscriptions` stores authenticated browser push subscriptions.
- `user_notifications` stores the in-app feed rows.
- `read_at IS NULL` means unread; non-null `read_at` means read.
- Mark-read queries only update rows where `read_at IS NULL`.
- `idx_user_notifications_user_unread` is a partial index on `user_id` where `read_at IS NULL`.
- Web Push endpoint validation currently allows Google/FCM, Mozilla, Microsoft, and Apple push
  service hosts.
- Web Push sends run in a goroutine with a detached context and timeout after enabled feed delivery
  is stored.
- Notification rows are listed through one authenticated endpoint, limited to 50 rows for now.
- Play notification kinds currently include waitlist join, direct confirmed join, player added,
  player confirmed, and player left.
- `play.waitlist_joined` is currently feed-only. `play.player_added`, `play.player_joined`,
  `play.player_confirmed`, and `play.player_left` currently create feed rows and send push.
- The service worker posts `{ type: "openplays:notification-received", notification: payload }` to
  open windows after push delivery.
- The web notification popover adds a temporary optimistic row from the push payload, then refreshes
  the feed from the backend.
- The notification feed polls every 10 minutes as a fallback.
