package me

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
)

type UpdateInput struct {
	Body struct {
		DisplayName string  `json:"display_name" required:"true" doc:"User's display name"`
		Username    *string `json:"username,omitempty" doc:"Optional unique handle"`
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

		updated, err := store.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
			DisplayName:   input.Body.DisplayName,
			Username:      input.Body.Username,
			PhotoUrl:      user.PhotoURL,
			SportsProfile: user.SportsProfile,
			ContactInfo:   user.ContactInfo,
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
