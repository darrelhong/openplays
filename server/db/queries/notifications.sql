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
