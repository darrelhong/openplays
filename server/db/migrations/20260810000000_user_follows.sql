-- +goose Up

CREATE TABLE user_follows (
    follower_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- user doing the following
    followee_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- user being followed
    status      TEXT NOT NULL DEFAULT 'accepted', -- accepted | pending (no CHECK: new values stay migration-free)
    created_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    accepted_at TIMESTAMP, -- set when a pending request is accepted; NULL while pending

    PRIMARY KEY (follower_id, followee_id),
    CHECK (follower_id <> followee_id)
);

-- Fan-out and follower lists read by followee; pending inbox is a hot subset.
CREATE INDEX idx_user_follows_followee ON user_follows(followee_id, status, created_at DESC);
CREATE INDEX idx_user_follows_pending ON user_follows(followee_id, created_at DESC)
    WHERE status = 'pending';

ALTER TABLE users ADD COLUMN restrict_follow BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE users DROP COLUMN restrict_follow;
DROP INDEX IF EXISTS idx_user_follows_pending;
DROP INDEX IF EXISTS idx_user_follows_followee;
DROP TABLE IF EXISTS user_follows;
