-- name: UpsertPlay :one
INSERT INTO plays (
    listing_type, sport, game_type, host_name,
    starts_at, ends_at, timezone,
    venue, venue_id,
    level_min, level_max, level_min_ord, level_max_ord,
    fee, currency, max_players, slots_left, courts,
    contacts, gender_pref, meta,
    source, source_sender_username, source_sender_name, source_raw_message, source_message_time,
    source_message_id, source_group
) VALUES (
    ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?
)
ON CONFLICT(host_name, starts_at, ends_at, sport, level_min, level_max, venue_id) DO UPDATE SET
    listing_type          = excluded.listing_type,
    game_type             = excluded.game_type,
    venue_id              = excluded.venue_id,
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
    source_sender_name    = excluded.source_sender_name,
    source_raw_message    = excluded.source_raw_message,
    source_message_time   = excluded.source_message_time,
    source_message_id     = excluded.source_message_id,
    source_group          = excluded.source_group,
    updated_at            = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
RETURNING *;

-- name: GetUpcomingPlays :many
SELECT * FROM plays
WHERE starts_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND listing_type = 'play'
ORDER BY starts_at ASC;

-- name: ListUpcomingPlays :many
-- Paginated upcoming listings with optional filters and venue data.
-- Forward-only cursor pagination using composite (starts_at, id) cursor
-- to match the sort order. Both cursor params must be provided together.
SELECT
    p.id, p.created_at, p.updated_at,
    p.listing_type, p.sport, p.game_type, p.host_name,
    p.starts_at, p.ends_at, p.timezone,
    p.venue, p.venue_id,
    p.level_min, p.level_max, p.level_min_ord, p.level_max_ord,
    p.fee, p.currency, p.max_players, p.slots_left, p.courts,
    p.contacts, p.gender_pref, p.meta,
    p.source, p.source_sender_username, p.source_message_id, p.source_group,
    COALESCE(v.name, NULLIF(p.venue, ''), 'No venue') AS venue_name, v.postal_code AS venue_postal_code,
    v.latitude AS venue_latitude, v.longitude AS venue_longitude
FROM plays p
LEFT JOIN venues v ON v.id = p.venue_id
WHERE p.starts_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND (sqlc.narg('starts_after') IS NULL OR p.starts_at >= sqlc.narg('starts_after'))
  AND (sqlc.narg('listing_type') IS NULL OR p.listing_type = sqlc.narg('listing_type'))
  AND (sqlc.narg('sport') IS NULL OR p.sport = sqlc.narg('sport'))
  AND (sqlc.narg('venue_id') IS NULL OR p.venue_id = sqlc.narg('venue_id'))
  AND (sqlc.narg('filter_level_min_ord') IS NULL OR (p.level_max_ord IS NULL OR p.level_max_ord >= sqlc.narg('filter_level_min_ord')))
  AND (sqlc.narg('filter_level_max_ord') IS NULL OR (p.level_min_ord IS NULL OR p.level_min_ord <= sqlc.narg('filter_level_max_ord')))
  AND (sqlc.narg('cursor_starts_at') IS NULL
    OR p.starts_at > sqlc.narg('cursor_starts_at')
    OR (p.starts_at = sqlc.narg('cursor_starts_at') AND p.id > sqlc.narg('cursor_id')))
ORDER BY p.starts_at ASC, p.id ASC
LIMIT sqlc.arg('page_size');

-- name: CountUpcomingPlays :one
-- Total count of upcoming listings matching the same filters.
SELECT COUNT(*) FROM plays p
WHERE p.starts_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND (sqlc.narg('starts_after') IS NULL OR p.starts_at >= sqlc.narg('starts_after'))
  AND (sqlc.narg('listing_type') IS NULL OR p.listing_type = sqlc.narg('listing_type'))
  AND (sqlc.narg('sport') IS NULL OR p.sport = sqlc.narg('sport'))
  AND (sqlc.narg('venue_id') IS NULL OR p.venue_id = sqlc.narg('venue_id'))
  AND (sqlc.narg('filter_level_min_ord') IS NULL OR (p.level_max_ord IS NULL OR p.level_max_ord >= sqlc.narg('filter_level_min_ord')))
  AND (sqlc.narg('filter_level_max_ord') IS NULL OR (p.level_min_ord IS NULL OR p.level_min_ord <= sqlc.narg('filter_level_max_ord')));

-- name: ListUpcomingPlaysByDistance :many
-- Paginated upcoming listings sorted by Haversine distance from a reference point.
-- Only includes plays with a resolved venue (INNER JOIN).
-- Forward-only cursor pagination using composite (distance_km, id).
SELECT
    p.id, p.created_at, p.updated_at,
    p.listing_type, p.sport, p.game_type, p.host_name,
    p.starts_at, p.ends_at, p.timezone,
    p.venue, p.venue_id,
    p.level_min, p.level_max, p.level_min_ord, p.level_max_ord,
    p.fee, p.currency, p.max_players, p.slots_left, p.courts,
    p.contacts, p.gender_pref, p.meta,
    p.source, p.source_sender_username, p.source_message_id, p.source_group,
    COALESCE(v.name, NULLIF(p.venue, ''), 'No venue') AS venue_name, v.postal_code AS venue_postal_code,
    v.latitude AS venue_latitude, v.longitude AS venue_longitude,
    CAST(2 * 6371 * asin(sqrt(
        pow(sin((radians(v.latitude) - radians(sqlc.arg('ref_lat'))) / 2), 2) +
        cos(radians(sqlc.arg('ref_lat'))) * cos(radians(v.latitude)) *
        pow(sin((radians(v.longitude) - radians(sqlc.arg('ref_lng'))) / 2), 2)
    )) AS REAL) AS distance_km
FROM plays p
INNER JOIN venues v ON v.id = p.venue_id
WHERE p.starts_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND (sqlc.narg('starts_after') IS NULL OR p.starts_at >= sqlc.narg('starts_after'))
  AND (sqlc.narg('listing_type') IS NULL OR p.listing_type = sqlc.narg('listing_type'))
  AND (sqlc.narg('sport') IS NULL OR p.sport = sqlc.narg('sport'))
  AND (sqlc.narg('venue_id') IS NULL OR p.venue_id = sqlc.narg('venue_id'))
  AND (sqlc.narg('filter_level_min_ord') IS NULL OR (p.level_max_ord IS NULL OR p.level_max_ord >= sqlc.narg('filter_level_min_ord')))
  AND (sqlc.narg('filter_level_max_ord') IS NULL OR (p.level_min_ord IS NULL OR p.level_min_ord <= sqlc.narg('filter_level_max_ord')))
  AND (sqlc.narg('cursor_distance') IS NULL
    OR 2 * 6371 * asin(sqrt(
        pow(sin((radians(v.latitude) - radians(sqlc.arg('ref_lat'))) / 2), 2) +
        cos(radians(sqlc.arg('ref_lat'))) * cos(radians(v.latitude)) *
        pow(sin((radians(v.longitude) - radians(sqlc.arg('ref_lng'))) / 2), 2)
    )) > sqlc.narg('cursor_distance')
    OR (2 * 6371 * asin(sqrt(
        pow(sin((radians(v.latitude) - radians(sqlc.arg('ref_lat'))) / 2), 2) +
        cos(radians(sqlc.arg('ref_lat'))) * cos(radians(v.latitude)) *
        pow(sin((radians(v.longitude) - radians(sqlc.arg('ref_lng'))) / 2), 2)
    )) = sqlc.narg('cursor_distance') AND p.id > sqlc.narg('cursor_id')))
ORDER BY distance_km ASC, p.id ASC
LIMIT sqlc.arg('page_size');

-- name: CountUpcomingPlaysByDistance :one
-- Total count of upcoming listings with a resolved venue, matching the same filters.
SELECT COUNT(*) FROM plays p
INNER JOIN venues v ON v.id = p.venue_id
WHERE p.starts_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND (sqlc.narg('starts_after') IS NULL OR p.starts_at >= sqlc.narg('starts_after'))
  AND (sqlc.narg('listing_type') IS NULL OR p.listing_type = sqlc.narg('listing_type'))
  AND (sqlc.narg('sport') IS NULL OR p.sport = sqlc.narg('sport'))
  AND (sqlc.narg('venue_id') IS NULL OR p.venue_id = sqlc.narg('venue_id'))
  AND (sqlc.narg('filter_level_min_ord') IS NULL OR (p.level_max_ord IS NULL OR p.level_max_ord >= sqlc.narg('filter_level_min_ord')))
  AND (sqlc.narg('filter_level_max_ord') IS NULL OR (p.level_min_ord IS NULL OR p.level_min_ord <= sqlc.narg('filter_level_max_ord')));

-- name: GetPlayByID :one
SELECT
    p.id, p.created_at, p.updated_at,
    p.listing_type, p.sport, p.game_type, p.host_name,
    p.starts_at, p.ends_at, p.timezone,
    p.venue, p.venue_id,
    p.level_min, p.level_max, p.level_min_ord, p.level_max_ord,
    p.fee, p.currency, p.max_players, p.slots_left, p.courts,
    p.contacts, p.gender_pref, p.meta,
    p.source, p.source_sender_username, p.source_message_id, p.source_group,
    COALESCE(v.name, NULLIF(p.venue, ''), 'No venue') AS venue_name, v.postal_code AS venue_postal_code,
    v.latitude AS venue_latitude, v.longitude AS venue_longitude
FROM plays p
LEFT JOIN venues v ON v.id = p.venue_id
WHERE p.id = ?;
