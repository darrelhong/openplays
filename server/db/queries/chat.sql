-- name: CreateDMConversation :one
INSERT INTO chat_conversations (id, kind, dm_key)
VALUES (?, 'dm', ?)
ON CONFLICT(dm_key) DO UPDATE SET
    updated_at = chat_conversations.updated_at
RETURNING *;

-- name: CreateDMParticipant :exec
INSERT OR IGNORE INTO chat_dm_participants (conversation_id, user_id)
VALUES (?, ?);

-- name: CreatePlayConversation :one
INSERT INTO chat_conversations (id, kind, play_id)
VALUES (?, 'play', ?)
ON CONFLICT(play_id) WHERE play_id IS NOT NULL DO UPDATE SET
    updated_at = chat_conversations.updated_at
RETURNING *;

-- name: GetChatConversation :one
SELECT * FROM chat_conversations
WHERE id = ?;

-- name: GetDMConversationPeer :one
SELECT
    other.id,
    other.display_name,
    other.username,
    other.photo_url,
    other.status
FROM chat_conversations c
JOIN chat_dm_participants mine
    ON mine.conversation_id = c.id
    AND mine.user_id = sqlc.arg('viewer_id')
JOIN chat_dm_participants theirs
    ON theirs.conversation_id = c.id
    AND theirs.user_id <> sqlc.arg('viewer_id')
JOIN users other ON other.id = theirs.user_id
WHERE c.id = sqlc.arg('conversation_id')
  AND c.kind = 'dm';

-- name: ListDMConversationsByUser :many
SELECT
    c.id,
    c.kind,
    c.play_id,
    c.created_at,
    c.updated_at,
    other.id AS other_user_id,
    other.display_name AS other_display_name,
    other.username AS other_username,
    other.photo_url AS other_photo_url,
    lm.id AS last_message_id,
    lm.sender_user_id AS last_message_sender_user_id,
    lm.body AS last_message_body,
    lm.deleted_at AS last_message_deleted_at,
    lm.created_at AS last_message_created_at,
    sender.display_name AS last_message_sender_display_name,
    sender.username AS last_message_sender_username,
    sender.photo_url AS last_message_sender_photo_url,
    (
        SELECT COUNT(*)
        FROM chat_messages unread
        LEFT JOIN chat_read_states rs
            ON rs.conversation_id = unread.conversation_id
            AND rs.user_id = sqlc.arg('viewer_id')
        WHERE unread.conversation_id = c.id
          AND unread.sender_user_id <> sqlc.arg('viewer_id')
          AND unread.deleted_at IS NULL
          AND unread.id > COALESCE(rs.last_read_message_id, 0)
    ) AS unread_count
FROM chat_conversations c
JOIN chat_dm_participants mine
    ON mine.conversation_id = c.id
    AND mine.user_id = sqlc.arg('viewer_id')
JOIN chat_dm_participants theirs
    ON theirs.conversation_id = c.id
    AND theirs.user_id <> sqlc.arg('viewer_id')
JOIN users other
    ON other.id = theirs.user_id
    AND other.status = 'active'
LEFT JOIN chat_messages lm
    ON lm.id = (
        SELECT id
        FROM chat_messages
        WHERE conversation_id = c.id
        ORDER BY id DESC
        LIMIT 1
    )
LEFT JOIN users sender ON sender.id = lm.sender_user_id
WHERE c.kind = 'dm'
  AND NOT EXISTS (
    SELECT 1 FROM user_blocks ub
    WHERE (ub.blocker_id = sqlc.arg('viewer_id') AND ub.blocked_id = other.id)
       OR (ub.blocker_id = other.id AND ub.blocked_id = sqlc.arg('viewer_id'))
  )
ORDER BY COALESCE(lm.created_at, c.updated_at) DESC, c.id DESC
LIMIT sqlc.arg('limit');

-- name: ListPlayConversationsByUser :many
SELECT
    c.id,
    c.kind,
    c.play_id,
    c.created_at,
    c.updated_at,
    COALESCE(NULLIF(p.name, ''), v.name, NULLIF(p.venue, ''), 'Game') AS title,
    lm.id AS last_message_id,
    lm.sender_user_id AS last_message_sender_user_id,
    lm.body AS last_message_body,
    lm.deleted_at AS last_message_deleted_at,
    lm.created_at AS last_message_created_at,
    sender.display_name AS last_message_sender_display_name,
    sender.username AS last_message_sender_username,
    sender.photo_url AS last_message_sender_photo_url,
    (
        SELECT COUNT(*)
        FROM chat_messages unread
        LEFT JOIN chat_read_states rs
            ON rs.conversation_id = unread.conversation_id
            AND rs.user_id = sqlc.arg('viewer_id')
        WHERE unread.conversation_id = c.id
          AND unread.sender_user_id <> sqlc.arg('viewer_id')
          AND unread.deleted_at IS NULL
          AND unread.id > COALESCE(rs.last_read_message_id, 0)
    ) AS unread_count
FROM chat_conversations c
JOIN plays p ON p.id = c.play_id
LEFT JOIN venues v ON v.id = p.venue_id
LEFT JOIN chat_messages lm
    ON lm.id = (
        SELECT id
        FROM chat_messages
        WHERE conversation_id = c.id
        ORDER BY id DESC
        LIMIT 1
    )
LEFT JOIN users sender ON sender.id = lm.sender_user_id
WHERE c.kind = 'play'
  AND c.play_id IS NOT NULL
  AND (
    EXISTS (
        SELECT 1
        FROM play_hosts ph
        WHERE ph.play_id = p.id AND ph.user_id = sqlc.arg('viewer_id')
    )
    OR p.created_by = sqlc.arg('viewer_id')
    OR EXISTS (
        SELECT 1
        FROM play_participants pp
        WHERE pp.play_id = p.id
          AND pp.user_id = sqlc.arg('viewer_id')
          AND pp.status IN ('confirmed', 'added')
    )
  )
ORDER BY COALESCE(lm.created_at, c.updated_at) DESC, c.id DESC
LIMIT sqlc.arg('limit');

-- name: IsPlayChatMember :one
SELECT EXISTS (
    SELECT 1
    FROM plays p
    WHERE p.id = sqlc.arg('play_id')
      AND (
        EXISTS (
            SELECT 1
            FROM play_hosts ph
            WHERE ph.play_id = p.id AND ph.user_id = sqlc.arg('user_id')
        )
        OR p.created_by = sqlc.arg('user_id')
        OR EXISTS (
            SELECT 1
            FROM play_participants pp
            WHERE pp.play_id = p.id
              AND pp.user_id = sqlc.arg('user_id')
              AND pp.status IN ('confirmed', 'added')
        )
      )
);

-- name: ListPlayChatRecipientUserIDs :many
SELECT ph.user_id
FROM play_hosts ph
WHERE ph.play_id = sqlc.arg('play_id')
  AND ph.user_id <> sqlc.arg('exclude_user_id')
UNION
SELECT p.created_by AS user_id
FROM plays p
WHERE p.id = sqlc.arg('play_id')
  AND p.created_by IS NOT NULL
  AND p.created_by <> sqlc.arg('exclude_user_id')
UNION
SELECT pp.user_id
FROM play_participants pp
WHERE pp.play_id = sqlc.arg('play_id')
  AND pp.user_id IS NOT NULL
  AND pp.user_id <> sqlc.arg('exclude_user_id')
  AND pp.status IN ('confirmed', 'added')
ORDER BY user_id ASC;

-- name: CreateChatMessage :one
INSERT INTO chat_messages (conversation_id, sender_user_id, body)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetChatMessageWithSender :one
SELECT
    m.id,
    m.conversation_id,
    m.sender_user_id,
    m.body,
    m.deleted_at,
    m.created_at,
    u.display_name AS sender_display_name,
    u.username AS sender_username,
    u.photo_url AS sender_photo_url
FROM chat_messages m
JOIN users u ON u.id = m.sender_user_id
WHERE m.id = ? AND m.conversation_id = ?;

-- name: ListChatMessages :many
SELECT
    m.id,
    m.conversation_id,
    m.sender_user_id,
    m.body,
    m.deleted_at,
    m.created_at,
    u.display_name AS sender_display_name,
    u.username AS sender_username,
    u.photo_url AS sender_photo_url
FROM chat_messages m
JOIN users u ON u.id = m.sender_user_id
WHERE m.conversation_id = sqlc.arg('conversation_id')
  AND (
    sqlc.arg('before_id') = 0
    OR m.id < sqlc.arg('before_id')
  )
ORDER BY m.id DESC
LIMIT sqlc.arg('limit');

-- name: SoftDeleteChatMessageBySender :one
UPDATE chat_messages
SET
    body = NULL,
    deleted_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE id = ?
  AND conversation_id = ?
  AND sender_user_id = ?
  AND deleted_at IS NULL
RETURNING *;

-- name: UpsertChatReadState :exec
INSERT INTO chat_read_states (conversation_id, user_id, last_read_message_id, read_at)
VALUES (?, ?, ?, strftime('%Y-%m-%d %H:%M:%S+00:00', 'now'))
ON CONFLICT(conversation_id, user_id) DO UPDATE SET
    last_read_message_id = max(chat_read_states.last_read_message_id, excluded.last_read_message_id),
    read_at = excluded.read_at;
