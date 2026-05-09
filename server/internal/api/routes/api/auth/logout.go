package auth

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	authpkg "openplays/server/internal/auth"
)

type LogoutInput struct {
	Cookie string `header:"Cookie"`
}

type LogoutOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

// RegisterLogout registers POST /auth/logout.
func RegisterLogout(api huma.API, svc *authpkg.Service, cookieCfg CookieConfig) {
	huma.Register(api, huma.Operation{
		OperationID: "auth-logout",
		Summary:     "Log out",
		Description: "Deletes the current session and clears the session cookie.",
		Method:      http.MethodPost,
		Path:        "/logout",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *LogoutInput) (*LogoutOutput, error) {
		token := extractSessionToken(input.Cookie)

		if err := svc.Logout(ctx, token); err != nil {
			return nil, huma.Error500InternalServerError("logout failed")
		}

		out := &LogoutOutput{}
		out.SetCookie = clearSessionCookie(cookieCfg)
		return out, nil
	})
}
