-- name: GetFavouriteablePlayID :one
SELECT id
FROM plays
WHERE id = ?
  AND ends_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND cancelled_at IS NULL;

-- name: FavouritePlay :exec
INSERT OR IGNORE INTO play_favourites (user_id, play_id)
VALUES (?, ?);

-- name: UnfavouritePlay :exec
DELETE FROM play_favourites
WHERE user_id = ? AND play_id = ?;

-- name: ListFavouritedPlayIDsByUserAndPlays :many
SELECT play_id
FROM play_favourites
WHERE user_id = ? AND play_id IN (sqlc.slice('play_ids'));

-- name: ListFavouriteUpcomingPlays :many
-- Paginated upcoming listings favourited by the current user.
SELECT
    p.id, p.created_at, p.updated_at,
    p.listing_type, p.sport, p.game_type, p.host_name,
    p.starts_at, p.ends_at, p.timezone,
    p.venue, p.venue_id, p.created_by, p.cancelled_at,
    p.level_min, p.level_max, p.level_min_ord, p.level_max_ord,
    p.fee, p.currency, p.max_players, p.slots_left, p.courts,
    p.contacts, p.gender_pref, p.meta,
    p.source, p.source_sender_username, p.source_message_id, p.source_group,
    COALESCE(v.name, NULLIF(p.venue, ''), 'No venue') AS venue_name, v.postal_code AS venue_postal_code,
    v.latitude AS venue_latitude, v.longitude AS venue_longitude,
    u.display_name AS creator_display_name, u.username AS creator_username, u.photo_url AS creator_photo_url
FROM play_favourites pf
INNER JOIN plays p ON p.id = pf.play_id
LEFT JOIN venues v ON v.id = p.venue_id
LEFT JOIN users u ON u.id = p.created_by
WHERE pf.user_id = sqlc.arg('user_id')
  AND p.ends_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND p.cancelled_at IS NULL
  AND (sqlc.narg('cursor_starts_at') IS NULL
    OR p.starts_at > sqlc.narg('cursor_starts_at')
    OR (p.starts_at = sqlc.narg('cursor_starts_at') AND p.id > sqlc.narg('cursor_id')))
ORDER BY p.starts_at ASC, p.id ASC
LIMIT sqlc.arg('page_size');

-- name: CountFavouriteUpcomingPlays :one
SELECT COUNT(*)
FROM play_favourites pf
INNER JOIN plays p ON p.id = pf.play_id
WHERE pf.user_id = sqlc.arg('user_id')
  AND p.ends_at > strftime('%Y-%m-%d %H:%M:%S+00:00', 'now')
  AND p.cancelled_at IS NULL;
