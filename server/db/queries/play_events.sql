-- name: CreatePlayEvent :one
INSERT INTO play_events (
    play_id,
    event_type,
    actor_user_id,
    actor_display_name,
    subject_user_id,
    subject_display_name,
    participant_id,
    metadata
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: ListParticipantVisiblePlayEvents :many
SELECT * FROM (
    SELECT *
    FROM play_events
    WHERE play_id = ?
      AND event_type IN (
        'participant.joined_confirmed',
        'participant.added',
        'participant.confirmed',
        'participant.left_confirmed',
        'participant.left_added',
        'play.cancelled',
        'play.updated'
      )
    ORDER BY created_at DESC, id DESC
    LIMIT ?
)
ORDER BY created_at ASC, id ASC;

-- name: ListHostVisiblePlayEvents :many
SELECT * FROM (
    SELECT *
    FROM play_events
    WHERE play_id = ?
      AND event_type != 'play.created'
    ORDER BY created_at DESC, id DESC
    LIMIT ?
)
ORDER BY created_at ASC, id ASC;
