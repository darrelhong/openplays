-- name: GetWebPushVAPIDKeys :one
SELECT * FROM web_push_vapid_keys
WHERE id = 1;

-- name: CreateWebPushVAPIDKeys :one
INSERT INTO web_push_vapid_keys (
    id,
    public_key,
    private_key
) VALUES (
    1, ?, ?
)
-- No-op update returns the existing singleton row during concurrent first boot.
ON CONFLICT(id) DO UPDATE SET
    public_key = web_push_vapid_keys.public_key
RETURNING *;

-- name: UpsertWebPushSubscription :exec
INSERT INTO web_push_subscriptions (
    endpoint,
    user_id,
    auth,
    p256dh,
    expiration_time_ms
) VALUES (
    ?, ?, ?, ?, ?
)
ON CONFLICT(endpoint) DO UPDATE SET
    user_id = excluded.user_id,
    auth = excluded.auth,
    p256dh = excluded.p256dh,
    expiration_time_ms = excluded.expiration_time_ms,
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now');

-- name: ListWebPushSubscriptionsByUser :many
SELECT * FROM web_push_subscriptions
WHERE user_id = ?
ORDER BY updated_at DESC, endpoint ASC;

-- name: DeleteWebPushSubscription :exec
DELETE FROM web_push_subscriptions
WHERE user_id = ? AND endpoint = ?;

-- name: CreateUserNotification :one
INSERT INTO user_notifications (
    id,
    user_id,
    title,
    body,
    url,
    tag,
    kind,
    play_id,
    data
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: UpsertChatUserNotificationByTag :one
INSERT INTO user_notifications (
    id,
    user_id,
    title,
    body,
    url,
    tag,
    kind,
    play_id,
    data
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
)
ON CONFLICT(user_id, tag) WHERE tag IS NOT NULL AND kind = 'chat.message' DO UPDATE SET
    title = excluded.title,
    body = excluded.body,
    url = excluded.url,
    kind = excluded.kind,
    play_id = excluded.play_id,
    data = excluded.data,
    read_at = NULL,
    created_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
RETURNING *;

-- name: ListUserNotifications :many
SELECT * FROM user_notifications
WHERE user_id = ?
ORDER BY created_at DESC, id DESC
LIMIT ?;

-- name: MarkAllUserNotificationsRead :exec
UPDATE user_notifications
SET read_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE user_id = ? AND read_at IS NULL;

-- name: MarkUserNotificationsRead :exec
UPDATE user_notifications
SET read_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE user_id = ?
  AND id IN (sqlc.slice('ids'))
  AND read_at IS NULL;

-- name: FanOutFollowerNotifications :exec
-- Writes one feed row per accepted follower of the actor in a single statement.
-- The recipient set and the insert are the same query, so a play with 500
-- followers costs one round trip instead of 500.
-- lower(hex(randomblob(16))) is intentionally a 32-character opaque TEXT
-- primary key, not a formatted UUID; notification IDs are opaque strings.
INSERT INTO user_notifications (id, user_id, title, body, url, tag, kind, play_id)
SELECT
    lower(hex(randomblob(16))),
    uf.follower_id,
    sqlc.arg('title'),
    sqlc.narg('body'),
    sqlc.narg('url'),
    sqlc.narg('tag'),
    sqlc.arg('kind'),
    sqlc.narg('play_id')
FROM user_follows uf
JOIN users u ON u.id = uf.follower_id
WHERE uf.followee_id = sqlc.arg('actor_user_id')
  AND uf.status = 'accepted'
  AND u.status = 'active'
  AND NOT EXISTS (
      SELECT 1
      FROM user_blocks ub
      WHERE (ub.blocker_id = sqlc.arg('actor_user_id') AND ub.blocked_id = uf.follower_id)
         OR (ub.blocker_id = uf.follower_id AND ub.blocked_id = sqlc.arg('actor_user_id'))
  );

-- name: ListAcceptedFollowerIDs :many
SELECT uf.follower_id
FROM user_follows uf
JOIN users u ON u.id = uf.follower_id
WHERE uf.followee_id = sqlc.arg('actor_user_id')
  AND uf.status = 'accepted'
  AND u.status = 'active'
  AND NOT EXISTS (
      SELECT 1
      FROM user_blocks ub
      WHERE (ub.blocker_id = sqlc.arg('actor_user_id') AND ub.blocked_id = uf.follower_id)
         OR (ub.blocker_id = uf.follower_id AND ub.blocked_id = sqlc.arg('actor_user_id'))
  )
ORDER BY uf.follower_id ASC;
