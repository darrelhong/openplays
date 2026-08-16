package users

import (
	"context"

	"openplays/server/internal/db"
)

// SearchStore is the DB boundary for user search operations.
type SearchStore interface {
	SearchActiveUsers(ctx context.Context, arg db.SearchActiveUsersParams) ([]db.SearchActiveUsersRow, error)
}

// ProfileStore is the DB boundary for public user profile operations.
type ProfileStore interface {
	GetActiveUserProfileByUsername(ctx context.Context, username *string) (db.GetActiveUserProfileByUsernameRow, error)
	CountRosteredPlaysByUser(ctx context.Context, userID string) (int64, error)
	CountRosteredPlaysByUserAndSport(ctx context.Context, userID string) ([]db.CountRosteredPlaysByUserAndSportRow, error)
	GetUserRatingAggregate(ctx context.Context, revieweeUserID string) (db.GetUserRatingAggregateRow, error)
	ListUserRatingDistribution(ctx context.Context, revieweeUserID string) ([]db.ListUserRatingDistributionRow, error)
	ListUserPropCounts(ctx context.Context, revieweeUserID string) ([]db.ListUserPropCountsRow, error)
	CountAcceptedFollowers(ctx context.Context, followeeID string) (int64, error)
	CountAcceptedFollowing(ctx context.Context, followerID string) (int64, error)
	GetUserFollow(ctx context.Context, arg db.GetUserFollowParams) (db.UserFollow, error)
}
