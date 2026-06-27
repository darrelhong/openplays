package plays

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/notifications"
)

// Register registers play endpoints.
// Public: /plays list and get. Protected: /plays mutations and /me/plays.
func Register(api huma.API, queries *db.Queries, svc *auth.Service, notifier notifications.Sender) {
	grp := huma.NewGroup(api, "/plays")

	RegisterMyList(api, queries, authmw.RequireAuth(api, svc))
	RegisterMyFavourites(api, queries, authmw.RequireAuth(api, svc))

	// Public
	RegisterList(grp, queries, authmw.OptionalAuth(api, svc))
	RegisterGet(grp, queries, authmw.OptionalAuth(api, svc))

	// Protected (auth middleware on this operation only)
	RegisterCreate(grp, queries, authmw.RequireAuth(api, svc))
	RegisterUpdate(grp, queries, authmw.RequireAuth(api, svc))
	RegisterDelete(grp, queries, authmw.RequireAuth(api, svc))
	RegisterFavourite(grp, queries, authmw.RequireAuth(api, svc))
	RegisterJoin(grp, queries, authmw.RequireAuth(api, svc), notifier)
	RegisterLeave(grp, queries, authmw.RequireAuth(api, svc), notifier)
	RegisterConfirmParticipant(grp, queries, authmw.RequireAuth(api, svc), notifier)
	RegisterHostRosterManagement(grp, queries, authmw.RequireAuth(api, svc), notifier)
}
