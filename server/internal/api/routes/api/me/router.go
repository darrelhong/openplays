package me

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
	"openplays/server/internal/notifications"
)

// Register registers all /me routes with auth middleware.
func Register(api huma.API, svc *auth.Service, store interface {
	ProfileStore
	SuggestedPlayersStore
	FollowRequestsStore
}, avatarService AvatarService, notifier notifications.Sender) {
	grp := huma.NewGroup(api, "/me")
	grp.UseMiddleware(authmw.RequireAuth(api, svc))
	RegisterGet(grp, store)
	RegisterSuggestedPlayers(grp, store)
	RegisterUpdate(grp, store)
	RegisterFollowRequests(grp, store, notifier)
	RegisterAvatar(grp, avatarService)
}
