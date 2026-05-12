-- name: CreatePlayParticipant :one
INSERT INTO play_participants (
    play_id, user_id, guest_name, rating_code, rating_ord, status
) VALUES (
    ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetPlayParticipantByID :one
SELECT * FROM play_participants
WHERE id = ?;

-- name: GetPlayParticipantByPlayAndUser :one
SELECT * FROM play_participants
WHERE play_id = ? AND user_id = ?;

-- name: ListPlayParticipantsByPlay :many
SELECT * FROM play_participants
WHERE play_id = ?
ORDER BY
    CASE status
        WHEN 'confirmed' THEN 1
        WHEN 'waitlisted' THEN 2
        ELSE 4
    END,
    created_at ASC,
    id ASC;

-- name: ListPlayParticipantsByPlayAndStatus :many
SELECT * FROM play_participants
WHERE play_id = ? AND status = ?
ORDER BY created_at ASC, id ASC;

-- name: ListPlayParticipantsByUser :many
SELECT * FROM play_participants
WHERE user_id = ?
ORDER BY created_at DESC, id DESC;

-- name: CountPlayParticipantsByStatus :one
SELECT COUNT(*) FROM play_participants
WHERE play_id = ? AND status = ?;

-- name: CountConfirmedPlayParticipants :one
SELECT COUNT(*) FROM play_participants
WHERE play_id = ? AND status = 'confirmed';

-- name: UpdatePlayParticipantStatus :one
UPDATE play_participants
SET
    status = ?,
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE id = ?
RETURNING *;

-- name: DeletePlayParticipant :exec
DELETE FROM play_participants
WHERE id = ?;

-- name: DeletePlayParticipantByPlayAndUser :exec
DELETE FROM play_participants
WHERE play_id = ? AND user_id = ?;
