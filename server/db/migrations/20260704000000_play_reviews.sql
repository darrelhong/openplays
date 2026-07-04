-- +goose Up

-- Post-game peer reviews. One row per (play, reviewer, reviewee), written
-- after the play ends and editable for 14 days (window enforced in Go).
-- Rows are never deleted: edits UPSERT onto the same row.
CREATE TABLE play_reviews (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    play_id          TEXT NOT NULL REFERENCES plays(id) ON DELETE CASCADE,
    reviewer_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reviewee_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating           INTEGER, -- 1-5 stars; anonymous, only aggregates are ever exposed
    props            TEXT NOT NULL DEFAULT '[]', -- JSON array of prop slugs ("great_sport"); vocabulary lives in internal/reviews
    shoutout         TEXT, -- free-text praise; attributed to the reviewer on the profile
    created_at       TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    updated_at       TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),

    CHECK (reviewer_user_id <> reviewee_user_id), -- no self-reviews
    CHECK (rating IS NULL OR rating BETWEEN 1 AND 5),
    CHECK (json_valid(props)),
    CHECK (shoutout IS NULL OR length(trim(shoutout)) > 0), -- present means non-blank
    CHECK (rating IS NOT NULL OR props <> '[]' OR shoutout IS NOT NULL), -- every part is optional, but not all at once

    UNIQUE (play_id, reviewer_user_id, reviewee_user_id)
);

-- Serves the profile aggregates: rating average, prop counts, and the
-- newest-first shoutout list for one reviewee.
CREATE INDEX idx_play_reviews_reviewee
    ON play_reviews(reviewee_user_id, created_at DESC, id DESC);

-- Once-only marker for the post-game "rate your co-players" notification.
-- The prompt scheduler INSERT OR IGNOREs a row per (play, user) BEFORE
-- notifying; a rescan that hits an existing row sends nothing (at-most-once).
CREATE TABLE play_review_prompts (
    play_id TEXT NOT NULL REFERENCES plays(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sent_at TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),

    PRIMARY KEY (play_id, user_id)
);

-- +goose Down

DROP TABLE IF EXISTS play_review_prompts;
DROP INDEX IF EXISTS idx_play_reviews_reviewee;
DROP TABLE IF EXISTS play_reviews;
