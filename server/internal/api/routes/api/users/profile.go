package users

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/usernames"
)

type ProfileInput struct {
	Username string `path:"username" doc:"Username"`
}

type PublicUserProfile struct {
	ID                 string               `json:"id"`
	DisplayName        string               `json:"display_name"`
	Username           string               `json:"username"`
	PhotoURL           *string              `json:"photo_url,omitempty"`
	SportsProfile      *model.SportsProfile `json:"sports_profile,omitempty"`
	RosteredPlayCount  int64                `json:"rostered_play_count"`
}

type ProfileOutput struct {
	Body PublicUserProfile
}

func RegisterProfile(api huma.API, store ProfileStore) {
	huma.Register(api, huma.Operation{
		OperationID: "get-user-profile",
		Summary:     "Get user profile",
		Description: "Returns a minimal public profile for an active user. Requires authentication.",
		Method:      http.MethodGet,
		Path:        "/{username}",
		Tags:        []string{"Users"},
	}, func(ctx context.Context, input *ProfileInput) (*ProfileOutput, error) {
		username, err := usernames.Normalize(input.Username)
		if err != nil {
			return nil, huma.Error404NotFound("user not found")
		}

		row, err := store.GetActiveUserProfileByUsername(ctx, &username)
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("user not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get user profile")
		}
		count, err := store.CountRosteredPlaysByUser(ctx, row.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count user plays")
		}

		return &ProfileOutput{Body: mapPublicUserProfile(row, count)}, nil
	})
}

func mapPublicUserProfile(row db.GetActiveUserProfileByUsernameRow, rosteredPlayCount int64) PublicUserProfile {
	profile, _ := model.ParseSportsProfile(row.SportsProfile)
	username := ""
	if row.Username != nil {
		username = *row.Username
	}
	return PublicUserProfile{
		ID:                row.ID,
		DisplayName:       row.DisplayName,
		Username:          username,
		PhotoURL:          row.PhotoUrl,
		SportsProfile:     profile,
		RosteredPlayCount: rosteredPlayCount,
	}
}
