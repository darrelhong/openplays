-- +goose Up

CREATE TABLE users (
    id               TEXT PRIMARY KEY,     -- UUID v4
    email            TEXT UNIQUE NOT NULL,  -- verified email from OAuth provider
    username         TEXT UNIQUE,           -- optional handle, could infer from email (set in profile, used for @mentions and profile URLs)
    display_name     TEXT NOT NULL,         -- full name from OAuth, (user-editable)
    photo_url        TEXT,                  -- profile picture URL from OAuth provider 
    google_id        TEXT UNIQUE,           -- Google subject ID from ID token
    facebook_id      TEXT UNIQUE,           -- Facebook user ID (future)
    status           TEXT NOT NULL DEFAULT 'active', -- account status: "active" | "suspended" | "banned"
    sports_profile   TEXT,                  -- JSON per-sport profile, e.g. {"badminton":{"level":"LI","strengths":["backhand","defense"]},"tennis":{"level":"3.5"}}
    contact_info     TEXT,                  -- JSON contact methods, e.g. {"telegram":"@darrel_h","whatsapp":"+6591234567"}
    created_at       TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    updated_at       TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now'))
);

CREATE TABLE sessions (
    token      TEXT PRIMARY KEY,            -- 32 random bytes hex-encoded
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,          -- 30-day rolling expiry, refreshed on each authenticated request
    created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now'))
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

CREATE TABLE user_blocks (
    blocker_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- user who initiated block
    blocked_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- user who got blocked
    created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    PRIMARY KEY (blocker_id, blocked_id)
);

-- Reverse lookup: "who blocked me?" for mutual hide in play listings
CREATE INDEX idx_user_blocks_blocked_id ON user_blocks(blocked_id);

ALTER TABLE plays ADD COLUMN created_by TEXT REFERENCES users(id); -- NULL for telegram-scraped plays, set for user-created plays

-- +goose Down

ALTER TABLE plays DROP COLUMN created_by;
DROP TABLE IF EXISTS user_blocks;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
