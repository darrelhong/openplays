package dev

import (
	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/auth"
	"openplays/server/internal/db"
)

type Config struct {
	Enabled      bool
	CookieSecure bool
}

func Register(api huma.API, queries *db.Queries, cfg Config) {
	if !cfg.Enabled {
		return
	}

	grp := huma.NewGroup(api, "/dev")
	RegisterLogin(grp, queries, cfg)
}

func mapUser(user db.User) auth.User {
	return auth.MapUser(user)
}
