package me

import (
	"context"

	"openplays/server/internal/db"
)

// ProfileStore is the DB boundary for profile operations.
// Subset of *db.Queries — mock this in tests.
type ProfileStore interface {
	UpdateUserProfile(ctx context.Context, arg db.UpdateUserProfileParams) (db.User, error)
}
