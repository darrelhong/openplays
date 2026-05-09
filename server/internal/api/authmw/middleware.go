// Package authmw provides authentication middleware for protected routes.
package authmw

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/auth"
)

type contextKey struct{}

// RequireAuth returns a middleware that validates the session cookie
// and attaches the user to the request context. If auth fails, writes 401.
func RequireAuth(api huma.API, svc *auth.Service) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		cookie, err := huma.ReadCookie(ctx, "session")
		if err != nil || cookie.Value == "" {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "not authenticated")
			return
		}

		user, err := svc.GetSession(ctx.Context(), cookie.Value)
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "not authenticated")
			return
		}

		next(huma.WithValue(ctx, contextKey{}, user))
	}
}

// UserFromContext returns the authenticated user from the request context.
// Returns nil if no user (should not happen behind middleware).
func UserFromContext(ctx context.Context) *auth.User {
	user, _ := ctx.Value(contextKey{}).(*auth.User)
	return user
}
