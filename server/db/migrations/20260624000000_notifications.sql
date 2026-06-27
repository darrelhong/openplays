-- +goose Up

CREATE TABLE web_push_vapid_keys (
    id          INTEGER PRIMARY KEY CHECK (id = 1), -- singleton keypair used to sign Web Push messages
    public_key  TEXT NOT NULL,
    private_key TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    updated_at  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now'))
);

CREATE TABLE web_push_subscriptions (
    endpoint           TEXT PRIMARY KEY, -- browser-vendor push URL, allowlisted before insert
    user_id            TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- subscribed OpenPlays user/device owner
    auth               TEXT NOT NULL, -- Web Push auth secret from PushSubscription.keys.auth
    p256dh             TEXT NOT NULL, -- Web Push ECDH public key from PushSubscription.keys.p256dh
    expiration_time_ms INTEGER, -- nullable browser-provided expirationTime in epoch milliseconds
    created_at         TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    updated_at         TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now'))
);

CREATE INDEX idx_web_push_subscriptions_user
    ON web_push_subscriptions(user_id, updated_at DESC);

CREATE TABLE user_notifications (
    id         TEXT PRIMARY KEY, -- UUID v4
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- notification recipient
    title      TEXT NOT NULL, -- short display title for push and feed rows
    body       TEXT, -- optional display body
    url        TEXT, -- optional app-relative destination, e.g. /play/{id}
    tag        TEXT, -- optional Web Push collapse key; repeated tags can replace native notifications
    kind       TEXT, -- event kind, e.g. play.player_added, play.player_joined, play.player_left
    play_id    TEXT, -- optional play context used by the frontend for typed links
    data       TEXT, -- optional JSON metadata for future feed actions
    read_at    TIMESTAMP, -- set once the feed marks this notification read
    created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now'))
);

CREATE INDEX idx_user_notifications_user_created
    ON user_notifications(user_id, created_at DESC, id DESC);

CREATE INDEX idx_user_notifications_user_unread
    ON user_notifications(user_id)
    WHERE read_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_user_notifications_user_unread;
DROP INDEX IF EXISTS idx_user_notifications_user_created;
DROP TABLE IF EXISTS user_notifications;
DROP INDEX IF EXISTS idx_web_push_subscriptions_user;
DROP TABLE IF EXISTS web_push_subscriptions;
DROP TABLE IF EXISTS web_push_vapid_keys;
