-- name: UpsertPlay :one
INSERT INTO plays (
    listing_type, sport, game_type, host_name,
    starts_at, ends_at, timezone,
    venue, venue_norm, level_min, level_max, level_min_ord, level_max_ord,
    fee, currency, max_players, slots_left, courts,
    contacts, gender_pref, meta,
    source, source_sender_username, source_raw_message, source_message_time
) VALUES (
    ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?, ?
)
ON CONFLICT(host_name, starts_at, venue) DO UPDATE SET
    listing_type          = excluded.listing_type,
    sport                 = excluded.sport,
    game_type             = excluded.game_type,
    ends_at               = excluded.ends_at,
    level_min             = excluded.level_min,
    level_max             = excluded.level_max,
    level_min_ord         = excluded.level_min_ord,
    level_max_ord         = excluded.level_max_ord,
    fee                   = excluded.fee,
    currency              = excluded.currency,
    max_players           = excluded.max_players,
    slots_left            = excluded.slots_left,
    courts                = excluded.courts,
    contacts              = excluded.contacts,
    gender_pref           = excluded.gender_pref,
    meta                  = excluded.meta,
    source_sender_username = excluded.source_sender_username,
    source_raw_message    = excluded.source_raw_message,
    source_message_time   = excluded.source_message_time,
    updated_at            = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetUpcomingPlays :many
SELECT * FROM plays
WHERE starts_at > CURRENT_TIMESTAMP
  AND listing_type = 'play'
ORDER BY starts_at ASC;
