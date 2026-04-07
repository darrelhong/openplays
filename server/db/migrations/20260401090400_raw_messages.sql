-- +goose Up

-- raw_messages is the job queue for LLM processing.
-- Each incoming message is stored here, then processed async by the worker.
CREATE TABLE raw_messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at      TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    updated_at      TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),

    source          TEXT NOT NULL,
    sender_username TEXT NOT NULL,
    message_text    TEXT NOT NULL,
    message_time    TIMESTAMP NOT NULL,

    -- for deduplication with trigram hash
    content_hash    TEXT NOT NULL,

    -- job processing state
    status          TEXT NOT NULL DEFAULT 'pending', -- pending, processing, done, failed, skipped
    retry_count     INTEGER NOT NULL DEFAULT 0,
    next_retry_at   TIMESTAMP,
    last_error      TEXT,

    -- raw llm json response
    llm_response    TEXT
);

CREATE INDEX idx_raw_messages_status ON raw_messages(status, next_retry_at);

-- +goose Down
DROP TABLE IF EXISTS raw_messages;
