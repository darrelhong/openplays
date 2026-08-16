package me

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
)

type GetOutput struct {
	Body auth.User
}

type PendingFollowRequestsStore interface {
	CountPendingFollowRequests(ctx context.Context, followeeID string) (int64, error)
}

// RegisterGet registers GET /me.
func RegisterGet(api huma.API, stores ...PendingFollowRequestsStore) {
	huma.Register(api, huma.Operation{
		OperationID: "get-me",
		Summary:     "Get current user",
		Description: "Returns the authenticated user. Requires session cookie.",
		Method:      http.MethodGet,
		Path:        "/",
		Tags:        []string{"Me"},
	}, func(ctx context.Context, _ *struct{}) (*GetOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		out := *user
		if len(stores) > 0 {
			count, err := stores[0].CountPendingFollowRequests(ctx, user.ID)
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to count follow requests")
			}
			out.PendingFollowRequestCount = count
		}
		return &GetOutput{Body: out}, nil
	})
}
