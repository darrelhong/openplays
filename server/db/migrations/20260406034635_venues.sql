-- +goose Up
CREATE TABLE venues (
    id INTEGER PRIMARY KEY,
    postal_code TEXT UNIQUE, -- nullable for generic locations (e.g. "Simei")
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    latitude REAL NOT NULL,
    longitude REAL NOT NULL,
    source TEXT NOT NULL, -- e.g 'onemap' | 'google' | 'manual'
    search_term TEXT -- the raw query used to find this venue (for auditing)
);

CREATE TABLE venue_aliases (
    alias TEXT PRIMARY KEY,
    venue_id INTEGER NOT NULL REFERENCES venues(id)
);

-- +goose Down
DROP TABLE IF EXISTS venue_aliases;
DROP TABLE IF EXISTS venues;
