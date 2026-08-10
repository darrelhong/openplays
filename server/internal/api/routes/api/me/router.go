package me

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
)

// Register registers all /me routes with auth middleware.
func Register(api huma.API, svc *auth.Service, store interface {
	ProfileStore
	SuggestedPlayersStore
}, avatarService AvatarService) {
	grp := huma.NewGroup(api, "/me")
	grp.UseMiddleware(authmw.RequireAuth(api, svc))
	RegisterGet(grp)
	RegisterSuggestedPlayers(grp, store)
	RegisterUpdate(grp, store)
	RegisterAvatar(grp, avatarService)
}
