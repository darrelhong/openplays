-- +goose Up

-- plays stores the parsed, canonical play records.
-- Times are stored as UTC for cross-timezone comparison and dedup.
-- Timezone is stored alongside so local display time can be reconstructed.
-- A valid play must have time and venue.
-- sqlc overrides are used to into typed structs defined in internal/model/play.go
CREATE TABLE plays (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    listing_type    TEXT NOT NULL, -- distinguishes between different kinds of listings
    sport           TEXT NOT NULL, -- sport type
    game_type       TEXT, -- game type
    host_name       TEXT NOT NULL,

    starts_at       TIMESTAMP NOT NULL,
    ends_at         TIMESTAMP NOT NULL,
    timezone        TEXT NOT NULL,

    venue           TEXT NOT NULL,
    venue_norm      TEXT NOT NULL,

    level_min       TEXT, -- sport-specific code: "HB", "LI" (badminton), "3.5", "4.0" (tennis)
    level_max       TEXT, -- sport-specific code
    level_min_ord   INTEGER, -- numeric ordinal for filtering/sorting (badminton: LB=10,MB=20,HB=30,LI=40,MI=50,HI=60,A=70; tennis: use NTRP x10)
    level_max_ord   INTEGER, -- numeric ordinal for filtering/sorting

    fee             INTEGER,  -- cents
    currency        TEXT NOT NULL,

    max_players     INTEGER,
    slots_left      INTEGER,
    courts          INTEGER,

    contacts        TEXT,  -- JSON array
    gender_pref     TEXT,

    meta            TEXT,  -- JSON object

    source                 TEXT DEFAULT 'telegram',
    source_sender_username TEXT,
    source_raw_message     TEXT,
    source_message_time    TIMESTAMP
);

-- Primary query: upcoming plays filtered by sport
CREATE INDEX idx_plays_upcoming ON plays(sport, starts_at);

-- Dedup: prevent duplicate plays from the same host at the same time/venue
CREATE UNIQUE INDEX idx_plays_dedup ON plays(host_name, starts_at, venue);

-- +goose Down
DROP TABLE IF EXISTS plays;
