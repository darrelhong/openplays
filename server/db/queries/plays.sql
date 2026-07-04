-- name: UpsertPlay :one
INSERT INTO plays (
    id, listing_type, sport, game_type, host_name,
    starts_at, ends_at, timezone,
    venue, venue_id,
    level_min, level_max, level_min_ord, level_max_ord,
    fee, currency, max_players, slots_left, courts,
    contacts, gender_pref, meta,
    source, source_sender_username, source_sender_name, source_raw_message, source_message_time,
    source_message_id, source_group
) VALUES (
    ?, ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?
)
ON CONFLICT(host_name, starts_at, sport, COALESCE(venue_id, 0)) DO UPDATE SET
    listing_type          = excluded.listing_type,
    game_type             = excluded.game_type,
    ends_at               = excluded.ends_at,
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

-- name: CreatePlay :one
INSERT INTO plays (
    id, listing_type, sport, game_type, host_name, name, description,
    starts_at, ends_at, timezone,
    venue, venue_id,
    level_min, level_max, level_min_ord, level_max_ord,
    fee, currency, max_players, slots_left, courts,
    contacts, gender_pref, meta,
    source, created_by, visibility, require_waitlist
) VALUES (
    ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?,
    ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?, ?,
    ?, ?, ?,
    'user', ?, COALESCE(NULLIF(sqlc.arg('visibility'), ''), 'public'), sqlc.arg('require_waitlist')
)
RETURNING *;

-- name: GetUpcomingPlays :many
SELECT * FROM plays
WHERE ends_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND cancelled_at IS NULL
  AND visibility = 'public'
  AND listing_type = 'play'
ORDER BY starts_at ASC;

-- name: ListUpcomingPlays :many
-- Paginated upcoming listings with optional filters and venue data.
-- Includes games still in progress (ends_at > now) not just future games.
-- Forward-only cursor pagination using composite (starts_at, id) cursor
-- to match the sort order. Both cursor params must be provided together.
SELECT
    p.id, p.created_at, p.updated_at,
    p.listing_type, p.sport, p.game_type, p.host_name, p.name, p.description, p.visibility, p.require_waitlist,
    p.starts_at, p.ends_at, p.timezone,
    p.venue, p.venue_id, p.created_by, p.cancelled_at,
    p.level_min, p.level_max, p.level_min_ord, p.level_max_ord,
    p.fee, p.currency, p.max_players, p.slots_left, p.courts,
    p.contacts, p.gender_pref, p.meta,
    p.source, p.source_sender_username, p.source_message_id, p.source_group,
    COALESCE(v.name, NULLIF(p.venue, ''), 'No venue') AS venue_name, v.postal_code AS venue_postal_code,
    v.latitude AS venue_latitude, v.longitude AS venue_longitude, v.google_place_id AS venue_google_place_id,
    u.display_name AS creator_display_name, u.username AS creator_username, u.photo_url AS creator_photo_url
FROM plays p
LEFT JOIN venues v ON v.id = p.venue_id
LEFT JOIN users u ON u.id = p.created_by
WHERE p.ends_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND p.cancelled_at IS NULL
  AND p.visibility = 'public'
  AND (sqlc.narg('starts_after') IS NULL OR p.starts_at >= sqlc.narg('starts_after'))
  AND (sqlc.narg('starts_before') IS NULL OR p.starts_at < sqlc.narg('starts_before'))
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
WHERE p.ends_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND p.cancelled_at IS NULL
  AND p.visibility = 'public'
  AND (sqlc.narg('starts_after') IS NULL OR p.starts_at >= sqlc.narg('starts_after'))
  AND (sqlc.narg('starts_before') IS NULL OR p.starts_at < sqlc.narg('starts_before'))
  AND (sqlc.narg('listing_type') IS NULL OR p.listing_type = sqlc.narg('listing_type'))
  AND (sqlc.narg('sport') IS NULL OR p.sport = sqlc.narg('sport'))
  AND (sqlc.narg('venue_id') IS NULL OR p.venue_id = sqlc.narg('venue_id'))
  AND (sqlc.narg('filter_level_min_ord') IS NULL OR (p.level_max_ord IS NULL OR p.level_max_ord >= sqlc.narg('filter_level_min_ord')))
  AND (sqlc.narg('filter_level_max_ord') IS NULL OR (p.level_min_ord IS NULL OR p.level_min_ord <= sqlc.narg('filter_level_max_ord')));

-- name: ListMyUpcomingPlays :many
-- Paginated upcoming listings where the current user is a host or participant.
-- Host relationship comes from play_hosts; created_by is a transitional fallback.
-- TODO: Audit remaining created_by usage and drop plays.created_by if play_hosts fully replaces it.
SELECT
    p.id, p.created_at, p.updated_at,
    p.listing_type, p.sport, p.game_type, p.host_name, p.name, p.description, p.visibility, p.require_waitlist,
    p.starts_at, p.ends_at, p.timezone,
    p.venue, p.venue_id, p.created_by, p.cancelled_at,
    p.level_min, p.level_max, p.level_min_ord, p.level_max_ord,
    p.fee, p.currency, p.max_players, p.slots_left, p.courts,
    p.contacts, p.gender_pref, p.meta,
    p.source, p.source_sender_username, p.source_message_id, p.source_group,
    COALESCE(v.name, NULLIF(p.venue, ''), 'No venue') AS venue_name, v.postal_code AS venue_postal_code,
    v.latitude AS venue_latitude, v.longitude AS venue_longitude, v.google_place_id AS venue_google_place_id,
    u.display_name AS creator_display_name, u.username AS creator_username, u.photo_url AS creator_photo_url,
    CAST(CASE
        WHEN EXISTS (
            SELECT 1
            FROM play_hosts ph
            WHERE ph.play_id = p.id AND ph.user_id = sqlc.arg('user_id')
        ) OR p.created_by = sqlc.arg('user_id') THEN 'creator'
        WHEN pp.status = 'confirmed' THEN 'confirmed'
        WHEN pp.status = 'added' THEN 'added'
        WHEN pp.status = 'waitlisted' THEN 'waitlisted'
        ELSE pp.status
    END AS TEXT) AS viewer_state
FROM plays p
LEFT JOIN play_participants pp ON pp.play_id = p.id AND pp.user_id = sqlc.arg('user_id')
LEFT JOIN venues v ON v.id = p.venue_id
LEFT JOIN users u ON u.id = p.created_by
WHERE p.ends_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND p.cancelled_at IS NULL
  AND (
    EXISTS (
        SELECT 1
        FROM play_hosts ph
        WHERE ph.play_id = p.id AND ph.user_id = sqlc.arg('user_id')
    )
    OR p.created_by = sqlc.arg('user_id')
    OR pp.id IS NOT NULL
  )
  AND (sqlc.narg('cursor_starts_at') IS NULL
    OR p.starts_at > sqlc.narg('cursor_starts_at')
    OR (p.starts_at = sqlc.narg('cursor_starts_at') AND p.id > sqlc.narg('cursor_id')))
ORDER BY p.starts_at ASC, p.id ASC
LIMIT sqlc.arg('page_size');

-- name: CountMyUpcomingPlays :one
-- Total count of upcoming listings where the current user is a host or participant.
SELECT COUNT(*) FROM plays p
LEFT JOIN play_participants pp ON pp.play_id = p.id AND pp.user_id = sqlc.arg('user_id')
WHERE p.ends_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND p.cancelled_at IS NULL
  AND (
    EXISTS (
        SELECT 1
        FROM play_hosts ph
        WHERE ph.play_id = p.id AND ph.user_id = sqlc.arg('user_id')
    )
    OR p.created_by = sqlc.arg('user_id')
    OR pp.id IS NOT NULL
  );

-- name: ListUpcomingPlaysByDistance :many
-- Paginated upcoming listings sorted by Haversine distance from a reference point.
-- Only includes plays with a resolved venue (INNER JOIN).
-- Forward-only cursor pagination using composite (distance_km, id).
SELECT
    p.id, p.created_at, p.updated_at,
    p.listing_type, p.sport, p.game_type, p.host_name, p.name, p.description, p.visibility, p.require_waitlist,
    p.starts_at, p.ends_at, p.timezone,
    p.venue, p.venue_id, p.created_by, p.cancelled_at,
    p.level_min, p.level_max, p.level_min_ord, p.level_max_ord,
    p.fee, p.currency, p.max_players, p.slots_left, p.courts,
    p.contacts, p.gender_pref, p.meta,
    p.source, p.source_sender_username, p.source_message_id, p.source_group,
    COALESCE(v.name, NULLIF(p.venue, ''), 'No venue') AS venue_name, v.postal_code AS venue_postal_code,
    v.latitude AS venue_latitude, v.longitude AS venue_longitude, v.google_place_id AS venue_google_place_id,
    u.display_name AS creator_display_name, u.username AS creator_username, u.photo_url AS creator_photo_url,
    CAST(2 * 6371 * asin(sqrt(
        pow(sin((radians(v.latitude) - radians(sqlc.arg('ref_lat'))) / 2), 2) +
        cos(radians(sqlc.arg('ref_lat'))) * cos(radians(v.latitude)) *
        pow(sin((radians(v.longitude) - radians(sqlc.arg('ref_lng'))) / 2), 2)
    )) AS REAL) AS distance_km
FROM plays p
INNER JOIN venues v ON v.id = p.venue_id
LEFT JOIN users u ON u.id = p.created_by
WHERE p.ends_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND p.cancelled_at IS NULL
  AND p.visibility = 'public'
  AND (sqlc.narg('starts_after') IS NULL OR p.starts_at >= sqlc.narg('starts_after'))
  AND (sqlc.narg('starts_before') IS NULL OR p.starts_at < sqlc.narg('starts_before'))
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
WHERE p.ends_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND p.cancelled_at IS NULL
  AND p.visibility = 'public'
  AND (sqlc.narg('starts_after') IS NULL OR p.starts_at >= sqlc.narg('starts_after'))
  AND (sqlc.narg('starts_before') IS NULL OR p.starts_at < sqlc.narg('starts_before'))
  AND (sqlc.narg('listing_type') IS NULL OR p.listing_type = sqlc.narg('listing_type'))
  AND (sqlc.narg('sport') IS NULL OR p.sport = sqlc.narg('sport'))
  AND (sqlc.narg('venue_id') IS NULL OR p.venue_id = sqlc.narg('venue_id'))
  AND (sqlc.narg('filter_level_min_ord') IS NULL OR (p.level_max_ord IS NULL OR p.level_max_ord >= sqlc.narg('filter_level_min_ord')))
  AND (sqlc.narg('filter_level_max_ord') IS NULL OR (p.level_min_ord IS NULL OR p.level_min_ord <= sqlc.narg('filter_level_max_ord')));

-- name: GetPlayByID :one
SELECT
    p.id, p.created_at, p.updated_at,
    p.listing_type, p.sport, p.game_type, p.host_name, p.name, p.description, p.visibility, p.require_waitlist,
    p.starts_at, p.ends_at, p.timezone,
    p.venue, p.venue_id, p.created_by, p.cancelled_at, p.cancelled_by,
    p.level_min, p.level_max, p.level_min_ord, p.level_max_ord,
    p.fee, p.currency, p.max_players, p.slots_left, p.courts,
    p.contacts, p.gender_pref, p.meta,
    p.source, p.source_sender_username, p.source_message_id, p.source_group,
    COALESCE(v.name, NULLIF(p.venue, ''), 'No venue') AS venue_name, v.postal_code AS venue_postal_code,
    v.latitude AS venue_latitude, v.longitude AS venue_longitude, v.google_place_id AS venue_google_place_id,
    u.display_name AS creator_display_name, u.username AS creator_username, u.photo_url AS creator_photo_url
FROM plays p
LEFT JOIN venues v ON v.id = p.venue_id
LEFT JOIN users u ON u.id = p.created_by
WHERE p.id = ?;

-- name: UpdateUserCreatedPlay :one
UPDATE plays
SET
    name = sqlc.arg('name'),
    description = sqlc.arg('description'),
    visibility = COALESCE(NULLIF(sqlc.arg('visibility'), ''), visibility),
    require_waitlist = sqlc.arg('require_waitlist'),
    game_type = sqlc.arg('game_type'),
    starts_at = sqlc.arg('starts_at'),
    ends_at = sqlc.arg('ends_at'),
    timezone = sqlc.arg('timezone'),
    level_min = sqlc.arg('level_min'),
    level_max = sqlc.arg('level_max'),
    level_min_ord = sqlc.arg('level_min_ord'),
    level_max_ord = sqlc.arg('level_max_ord'),
    fee = sqlc.arg('fee'),
    max_players = sqlc.arg('max_players'),
    slots_left = CASE
        WHEN sqlc.arg('max_players') IS NULL THEN NULL
        ELSE max(sqlc.arg('max_players') - (
            SELECT COUNT(*)
            FROM play_participants pp
            WHERE pp.play_id = plays.id
              AND pp.status IN ('confirmed', 'added')
        ), 0)
    END,
    courts = sqlc.arg('courts'),
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE plays.id = sqlc.arg('id')
  AND plays.created_by IS NOT NULL
RETURNING *;

-- name: CancelUserCreatedPlay :one
UPDATE plays
SET
    cancelled_at = COALESCE(cancelled_at, strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')),
    cancelled_by = COALESCE(cancelled_by, sqlc.arg('cancelled_by')),
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE id = sqlc.arg('id')
  AND created_by IS NOT NULL
RETURNING *;

-- name: UpdatePlaySlotsLeft :exec
UPDATE plays
SET
    slots_left = CASE
        WHEN max_players IS NULL THEN NULL
        ELSE max(max_players - (
            SELECT COUNT(*)
            FROM play_participants pp
            WHERE pp.play_id = plays.id
              AND pp.status IN ('confirmed', 'added')
        ), 0)
    END,
    updated_at = strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
WHERE plays.id = ?;
