package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	authpkg "openplays/server/internal/auth"
)

type MeInput struct {
	Cookie string `header:"Cookie"`
}

type MeOutput struct {
	Body authpkg.User
}

// RegisterMe registers GET /auth/me.
func RegisterMe(api huma.API, svc *authpkg.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "auth-me",
		Summary:     "Get current user",
		Description: "Returns the authenticated user from the session cookie.",
		Method:      http.MethodGet,
		Path:        "/me",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *MeInput) (*MeOutput, error) {
		token := extractSessionToken(input.Cookie)

		user, err := svc.GetSession(ctx, token)
		if err != nil {
			if errors.Is(err, authpkg.ErrNoSession) || errors.Is(err, authpkg.ErrAccountBanned) {
				return nil, huma.Error401Unauthorized("not authenticated")
			}
			return nil, huma.Error500InternalServerError("session error")
		}

		return &MeOutput{Body: *user}, nil
	})
}

// extractSessionToken parses the "session" cookie value from a raw Cookie header.
func extractSessionToken(rawCookie string) string {
	header := http.Header{}
	header.Add("Cookie", rawCookie)
	req := http.Request{Header: header}
	c, err := req.Cookie("session")
	if err != nil {
		return ""
	}
	return c.Value
}
