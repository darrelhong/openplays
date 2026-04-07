package plays

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/db"
)

// Register registers all /api/plays endpoints.
func Register(api huma.API, queries *db.Queries) {
	grp := huma.NewGroup(api, "/plays")
	RegisterList(grp, queries)
	RegisterGet(grp, queries)
}
