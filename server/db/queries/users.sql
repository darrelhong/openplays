-- name: UpsertUserByGoogleID :one
INSERT INTO users (id, email, username, display_name, photo_url, google_id, updated_at)
VALUES (?, ?, ?, ?, ?, ?, strftime('%Y-%m-%d %H:%M:%S+00:00', 'now'))
ON CONFLICT(google_id) DO UPDATE SET
    email = excluded.email,
    display_name = excluded.display_name,
    photo_url = excluded.photo_url,
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
RETURNING *;

-- name: UpsertUserByFacebookID :one
INSERT INTO users (id, email, username, display_name, photo_url, facebook_id, updated_at)
VALUES (?, ?, ?, ?, ?, ?, strftime('%Y-%m-%d %H:%M:%S+00:00', 'now'))
ON CONFLICT(facebook_id) DO UPDATE SET
    email = excluded.email,
    display_name = excluded.display_name,
    photo_url = excluded.photo_url,
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
RETURNING *;

-- name: LinkGoogleID :one
UPDATE users SET google_id = ?, updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE email = ? AND google_id IS NULL
RETURNING *;

-- name: LinkFacebookID :one
UPDATE users SET facebook_id = ?, updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE email = ? AND facebook_id IS NULL
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetActiveUserProfileByUsername :one
SELECT id, display_name, username, photo_url, sports_profile
FROM users
WHERE username = ? AND status = 'active';

-- name: CountRosteredPlaysByUser :one
SELECT COUNT(DISTINCT play_id)
FROM (
    SELECT play_id
    FROM play_hosts ph
    WHERE ph.user_id = sqlc.arg('user_id')
    UNION
    SELECT play_id
    FROM play_participants pp
    WHERE pp.user_id = sqlc.arg('user_id') AND pp.status IN ('confirmed', 'added')
);

-- name: SearchActiveUsers :many
SELECT id, display_name, username, photo_url, sports_profile
FROM users
WHERE status = 'active'
  AND (
    sqlc.arg('query') = ''
    OR lower(display_name) LIKE '%' || lower(sqlc.arg('query')) || '%'
    OR lower(COALESCE(username, '')) LIKE '%' || lower(sqlc.arg('query')) || '%'
  )
ORDER BY display_name ASC, id ASC
LIMIT sqlc.arg('limit');

-- name: UpdateUserProfile :one
UPDATE users SET
    display_name = ?,
    username = ?,
    photo_url = ?,
    sports_profile = ?,
    contact_info = ?,
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE id = ?
RETURNING *;

-- name: UpdateUserStatus :exec
UPDATE users SET
    status = ?,
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE id = ?;

-- name: CreateSession :exec
INSERT INTO sessions (token, user_id, expires_at)
VALUES (?, ?, ?);

-- name: GetSessionWithUser :one
SELECT
    s.token,
    s.user_id,
    s.expires_at,
    u.id AS user_id_2,
    u.email,
    u.username,
    u.display_name,
    u.photo_url,
    u.google_id,
    u.facebook_id,
    u.status,
    u.sports_profile,
    u.contact_info,
    u.created_at,
    u.updated_at
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token = ? AND s.expires_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now');

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= strftime('%Y-%m-%d %H:%M:%S+00:00', 'now');

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = ?;

-- name: RefreshSession :exec
UPDATE sessions SET expires_at = ? WHERE token = ?;

-- name: CreateBlock :exec
INSERT OR IGNORE INTO user_blocks (blocker_id, blocked_id) VALUES (?, ?);

-- name: DeleteBlock :exec
DELETE FROM user_blocks WHERE blocker_id = ? AND blocked_id = ?;

-- name: ListBlockedUserIDs :many
SELECT blocked_id FROM user_blocks WHERE blocker_id = ?;

-- name: IsBlocked :one
-- Returns 1 if either user blocked the other (mutual check for hide logic)
SELECT EXISTS(
    SELECT 1 FROM user_blocks
    WHERE (blocker_id = ? AND blocked_id = ?)
       OR (blocker_id = ? AND blocked_id = ?)
) AS is_blocked;
