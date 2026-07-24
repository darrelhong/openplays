-- +goose Up

-- Telegram imports and user-created games are separate listings, so they must
-- not conflict with each other.
DROP INDEX IF EXISTS idx_plays_dedupe;

CREATE UNIQUE INDEX idx_plays_telegram_dedupe ON plays(
    host_name, starts_at, sport,
    COALESCE(venue_id, 0)
)
WHERE source = 'telegram';

-- User-created games use the creator's stable identity in place of the
-- imported host name. Other editable details do not split the same game.
CREATE UNIQUE INDEX idx_plays_user_dedupe ON plays(
    created_by, starts_at, sport,
    COALESCE(
        'id:' || CAST(venue_id AS TEXT),
        'name:' || lower(trim(venue))
    )
)
WHERE source = 'user'
  AND created_by IS NOT NULL
  AND cancelled_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_plays_user_dedupe;
DROP INDEX IF EXISTS idx_plays_telegram_dedupe;

CREATE UNIQUE INDEX idx_plays_dedupe ON plays(
    host_name, starts_at, sport,
    COALESCE(venue_id, 0)
);
