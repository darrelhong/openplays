-- name: UpsertPlay :one
INSERT INTO plays (
    listing_type, sport, game_type, host_name,
    starts_at, ends_at, timezone,
    venue, venue_norm, venue_id,
    level_min, level_max, level_min_ord, level_max_ord,
    fee, currency, max_players, slots_left, courts,
    contacts, gender_pref, meta,
    source, source_sender_username, source_raw_message, source_message_time
) VALUES (
    ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?, ?
)
ON CONFLICT(host_name, starts_at, ends_at, sport, venue_id) DO UPDATE SET
    listing_type          = excluded.listing_type,
    game_type             = excluded.game_type,
    venue                 = excluded.venue,
    venue_norm            = excluded.venue_norm,
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

-- name: ListUpcomingPlays :many
-- Paginated upcoming plays with optional filters and venue data.
-- Forward-only cursor pagination: pass last seen play ID as 'cursor'.
-- Requests page_size + 1 rows; if all are returned, there are more pages.
SELECT
    p.id, p.listing_type, p.sport, p.game_type, p.host_name,
    p.starts_at, p.ends_at, p.timezone,
    p.venue, p.venue_norm, p.venue_id,
    p.level_min, p.level_max, p.level_min_ord, p.level_max_ord,
    p.fee, p.currency, p.max_players, p.slots_left, p.courts,
    p.contacts, p.gender_pref, p.meta,
    v.name AS venue_name, v.postal_code AS venue_postal_code,
    v.latitude AS venue_latitude, v.longitude AS venue_longitude
FROM plays p
LEFT JOIN venues v ON v.id = p.venue_id
WHERE p.starts_at > CURRENT_TIMESTAMP
  AND p.listing_type = 'play'
  AND (sqlc.narg('sport') IS NULL OR p.sport = sqlc.narg('sport'))
  AND (sqlc.narg('venue_id') IS NULL OR p.venue_id = sqlc.narg('venue_id'))
  AND (sqlc.narg('cursor') IS NULL OR p.id > sqlc.narg('cursor'))
ORDER BY p.starts_at ASC, p.id ASC
LIMIT sqlc.arg('page_size');

-- name: CountUpcomingPlays :one
-- Total count of upcoming plays matching the same filters (for "showing X plays").
SELECT COUNT(*) FROM plays p
WHERE p.starts_at > CURRENT_TIMESTAMP
  AND p.listing_type = 'play'
  AND (sqlc.narg('sport') IS NULL OR p.sport = sqlc.narg('sport'))
  AND (sqlc.narg('venue_id') IS NULL OR p.venue_id = sqlc.narg('venue_id'));

-- name: GetPlayByID :one
SELECT
    p.id, p.listing_type, p.sport, p.game_type, p.host_name,
    p.starts_at, p.ends_at, p.timezone,
    p.venue, p.venue_norm, p.venue_id,
    p.level_min, p.level_max, p.level_min_ord, p.level_max_ord,
    p.fee, p.currency, p.max_players, p.slots_left, p.courts,
    p.contacts, p.gender_pref, p.meta,
    v.name AS venue_name, v.postal_code AS venue_postal_code,
    v.latitude AS venue_latitude, v.longitude AS venue_longitude
FROM plays p
LEFT JOIN venues v ON v.id = p.venue_id
WHERE p.id = ?;
