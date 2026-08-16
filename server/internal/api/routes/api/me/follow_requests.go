package me

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/pagination"
	"openplays/server/internal/api/routes/api/users"
	"openplays/server/internal/db"
	"openplays/server/internal/notifications"
	"openplays/server/internal/sqliteutils"
)

type FollowRequestsInput struct {
	Cursor string `query:"cursor" doc:"Opaque cursor from previous page"`
	Limit  int64  `query:"limit" default:"10" minimum:"1" maximum:"50" doc:"Number of requests per page"`
}

type FollowRequestUserInput struct {
	UserID string `path:"user_id" doc:"Follower user ID"`
}

type FollowRequestsOutput struct {
	Body pagination.Page[users.UserSummary]
}

type FollowRequestsStore interface {
	ListPendingFollowRequests(ctx context.Context, arg db.ListPendingFollowRequestsParams) ([]db.ListPendingFollowRequestsRow, error)
	CountPendingFollowRequests(ctx context.Context, followeeID string) (int64, error)
	AcceptUserFollow(ctx context.Context, arg db.AcceptUserFollowParams) (int64, error)
	DeletePendingUserFollow(ctx context.Context, arg db.DeletePendingUserFollowParams) (int64, error)
}

func RegisterFollowRequests(api huma.API, store FollowRequestsStore, notifier notifications.Sender) {
	huma.Register(api, huma.Operation{
		OperationID: "list-follow-requests",
		Summary:     "List follow requests",
		Description: "Returns pending requests to follow the authenticated user, newest first.",
		Method:      http.MethodGet,
		Path:        "/follow-requests",
		Tags:        []string{"Me"},
	}, func(ctx context.Context, input *FollowRequestsInput) (*FollowRequestsOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		var cursorCreatedAt interface{}
		var cursorFollowerID *string
		if input.Cursor != "" {
			createdAt, followerID, ok := decodeFollowRequestCursor(input.Cursor)
			if !ok {
				return nil, huma.Error422UnprocessableEntity("invalid cursor")
			}
			cursorCreatedAt = createdAt.UTC().Format(sqliteutils.DateTimeFormat)
			cursorFollowerID = &followerID
		}

		rows, err := store.ListPendingFollowRequests(ctx, db.ListPendingFollowRequestsParams{
			FolloweeID:       user.ID,
			CursorCreatedAt:  cursorCreatedAt,
			CursorFollowerID: cursorFollowerID,
			PageSize:         input.Limit + 1,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list follow requests")
		}
		total, err := store.CountPendingFollowRequests(ctx, user.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count follow requests")
		}

		page := pagination.Paginate(rows, input.Limit, total, func(row db.ListPendingFollowRequestsRow) string {
			return encodeFollowRequestCursor(row.CreatedAt, row.FollowerID)
		})
		out := &FollowRequestsOutput{}
		out.Body = pagination.Page[users.UserSummary]{
			Total:      page.Total,
			NextCursor: page.NextCursor,
			HasMore:    page.HasMore,
			Items:      make([]users.UserSummary, 0, len(page.Items)),
		}
		for _, row := range page.Items {
			out.Body.Items = append(out.Body.Items, users.UserSummary{
				ID:          row.FollowerID,
				DisplayName: row.DisplayName,
				Username:    row.Username,
				PhotoURL:    row.PhotoUrl,
			})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "accept-follow-request",
		Summary:     "Accept a follow request",
		Description: "Accepts a pending request to follow the authenticated user.",
		Method:      http.MethodPost,
		Path:        "/follow-requests/{user_id}/accept",
		Tags:        []string{"Me"},
	}, func(ctx context.Context, input *FollowRequestUserInput) (*struct{}, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		rows, err := store.AcceptUserFollow(ctx, db.AcceptUserFollowParams{
			FollowerID: input.UserID,
			FolloweeID: user.ID,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to accept follow request")
		}
		if rows == 0 {
			return nil, huma.Error404NotFound("follow request not found")
		}
		followeeUsername := ""
		if user.Username != nil {
			followeeUsername = *user.Username
		}
		_ = notifications.NotifyFollowAccepted(ctx, notifier, input.UserID, user.DisplayName, followeeUsername)
		return &struct{}{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "decline-follow-request",
		Summary:     "Decline a follow request",
		Description: "Declines a pending request to follow the authenticated user.",
		Method:      http.MethodPost,
		Path:        "/follow-requests/{user_id}/decline",
		Tags:        []string{"Me"},
	}, func(ctx context.Context, input *FollowRequestUserInput) (*struct{}, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if _, err := store.DeletePendingUserFollow(ctx, db.DeletePendingUserFollowParams{
			FollowerID: input.UserID,
			FolloweeID: user.ID,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to decline follow request")
		}
		return &struct{}{}, nil
	})
}

func encodeFollowRequestCursor(createdAt time.Time, followerID string) string {
	return fmt.Sprintf("%s,%s", createdAt.UTC().Format(time.RFC3339), followerID)
}

func decodeFollowRequestCursor(cursor string) (createdAt time.Time, followerID string, ok bool) {
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
