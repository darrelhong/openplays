-- name: CreateUserFollow :one
INSERT INTO user_follows (follower_id, followee_id, status, accepted_at)
VALUES (
    sqlc.arg('follower_id'),
    sqlc.arg('followee_id'),
    sqlc.arg('status'),
    CASE
        WHEN sqlc.arg('status') = 'accepted' THEN strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
        ELSE NULL
    END
)
ON CONFLICT(follower_id, followee_id) DO NOTHING
RETURNING *;

-- name: GetUserFollow :one
SELECT * FROM user_follows
WHERE follower_id = ? AND followee_id = ?;

-- name: DeleteUserFollow :exec
DELETE FROM user_follows
WHERE follower_id = ? AND followee_id = ?;

-- name: ListPendingFollowRequests :many
SELECT
    uf.follower_id,
    uf.created_at,
    u.display_name,
    u.username,
    u.photo_url
FROM user_follows uf
JOIN users u ON u.id = uf.follower_id AND u.status = 'active'
WHERE uf.followee_id = sqlc.arg('followee_id')
  AND uf.status = 'pending'
  AND NOT EXISTS (
      SELECT 1
      FROM user_blocks ub
      WHERE (ub.blocker_id = uf.followee_id AND ub.blocked_id = uf.follower_id)
         OR (ub.blocker_id = uf.follower_id AND ub.blocked_id = uf.followee_id)
  )
  AND (sqlc.narg('cursor_created_at') IS NULL
    OR uf.created_at < sqlc.narg('cursor_created_at')
    OR (uf.created_at = sqlc.narg('cursor_created_at') AND uf.follower_id < sqlc.narg('cursor_follower_id')))
ORDER BY uf.created_at DESC, uf.follower_id DESC
LIMIT sqlc.arg('page_size');

-- name: CountPendingFollowRequests :one
SELECT COUNT(*)
FROM user_follows uf
JOIN users u ON u.id = uf.follower_id AND u.status = 'active'
WHERE uf.followee_id = ?
  AND uf.status = 'pending'
  AND NOT EXISTS (
      SELECT 1
      FROM user_blocks ub
      WHERE (ub.blocker_id = uf.followee_id AND ub.blocked_id = uf.follower_id)
         OR (ub.blocker_id = uf.follower_id AND ub.blocked_id = uf.followee_id)
  );

-- name: AcceptUserFollow :execrows
UPDATE user_follows
SET status = 'accepted',
    accepted_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE follower_id = ? AND followee_id = ? AND status = 'pending';

-- name: DeletePendingUserFollow :execrows
DELETE FROM user_follows
WHERE follower_id = ? AND followee_id = ? AND status = 'pending';

-- name: CountAcceptedFollowers :one
SELECT COUNT(*)
FROM user_follows
WHERE followee_id = ? AND status = 'accepted';

-- name: CountAcceptedFollowing :one
SELECT COUNT(*)
FROM user_follows
WHERE follower_id = ? AND status = 'accepted';

-- name: ListUserFollowers :many
SELECT
    u.id AS user_id,
    u.display_name,
    u.username,
    u.photo_url,
    u.sports_profile,
    uf.created_at
FROM user_follows uf
JOIN users u ON u.id = uf.follower_id AND u.status = 'active'
WHERE uf.followee_id = sqlc.arg('profile_user_id')
  AND uf.status = 'accepted'
  AND NOT EXISTS (
      SELECT 1
      FROM user_blocks ub
      WHERE (ub.blocker_id = sqlc.arg('viewer_id') AND ub.blocked_id = u.id)
         OR (ub.blocker_id = u.id AND ub.blocked_id = sqlc.arg('viewer_id'))
  )
  AND (sqlc.narg('cursor_created_at') IS NULL
    OR uf.created_at < sqlc.narg('cursor_created_at')
    OR (uf.created_at = sqlc.narg('cursor_created_at') AND uf.follower_id < sqlc.narg('cursor_user_id')))
ORDER BY uf.created_at DESC, uf.follower_id DESC
LIMIT sqlc.arg('page_size');

-- name: CountUserFollowers :one
SELECT COUNT(*)
FROM user_follows uf
JOIN users u ON u.id = uf.follower_id AND u.status = 'active'
WHERE uf.followee_id = sqlc.arg('profile_user_id')
  AND uf.status = 'accepted'
  AND NOT EXISTS (
      SELECT 1
      FROM user_blocks ub
      WHERE (ub.blocker_id = sqlc.arg('viewer_id') AND ub.blocked_id = u.id)
         OR (ub.blocker_id = u.id AND ub.blocked_id = sqlc.arg('viewer_id'))
  );

-- name: ListUserFollowing :many
SELECT
    u.id AS user_id,
    u.display_name,
    u.username,
    u.photo_url,
    u.sports_profile,
    uf.created_at
FROM user_follows uf
JOIN users u ON u.id = uf.followee_id AND u.status = 'active'
WHERE uf.follower_id = sqlc.arg('profile_user_id')
  AND uf.status = 'accepted'
  AND NOT EXISTS (
      SELECT 1
      FROM user_blocks ub
      WHERE (ub.blocker_id = sqlc.arg('viewer_id') AND ub.blocked_id = u.id)
         OR (ub.blocker_id = u.id AND ub.blocked_id = sqlc.arg('viewer_id'))
  )
  AND (sqlc.narg('cursor_created_at') IS NULL
    OR uf.created_at < sqlc.narg('cursor_created_at')
    OR (uf.created_at = sqlc.narg('cursor_created_at') AND uf.followee_id < sqlc.narg('cursor_user_id')))
ORDER BY uf.created_at DESC, uf.followee_id DESC
LIMIT sqlc.arg('page_size');

-- name: CountUserFollowing :one
SELECT COUNT(*)
FROM user_follows uf
JOIN users u ON u.id = uf.followee_id AND u.status = 'active'
WHERE uf.follower_id = sqlc.arg('profile_user_id')
  AND uf.status = 'accepted'
  AND NOT EXISTS (
      SELECT 1
      FROM user_blocks ub
      WHERE (ub.blocker_id = sqlc.arg('viewer_id') AND ub.blocked_id = u.id)
         OR (ub.blocker_id = u.id AND ub.blocked_id = sqlc.arg('viewer_id'))
  );
