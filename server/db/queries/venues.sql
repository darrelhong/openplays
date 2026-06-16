-- name: UpsertVenue :one
-- For venues with a postal code, upsert on postal_code.
-- For venues without (generic locations), always insert a new row.
INSERT INTO venues (postal_code, name, address, latitude, longitude, source, search_term, google_place_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(postal_code) DO UPDATE SET
    name        = excluded.name,
    address     = excluded.address,
    latitude    = excluded.latitude,
    longitude   = excluded.longitude,
    source      = excluded.source,
    google_place_id = COALESCE(excluded.google_place_id, venues.google_place_id)
RETURNING *;

-- name: UpsertVenueByGooglePlaceID :one
-- Catch both google_place_id and postal_code uniqueness conflicts so Google
-- resolutions can reuse an existing postal-code venue instead of failing.
INSERT INTO venues (google_place_id, postal_code, name, address, latitude, longitude, source, search_term)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT DO UPDATE SET
    google_place_id = excluded.google_place_id,
    name            = excluded.name,
    address         = excluded.address,
    latitude        = excluded.latitude,
    longitude       = excluded.longitude,
    source          = excluded.source,
    search_term     = excluded.search_term
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

-- name: ListVenuesMissingGooglePlaceIDWithName :many
SELECT * FROM venues
WHERE google_place_id IS NULL
  AND trim(name) <> ''
ORDER BY id;

-- name: UpdateVenueGooglePlaceID :one
UPDATE venues
SET google_place_id = ?
WHERE id = ?
RETURNING *;

-- name: ListVenueNames :many
SELECT id, name FROM venues;

-- name: ListAliases :many
SELECT va.alias, va.venue_id, v.name AS venue_name
FROM venue_aliases va
JOIN venues v ON v.id = va.venue_id
ORDER BY va.alias;

-- name: GetVenueByID :one
SELECT * FROM venues WHERE id = ?;

-- name: ListVenuesWithPostalCode :many
SELECT id, name, address, postal_code, latitude, longitude, google_place_id
FROM venues
WHERE postal_code IS NOT NULL
ORDER BY name;

-- name: SearchVenues :many
SELECT DISTINCT
    v.id, v.name, v.address, v.postal_code, v.latitude, v.longitude, v.google_place_id
FROM venues v
LEFT JOIN venue_aliases va ON va.venue_id = v.id
WHERE lower(v.name) LIKE '%' || lower(sqlc.arg('query')) || '%'
   OR lower(v.address) LIKE '%' || lower(sqlc.arg('query')) || '%'
   OR lower(COALESCE(v.postal_code, '')) LIKE '%' || lower(sqlc.arg('query')) || '%'
   OR lower(COALESCE(va.alias, '')) LIKE '%' || lower(sqlc.arg('query')) || '%'
ORDER BY
    v.name
LIMIT sqlc.arg('limit');

-- name: GetVenueByGooglePlaceID :one
SELECT * FROM venues WHERE google_place_id = ?;
