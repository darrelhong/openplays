-- +goose Up

CREATE TABLE play_participants_new (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    play_id     TEXT NOT NULL REFERENCES plays(id) ON DELETE CASCADE,
    user_id     TEXT REFERENCES users(id) ON DELETE CASCADE,
    guest_name  TEXT,
    rating_code TEXT,
    rating_ord  INTEGER,
    status      TEXT NOT NULL, -- confirmed, waitlisted, added
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

INSERT INTO play_participants_new (
    id, play_id, user_id, guest_name, rating_code, rating_ord, status, created_at, updated_at
)
SELECT
    id, play_id, user_id, guest_name, rating_code, rating_ord, status, created_at, updated_at
FROM play_participants;

DROP TABLE play_participants;
ALTER TABLE play_participants_new RENAME TO play_participants;

CREATE UNIQUE INDEX idx_play_participants_play_user
    ON play_participants(play_id, user_id)
    WHERE user_id IS NOT NULL;

CREATE INDEX idx_play_participants_play_status
    ON play_participants(play_id, status, created_at, id);

CREATE INDEX idx_play_participants_user
    ON play_participants(user_id, created_at)
    WHERE user_id IS NOT NULL;

-- +goose Down

CREATE TABLE play_participants_old (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    play_id     TEXT NOT NULL REFERENCES plays(id) ON DELETE CASCADE,
    user_id     TEXT REFERENCES users(id) ON DELETE CASCADE,
    guest_name  TEXT,
    rating_code TEXT,
    rating_ord  INTEGER,
    status      TEXT NOT NULL, -- confirmed, waitlisted, added
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

INSERT INTO play_participants_old (
    id, play_id, user_id, guest_name, rating_code, rating_ord, status, created_at, updated_at
)
SELECT
    id,
    play_id,
    user_id,
    guest_name,
    rating_code,
    rating_ord,
    CASE status WHEN 'added' THEN 'waitlisted' ELSE status END,
    created_at,
    updated_at
FROM play_participants;

DROP TABLE play_participants;
ALTER TABLE play_participants_old RENAME TO play_participants;

CREATE UNIQUE INDEX idx_play_participants_play_user
    ON play_participants(play_id, user_id)
    WHERE user_id IS NOT NULL;

CREATE INDEX idx_play_participants_play_status
    ON play_participants(play_id, status, created_at, id);

CREATE INDEX idx_play_participants_user
    ON play_participants(user_id, created_at)
    WHERE user_id IS NOT NULL;
