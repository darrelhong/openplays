package auth

import (
	"github.com/danielgtaylor/huma/v2"

	authpkg "openplays/server/internal/auth"
)

// Register registers auth routes under /auth (login + logout only).
func Register(api huma.API, svc *authpkg.Service, verifier *authpkg.GoogleVerifier, cookieCfg CookieConfig) {
	grp := huma.NewGroup(api, "/auth")
	RegisterGoogle(grp, svc, verifier, cookieCfg)
	RegisterLogout(grp, svc, cookieCfg)
}
