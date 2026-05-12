package users

import (
	"context"

	"openplays/server/internal/db"
)

// SearchStore is the DB boundary for user search operations.
type SearchStore interface {
	SearchActiveUsers(ctx context.Context, arg db.SearchActiveUsersParams) ([]db.SearchActiveUsersRow, error)
}
