-- name: CreatePlayHost :one
INSERT INTO play_hosts (
    play_id, user_id
) VALUES (
    ?, ?
)
ON CONFLICT(play_id, user_id) DO UPDATE SET
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
RETURNING *;

-- name: GetPlayHost :one
SELECT * FROM play_hosts
WHERE play_id = ? AND user_id = ?;

-- name: ListPlayHostUserIDsByPlay :many
SELECT user_id FROM play_hosts
WHERE play_id = ?
ORDER BY created_at ASC, user_id ASC;

-- name: ListPlayHostUserIDsByPlays :many
SELECT play_id, user_id FROM play_hosts
WHERE play_id IN (sqlc.slice('play_ids'))
ORDER BY play_id ASC, created_at ASC, user_id ASC;

-- name: DeletePlayHostsByPlay :exec
DELETE FROM play_hosts
WHERE play_id = ?;
