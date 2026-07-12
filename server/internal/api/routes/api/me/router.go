package me

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
)

// Register registers all /me routes with auth middleware.
func Register(api huma.API, svc *auth.Service, store ProfileStore, avatarService AvatarService) {
	grp := huma.NewGroup(api, "/me")
	grp.UseMiddleware(authmw.RequireAuth(api, svc))
	RegisterGet(grp)
	RegisterUpdate(grp, store)
	RegisterAvatar(grp, avatarService)
}
