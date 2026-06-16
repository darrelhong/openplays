package api

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/geo"

	authRouter "openplays/server/internal/api/routes/api/auth"
	devRouter "openplays/server/internal/api/routes/api/dev"
	meRouter "openplays/server/internal/api/routes/api/me"
	playsRouter "openplays/server/internal/api/routes/api/plays"
	usersRouter "openplays/server/internal/api/routes/api/users"
	venuesRouter "openplays/server/internal/api/routes/api/venues"
)

// Register registers all routes under /api.
func Register(api huma.API, queries *db.Queries, svc *auth.Service, googleVerifier *auth.GoogleVerifier, facebookVerifier *auth.FacebookVerifier, places geo.PlaceProvider, cookieSecure bool, devAuthEnabled bool) {
	grp := huma.NewGroup(api, "/api")

	// Public routes
	authRouter.Register(grp, svc, googleVerifier, facebookVerifier, authRouter.CookieConfig{Secure: cookieSecure})
	playsRouter.Register(grp, queries, svc)
	venuesRouter.Register(grp, queries, svc, places)
	devRouter.Register(grp, queries, devRouter.Config{Enabled: devAuthEnabled, CookieSecure: cookieSecure})

	// Protected routes (auth middleware applied via huma.UseMiddleware)
	meRouter.Register(grp, svc, queries)
	usersRouter.Register(grp, queries, svc)
}
