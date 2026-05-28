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
        WHEN 'added' THEN 2
        WHEN 'waitlisted' THEN 3
        ELSE 4
    END,
    created_at ASC,
    id ASC;

-- name: ListPlayParticipantsByPlayAndStatus :many
SELECT * FROM play_participants
WHERE play_id = ? AND status = ?
ORDER BY created_at ASC, id ASC;

-- name: ListConfirmedParticipantPreviewsByPlay :many
SELECT
    pp.id,
    pp.play_id,
    pp.user_id,
    pp.guest_name,
    pp.rating_code,
    u.display_name,
    u.photo_url,
    u.sports_profile
FROM play_participants pp
LEFT JOIN users u ON u.id = pp.user_id
WHERE pp.play_id = ? AND pp.status = 'confirmed'
ORDER BY pp.created_at ASC, pp.id ASC;

-- name: ListConfirmedParticipantPreviewsByPlays :many
SELECT
    pp.id,
    pp.play_id,
    pp.user_id,
    pp.guest_name,
    pp.rating_code,
    u.display_name,
    u.photo_url,
    u.sports_profile
FROM play_participants pp
LEFT JOIN users u ON u.id = pp.user_id
WHERE pp.play_id IN (sqlc.slice('play_ids')) AND pp.status = 'confirmed'
ORDER BY pp.play_id ASC, pp.created_at ASC, pp.id ASC;

-- name: ListParticipantPreviewsByPlayAndStatus :many
SELECT
    pp.id,
    pp.play_id,
    pp.user_id,
    pp.guest_name,
    pp.rating_code,
    u.display_name,
    u.photo_url,
    u.sports_profile
FROM play_participants pp
LEFT JOIN users u ON u.id = pp.user_id
WHERE pp.play_id = ? AND pp.status = ?
ORDER BY pp.created_at ASC, pp.id ASC;

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

-- name: CountReservedPlayParticipants :one
SELECT COUNT(*) FROM play_participants
WHERE play_id = ? AND status IN ('confirmed', 'added');

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

-- name: DeletePlayParticipantsByPlay :exec
DELETE FROM play_participants
WHERE play_id = ?;
