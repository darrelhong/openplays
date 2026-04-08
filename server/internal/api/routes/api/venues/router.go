package venues

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/db"
)

// Register registers all /api/venues endpoints.
func Register(api huma.API, queries *db.Queries) {
	grp := huma.NewGroup(api, "/venues")
	RegisterList(grp, queries)
}
