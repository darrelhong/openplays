package api

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/auth"
	"openplays/server/internal/db"

	authRouter "openplays/server/internal/api/routes/api/auth"
	meRouter "openplays/server/internal/api/routes/api/me"
	playsRouter "openplays/server/internal/api/routes/api/plays"
	venuesRouter "openplays/server/internal/api/routes/api/venues"
)

// Register registers all routes under /api.
func Register(api huma.API, queries *db.Queries, svc *auth.Service, googleVerifier *auth.GoogleVerifier, cookieSecure bool) {
	grp := huma.NewGroup(api, "/api")

	// Public routes
	authRouter.Register(grp, svc, googleVerifier, authRouter.CookieConfig{Secure: cookieSecure})
	playsRouter.Register(grp, queries)
	venuesRouter.Register(grp, queries)

	// Protected routes (auth middleware applied via huma.UseMiddleware)
	meRouter.Register(grp, svc, queries)
}
