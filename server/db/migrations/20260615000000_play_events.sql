-- +goose Up

CREATE TABLE play_events (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    play_id              TEXT NOT NULL REFERENCES plays(id) ON DELETE CASCADE,
    event_type           TEXT NOT NULL,
    actor_user_id        TEXT REFERENCES users(id) ON DELETE SET NULL,
    actor_display_name   TEXT,
    subject_user_id      TEXT REFERENCES users(id) ON DELETE SET NULL,
    subject_display_name TEXT,
    participant_id       INTEGER,
    metadata             TEXT,
    created_at           TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now'))
);

CREATE INDEX idx_play_events_play_created
    ON play_events(play_id, created_at DESC, id DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_play_events_play_created;
DROP TABLE IF EXISTS play_events;
