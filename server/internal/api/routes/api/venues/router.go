package venues

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/geo"
)

// Register registers all /api/venues endpoints.
func Register(api huma.API, queries *db.Queries, svc *auth.Service, places geo.PlaceProvider) {
	grp := huma.NewGroup(api, "/venues")
	RegisterList(grp, queries)
	RegisterSearch(grp, queries, places, authmw.RequireAuth(api, svc))
	RegisterResolve(grp, queries, places, authmw.RequireAuth(api, svc))
}
