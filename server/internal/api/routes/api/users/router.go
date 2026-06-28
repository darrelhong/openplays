package users

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
)

// Register registers all /users routes with auth middleware.
func Register(api huma.API, queries *db.Queries, svc *auth.Service) {
	grp := huma.NewGroup(api, "/users")
	grp.UseMiddleware(authmw.RequireAuth(api, svc))
	RegisterSearch(grp, queries)
	RegisterProfile(grp, queries)
}
