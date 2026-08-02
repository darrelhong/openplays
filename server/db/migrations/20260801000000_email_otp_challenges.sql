-- +goose Up

CREATE TABLE email_otp_challenges (
    email      TEXT PRIMARY KEY,
    code_hash  TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    attempts   INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now'))
);

CREATE INDEX idx_email_otp_challenges_expires_at ON email_otp_challenges(expires_at);

-- +goose Down

DROP TABLE IF EXISTS email_otp_challenges;
