package chat

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/notifications"
)

func Register(api huma.API, queries *db.Queries, svc *auth.Service, notifier notifications.Sender) {
	grp := huma.NewGroup(api, "/chat")
	grp.UseMiddleware(authmw.RequireAuth(api, svc))

	RegisterConversations(grp, queries)
	RegisterMessages(grp, queries, notifier)
}
