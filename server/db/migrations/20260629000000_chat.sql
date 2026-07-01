-- +goose Up

CREATE TABLE chat_conversations (
    id         TEXT PRIMARY KEY, -- UUID v4
    kind       TEXT NOT NULL, -- conversation kind: "dm" | "play"
    play_id    TEXT REFERENCES plays(id) ON DELETE CASCADE, -- set for play conversations
    dm_key     TEXT UNIQUE, -- stable sorted user pair key for DMs, e.g. user_a:user_b
    created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    updated_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    CHECK (
        (kind = 'dm' AND dm_key IS NOT NULL AND play_id IS NULL)
        OR (kind = 'play' AND play_id IS NOT NULL AND dm_key IS NULL)
    )
);

CREATE UNIQUE INDEX idx_chat_conversations_play
    ON chat_conversations(play_id)
    WHERE play_id IS NOT NULL;

-- DM participants live here; play chat membership is derived from the play roster.
CREATE TABLE chat_dm_participants (
    conversation_id TEXT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at      TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX idx_chat_dm_participants_user
    ON chat_dm_participants(user_id, conversation_id);

CREATE TABLE chat_messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id TEXT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    sender_user_id  TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body            TEXT, -- plain text; nulled when sender soft-deletes the message
    deleted_at      TIMESTAMP, -- set when sender soft-deletes the message
    created_at      TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now'))
);

CREATE INDEX idx_chat_messages_conversation
    ON chat_messages(conversation_id, id DESC);

CREATE TABLE chat_read_states (
    conversation_id       TEXT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    user_id               TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_read_message_id  INTEGER NOT NULL DEFAULT 0, -- cursor, not an FK: 0 and retained/pruned message IDs are valid
    read_at               TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    PRIMARY KEY (conversation_id, user_id)
);

-- +goose Down

DROP TABLE IF EXISTS chat_read_states;
DROP INDEX IF EXISTS idx_chat_messages_conversation;
DROP TABLE IF EXISTS chat_messages;
DROP INDEX IF EXISTS idx_chat_dm_participants_user;
DROP TABLE IF EXISTS chat_dm_participants;
DROP INDEX IF EXISTS idx_chat_conversations_play;
DROP TABLE IF EXISTS chat_conversations;
