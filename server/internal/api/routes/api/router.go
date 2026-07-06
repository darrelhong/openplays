package api

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/geo"
	"openplays/server/internal/notifications"

	"openplays/server/internal/api/authmw"
	authRouter "openplays/server/internal/api/routes/api/auth"
	chatRouter "openplays/server/internal/api/routes/api/chat"
	devRouter "openplays/server/internal/api/routes/api/dev"
	meRouter "openplays/server/internal/api/routes/api/me"
	notificationsRouter "openplays/server/internal/api/routes/api/notifications"
	playsRouter "openplays/server/internal/api/routes/api/plays"
	usersRouter "openplays/server/internal/api/routes/api/users"
	venuesRouter "openplays/server/internal/api/routes/api/venues"
)

// Register registers all routes under /api. The push service is created by
// the caller so background workers (e.g. the review prompter) share it.
func Register(api huma.API, queries *db.Queries, svc *auth.Service, googleVerifier *auth.GoogleVerifier, facebookVerifier *auth.FacebookVerifier, places geo.PlaceProvider, pushService *notifications.WebPushService, cookieSecure bool, devAuthEnabled bool) {
	grp := huma.NewGroup(api, "/api")

	// Public routes
	authRouter.Register(grp, svc, googleVerifier, facebookVerifier, authRouter.CookieConfig{Secure: cookieSecure})
	chatRouter.Register(grp, queries, svc, pushService)
	notificationsRouter.Register(grp, pushService, authmw.RequireAuth(api, svc))
	playsRouter.Register(grp, queries, svc, pushService)
	venuesRouter.Register(grp, queries, svc, places)
	devRouter.Register(grp, queries, devRouter.Config{Enabled: devAuthEnabled, CookieSecure: cookieSecure})

	// Protected routes (auth middleware applied via huma.UseMiddleware)
	meRouter.Register(grp, svc, queries)
	usersRouter.Register(grp, queries, svc)
}
