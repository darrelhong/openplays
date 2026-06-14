-- +goose Up

CREATE TABLE play_favourites (
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    play_id    TEXT NOT NULL REFERENCES plays(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),

    PRIMARY KEY (user_id, play_id)
);

CREATE INDEX idx_play_favourites_play
    ON play_favourites(play_id);

-- +goose Down

DROP INDEX IF EXISTS idx_play_favourites_play;
DROP TABLE IF EXISTS play_favourites;
