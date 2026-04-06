-- name: UpsertVenue :one
-- For venues with a postal code, upsert on postal_code.
-- For venues without (generic locations), always insert a new row.
INSERT INTO venues (postal_code, name, address, latitude, longitude, source, search_term)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(postal_code) DO UPDATE SET
    name        = excluded.name,
    address     = excluded.address,
    latitude    = excluded.latitude,
    longitude   = excluded.longitude,
    source      = excluded.source
RETURNING *;

-- name: GetVenueByAlias :one
SELECT v.*
FROM venues v
JOIN venue_aliases va ON va.venue_id = v.id
WHERE va.alias = ?;

-- name: UpsertVenueAlias :exec
INSERT INTO venue_aliases (alias, venue_id)
VALUES (?, ?)
ON CONFLICT(alias) DO UPDATE SET
    venue_id = excluded.venue_id;

-- name: ListVenues :many
SELECT * FROM venues
ORDER BY name;

-- name: ListVenueNames :many
SELECT id, name FROM venues;

-- name: ListAliases :many
SELECT va.alias, va.venue_id, v.name AS venue_name
FROM venue_aliases va
JOIN venues v ON v.id = va.venue_id
ORDER BY va.alias;
