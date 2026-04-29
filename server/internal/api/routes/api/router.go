package api

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/auth"
	"openplays/server/internal/db"

	authRouter "openplays/server/internal/api/routes/api/auth"
	playsRouter "openplays/server/internal/api/routes/api/plays"
	venuesRouter "openplays/server/internal/api/routes/api/venues"
)

// Register registers all routes under /api.
func Register(api huma.API, queries *db.Queries, googleVerifier *auth.GoogleVerifier, cookieSecure bool) {
	grp := huma.NewGroup(api, "/api")
	authRouter.Register(grp, queries, googleVerifier, authRouter.CookieConfig{Secure: cookieSecure})
	playsRouter.Register(grp, queries)
	venuesRouter.Register(grp, queries)
}
