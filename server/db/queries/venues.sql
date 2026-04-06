-- name: UpsertVenue :one
INSERT INTO venues (postal_code, name, address, latitude, longitude, source)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(postal_code) DO UPDATE SET
    name      = excluded.name,
    address   = excluded.address,
    latitude  = excluded.latitude,
    longitude = excluded.longitude,
    source    = excluded.source
RETURNING *;

-- name: GetVenueByAlias :one
SELECT v.*
FROM venues v
JOIN venue_aliases va ON va.venue_postal_code = v.postal_code
WHERE va.alias = ?;

-- name: InsertAlias :exec
INSERT INTO venue_aliases (alias, venue_postal_code)
VALUES (?, ?)
ON CONFLICT(alias) DO NOTHING;

-- name: ListVenues :many
SELECT * FROM venues
ORDER BY name;
