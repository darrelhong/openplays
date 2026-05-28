package plays

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
)

// Register registers all /api/plays endpoints.
// Public: list, get. Protected: create.
func Register(api huma.API, queries *db.Queries, svc *auth.Service) {
	grp := huma.NewGroup(api, "/plays")

	// Public
	RegisterList(grp, queries)
	RegisterGet(grp, queries, authmw.OptionalAuth(api, svc))

	// Protected (auth middleware on this operation only)
	RegisterCreate(grp, queries, authmw.RequireAuth(api, svc))
	RegisterUpdate(grp, queries, authmw.RequireAuth(api, svc))
	RegisterDelete(grp, queries, authmw.RequireAuth(api, svc))
	RegisterJoin(grp, queries, authmw.RequireAuth(api, svc))
	RegisterLeave(grp, queries, authmw.RequireAuth(api, svc))
	RegisterConfirmParticipant(grp, queries, authmw.RequireAuth(api, svc))
	RegisterHostRosterManagement(grp, queries, authmw.RequireAuth(api, svc))
}
