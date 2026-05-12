-- +goose Up

CREATE TABLE play_participants (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    play_id     TEXT NOT NULL REFERENCES plays(id) ON DELETE CASCADE,
    user_id     TEXT REFERENCES users(id) ON DELETE CASCADE,
    guest_name  TEXT,
    rating_code TEXT,
    rating_ord  INTEGER,
    status      TEXT NOT NULL CHECK (status IN ('confirmed', 'waitlisted')),
    created_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    updated_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),

    CHECK (
        (user_id IS NOT NULL AND guest_name IS NULL)
        OR
        (user_id IS NULL AND guest_name IS NOT NULL AND length(trim(guest_name)) > 0)
    ),
    CHECK (
        (rating_code IS NULL AND rating_ord IS NULL)
        OR
        (rating_code IS NOT NULL AND rating_ord IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_play_participants_play_user
    ON play_participants(play_id, user_id)
    WHERE user_id IS NOT NULL;

CREATE INDEX idx_play_participants_play_status
    ON play_participants(play_id, status, created_at, id);

CREATE INDEX idx_play_participants_user
    ON play_participants(user_id, created_at)
    WHERE user_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_play_participants_user;
DROP INDEX IF EXISTS idx_play_participants_play_status;
DROP INDEX IF EXISTS idx_play_participants_play_user;
DROP TABLE IF EXISTS play_participants;
