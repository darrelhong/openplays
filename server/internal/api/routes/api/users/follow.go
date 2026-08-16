package users

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
	"openplays/server/internal/notifications"
	"openplays/server/internal/usernames"
)

type FollowInput struct {
	Username string `path:"username" doc:"Username"`
}

type FollowOutput struct {
	Body struct {
		Status string `json:"status" enum:"accepted,pending"`
	}
}

type FollowStore interface {
	GetActiveUserProfileByUsername(ctx context.Context, username *string) (db.GetActiveUserProfileByUsernameRow, error)
	IsBlocked(ctx context.Context, arg db.IsBlockedParams) (bool, error)
	CreateUserFollow(ctx context.Context, arg db.CreateUserFollowParams) (db.UserFollow, error)
	GetUserFollow(ctx context.Context, arg db.GetUserFollowParams) (db.UserFollow, error)
	DeleteUserFollow(ctx context.Context, arg db.DeleteUserFollowParams) error
}

func RegisterFollow(api huma.API, store FollowStore, notifier notifications.Sender) {
	huma.Register(api, huma.Operation{
		OperationID: "follow-user",
		Summary:     "Follow a user",
		Description: "Follow an active user or request to follow them when they require approval.",
		Method:      http.MethodPut,
		Path:        "/{username}/follow",
		Tags:        []string{"Users"},
	}, func(ctx context.Context, input *FollowInput) (*FollowOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		followee, err := activeFollowee(ctx, store, input.Username)
		if err != nil {
			return nil, err
		}
		if followee.ID == user.ID {
			return nil, huma.Error422UnprocessableEntity("cannot follow yourself")
		}

		blocked, err := store.IsBlocked(ctx, db.IsBlockedParams{
			BlockerID: user.ID, BlockedID: followee.ID,
			BlockerID_2: followee.ID, BlockedID_2: user.ID,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to check user relationship")
		}
		if blocked {
			return nil, huma.Error403Forbidden("cannot follow this user")
		}

		status := "accepted"
		if followee.RestrictFollow {
			status = "pending"
		}
		follow, err := store.CreateUserFollow(ctx, db.CreateUserFollowParams{
			FollowerID: user.ID,
			FolloweeID: followee.ID,
			Status:     status,
		})
		inserted := true
		if err == sql.ErrNoRows {
			inserted = false
			follow, err = store.GetUserFollow(ctx, db.GetUserFollowParams{
				FollowerID: user.ID,
				FolloweeID: followee.ID,
			})
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to follow user")
		}

		if inserted {
			followerUsername := ""
			if user.Username != nil {
				followerUsername = *user.Username
			}
			switch follow.Status {
			case "accepted":
				_ = notifications.NotifyFollowStarted(ctx, notifier, followee.ID, user.ID, user.DisplayName, followerUsername)
			case "pending":
				_ = notifications.NotifyFollowRequested(ctx, notifier, followee.ID, user.ID, user.DisplayName)
			}
		}

		out := &FollowOutput{}
		out.Body.Status = follow.Status
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "unfollow-user",
		Summary:     "Unfollow a user",
		Description: "Remove the current user's follow or follow request.",
		Method:      http.MethodDelete,
		Path:        "/{username}/follow",
		Tags:        []string{"Users"},
	}, func(ctx context.Context, input *FollowInput) (*struct{}, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		followee, err := activeFollowee(ctx, store, input.Username)
		if err != nil {
			return nil, err
		}
		if err := store.DeleteUserFollow(ctx, db.DeleteUserFollowParams{
			FollowerID: user.ID,
			FolloweeID: followee.ID,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to unfollow user")
		}
		return &struct{}{}, nil
	})
}

func activeFollowee(ctx context.Context, store FollowStore, rawUsername string) (db.GetActiveUserProfileByUsernameRow, error) {
	username, err := usernames.Normalize(rawUsername)
	if err != nil {
		return db.GetActiveUserProfileByUsernameRow{}, huma.Error404NotFound("user not found")
	}
	followee, err := store.GetActiveUserProfileByUsername(ctx, &username)
	if err == sql.ErrNoRows {
		return db.GetActiveUserProfileByUsernameRow{}, huma.Error404NotFound("user not found")
	}
	if err != nil {
		return db.GetActiveUserProfileByUsernameRow{}, huma.Error500InternalServerError("failed to get user")
	}
	return followee, nil
}
