package auth

import (
	"github.com/danielgtaylor/huma/v2"

	authpkg "openplays/server/internal/auth"
)

// Register registers auth routes under /auth (login + logout).
func Register(api huma.API, svc *authpkg.Service, emailOTPSvc *authpkg.EmailOTPService, googleVerifier *authpkg.GoogleVerifier, facebookVerifier *authpkg.FacebookVerifier, cookieCfg CookieConfig) {
	grp := huma.NewGroup(api, "/auth")
	RegisterGoogle(grp, svc, googleVerifier, cookieCfg)
	RegisterFacebook(grp, svc, facebookVerifier, cookieCfg)
	RegisterEmailOTP(grp, emailOTPSvc, cookieCfg)
	RegisterLogout(grp, svc, cookieCfg)
}
