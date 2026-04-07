package api

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/db"

	playsRouter "openplays/server/internal/api/routes/api/plays"
)

// Register registers all routes under /api.
func Register(api huma.API, queries *db.Queries) {
	grp := huma.NewGroup(api, "/api")
	playsRouter.Register(grp, queries)
}
