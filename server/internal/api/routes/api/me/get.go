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

// RegisterGet registers GET /me.
func RegisterGet(api huma.API) {
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
		return &GetOutput{Body: *user}, nil
	})
}
