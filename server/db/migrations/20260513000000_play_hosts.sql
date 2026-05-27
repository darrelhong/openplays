-- +goose Up

CREATE TABLE play_hosts (
    play_id    TEXT NOT NULL REFERENCES plays(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    updated_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),

    PRIMARY KEY (play_id, user_id)
);

CREATE INDEX idx_play_hosts_user
    ON play_hosts(user_id, play_id);

INSERT OR IGNORE INTO play_hosts (play_id, user_id)
SELECT id, created_by
FROM plays
WHERE created_by IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_play_hosts_user;
DROP TABLE IF EXISTS play_hosts;
