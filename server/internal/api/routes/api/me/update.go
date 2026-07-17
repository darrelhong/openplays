package me

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/usernames"
)

type UpdateInput struct {
	Body struct {
		DisplayName   string               `json:"display_name" required:"true" doc:"User's display name"`
		Username      *string              `json:"username,omitempty" doc:"Optional unique handle"`
		SportsProfile *model.SportsProfile `json:"sports_profile,omitempty" doc:"Self-rated sport levels"`
		Bio           *string              `json:"bio,omitempty" doc:"Optional public bio, up to 500 characters"`
		ProfileLinks  *model.ProfileLinks  `json:"profile_links,omitempty" doc:"Optional public profile identifiers"`
	}
}

type UpdateOutput struct {
	Body auth.User
}

// RegisterUpdate registers PATCH /me.
func RegisterUpdate(api huma.API, store ProfileStore) {
	huma.Register(api, huma.Operation{
		OperationID: "update-me",
		Summary:     "Update profile",
		Description: "Update the current user's display name and username. Requires session cookie.",
		Method:      http.MethodPatch,
		Path:        "/",
		Tags:        []string{"Me"},
	}, func(ctx context.Context, input *UpdateInput) (*UpdateOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		// Validate display_name not empty
		displayName := strings.TrimSpace(input.Body.DisplayName)
		if displayName == "" {
			return nil, huma.Error422UnprocessableEntity("display_name cannot be empty")
		}

		// Username: if provided, normalize and validate for profile URLs.
		username := user.Username
		if input.Body.Username != nil {
			normalized, err := usernames.Normalize(*input.Body.Username)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(err.Error())
			}
			username = &normalized
		}

		sportsProfile := user.SportsProfile
		if input.Body.SportsProfile != nil {
			sportsProfile = input.Body.SportsProfile
		}
		sportsProfileRaw, err := model.SportsProfileString(sportsProfile)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}

		bio := user.Bio
		if input.Body.Bio != nil {
			bio, err = model.NormalizeBio(input.Body.Bio)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(err.Error())
			}
		}

		profileLinks := user.ProfileLinks
		if input.Body.ProfileLinks != nil {
			profileLinks = input.Body.ProfileLinks
		}
		profileLinksRaw, err := model.ProfileLinksString(profileLinks)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}

		updated, err := store.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
			DisplayName:   displayName,
			Username:      username,
			SportsProfile: sportsProfileRaw,
			Bio:           bio,
			ProfileLinks:  profileLinksRaw,
			ID:            user.ID,
		})
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return nil, huma.Error409Conflict("username already taken")
			}
			return nil, huma.Error500InternalServerError("failed to update profile")
		}

		return &UpdateOutput{Body: auth.MapUser(updated)}, nil
	})
}
