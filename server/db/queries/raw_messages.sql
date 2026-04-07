-- name: InsertRawMessage :one
INSERT INTO raw_messages (source, sender_username, message_text, message_time, content_hash, status)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetPendingJob :one
SELECT *
FROM raw_messages
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT 1;

-- name: GetRetryJob :one
SELECT *
FROM raw_messages
WHERE status = 'failed' AND next_retry_at <= strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
ORDER BY next_retry_at ASC
LIMIT 1;

-- name: MarkProcessing :exec
UPDATE raw_messages
SET status = 'processing', updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE id = ?;

-- name: MarkDone :exec
UPDATE raw_messages
SET status = 'done', llm_response = ?, updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE id = ?;

-- name: MarkFailed :exec
UPDATE raw_messages
SET status = 'failed',
    retry_count = retry_count + 1,
    next_retry_at = ?,
    last_error = ?,
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE id = ?;

-- name: MarkSkipped :exec
UPDATE raw_messages
SET status = 'skipped', updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE id = ?;

-- name: GetRecentMessageTexts :many
SELECT id, message_text
FROM raw_messages
WHERE created_at > ?
  AND status != 'skipped';
