package users

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/pagination"
	"openplays/server/internal/db"
	"openplays/server/internal/sqliteutils"
	"openplays/server/internal/usernames"
)

type FollowListInput struct {
	Username string `path:"username" doc:"Username"`
	Cursor   string `query:"cursor" doc:"Opaque cursor from previous page"`
	Limit    int64  `query:"limit" default:"20" minimum:"1" maximum:"50" doc:"Number of users per page"`
}

type FollowListOutput struct {
	Body pagination.Page[UserSummary]
}

type FollowListStore interface {
	ProfileStore
	ListUserFollowers(ctx context.Context, arg db.ListUserFollowersParams) ([]db.ListUserFollowersRow, error)
	CountUserFollowers(ctx context.Context, arg db.CountUserFollowersParams) (int64, error)
	ListUserFollowing(ctx context.Context, arg db.ListUserFollowingParams) ([]db.ListUserFollowingRow, error)
	CountUserFollowing(ctx context.Context, arg db.CountUserFollowingParams) (int64, error)
}

func RegisterFollowLists(api huma.API, store FollowListStore) {
	registerFollowList(api, store, false)
	registerFollowList(api, store, true)
}

func registerFollowList(api huma.API, store FollowListStore, following bool) {
	path := "/{username}/followers"
	operationID := "list-user-followers"
	summary := "List a user's followers"
	if following {
		path = "/{username}/following"
		operationID = "list-user-following"
		summary = "List users a person follows"
	}
	huma.Register(api, huma.Operation{
		OperationID: operationID,
		Summary:     summary,
		Description: "Returns a paginated list visible to the profile owner or an accepted follower.",
		Method:      http.MethodGet,
		Path:        path,
		Tags:        []string{"Users"},
	}, func(ctx context.Context, input *FollowListInput) (*FollowListOutput, error) {
		viewer := authmw.UserFromContext(ctx)
		if viewer == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		username, err := usernames.Normalize(input.Username)
		if err != nil {
			return nil, huma.Error404NotFound("user not found")
		}
		profile, err := store.GetActiveUserProfileByUsername(ctx, &username)
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("user not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get user")
		}
		if viewer.ID != profile.ID {
			follow, followErr := store.GetUserFollow(ctx, db.GetUserFollowParams{
				FollowerID: viewer.ID,
				FolloweeID: profile.ID,
			})
			if followErr != nil && followErr != sql.ErrNoRows {
				return nil, huma.Error500InternalServerError("failed to get follow state")
			}
			if followErr == sql.ErrNoRows || follow.Status != "accepted" {
				return nil, huma.Error403Forbidden("forbidden")
			}
		}

		var cursorCreatedAt interface{}
		var cursorUserID *string
		if input.Cursor != "" {
			createdAt, userID, ok := decodeFollowListCursor(input.Cursor)
			if !ok {
				return nil, huma.Error422UnprocessableEntity("invalid cursor")
			}
			cursorCreatedAt = createdAt.UTC().Format(sqliteutils.DateTimeFormat)
			cursorUserID = &userID
		}

		out := &FollowListOutput{}
		if following {
			rows, err := store.ListUserFollowing(ctx, db.ListUserFollowingParams{
				ProfileUserID: profile.ID, ViewerID: viewer.ID,
				CursorCreatedAt: cursorCreatedAt, CursorUserID: cursorUserID,
				PageSize: input.Limit + 1,
			})
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to list following")
			}
			total, err := store.CountUserFollowing(ctx, db.CountUserFollowingParams{
				ProfileUserID: profile.ID, ViewerID: viewer.ID,
			})
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to count following")
			}
			page := pagination.Paginate(rows, input.Limit, total, func(row db.ListUserFollowingRow) string {
				return encodeFollowListCursor(row.CreatedAt, row.UserID)
			})
			out.Body = pagination.Page[UserSummary]{
				Items: mapFollowingSummaries(page.Items), Total: page.Total,
				NextCursor: page.NextCursor, HasMore: page.HasMore,
			}
			return out, nil
		}

		rows, err := store.ListUserFollowers(ctx, db.ListUserFollowersParams{
			ProfileUserID: profile.ID, ViewerID: viewer.ID,
			CursorCreatedAt: cursorCreatedAt, CursorUserID: cursorUserID,
			PageSize: input.Limit + 1,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list followers")
		}
		total, err := store.CountUserFollowers(ctx, db.CountUserFollowersParams{
			ProfileUserID: profile.ID, ViewerID: viewer.ID,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count followers")
		}
		page := pagination.Paginate(rows, input.Limit, total, func(row db.ListUserFollowersRow) string {
			return encodeFollowListCursor(row.CreatedAt, row.UserID)
		})
		out.Body = pagination.Page[UserSummary]{
			Items: mapFollowerSummaries(page.Items), Total: page.Total,
			NextCursor: page.NextCursor, HasMore: page.HasMore,
		}
		return out, nil
	})
}

func mapFollowerSummaries(rows []db.ListUserFollowersRow) []UserSummary {
	items := make([]UserSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, UserSummary{ID: row.UserID, DisplayName: row.DisplayName, Username: row.Username, PhotoURL: row.PhotoUrl})
	}
	return items
}

func mapFollowingSummaries(rows []db.ListUserFollowingRow) []UserSummary {
	items := make([]UserSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, UserSummary{ID: row.UserID, DisplayName: row.DisplayName, Username: row.Username, PhotoURL: row.PhotoUrl})
	}
	return items
}

func encodeFollowListCursor(createdAt time.Time, userID string) string {
	return fmt.Sprintf("%s,%s", createdAt.UTC().Format(time.RFC3339), userID)
}

func decodeFollowListCursor(cursor string) (createdAt time.Time, userID string, ok bool) {
	parts := strings.SplitN(cursor, ",", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return time.Time{}, "", false
	}
	createdAt, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return time.Time{}, "", false
	}
	return createdAt, parts[1], true
}
