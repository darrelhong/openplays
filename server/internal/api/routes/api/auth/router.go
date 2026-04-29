package auth

import (
	"github.com/danielgtaylor/huma/v2"

	authpkg "openplays/server/internal/auth"
	"openplays/server/internal/db"
)

// Register registers all auth routes under /auth.
func Register(api huma.API, queries *db.Queries, verifier *authpkg.GoogleVerifier, cookieCfg CookieConfig) {
	svc := authpkg.NewService(queries)
	grp := huma.NewGroup(api, "/auth")
	RegisterGoogle(grp, svc, verifier, cookieCfg)
	RegisterMe(grp, svc)
	RegisterLogout(grp, svc, cookieCfg)
}
