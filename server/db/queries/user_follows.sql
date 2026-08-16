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
