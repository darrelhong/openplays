-- name: InsertPlay :one
INSERT INTO plays (
    listing_type, sport, game_type, host_name,
    starts_at, ends_at, timezone,
    venue, venue_norm, level_min, level_max,
    fee, currency, max_players, slots_left, courts,
    contacts, gender_pref, meta,
    source, source_sender_username, source_raw_message, source_message_time
) VALUES (
    ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?, ?
)
RETURNING *;

-- name: GetUpcomingPlays :many
SELECT * FROM plays
WHERE starts_at > CURRENT_TIMESTAMP
  AND listing_type = 'play'
ORDER BY starts_at ASC;
