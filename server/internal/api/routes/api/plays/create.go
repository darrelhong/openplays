package plays

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

type CreateInput struct {
	Body struct {
		Sport           model.Sport       `json:"sport" required:"true" doc:"Sport type" enum:"badminton,tennis,football,pickleball"`
		Venue           string            `json:"venue" required:"true" doc:"Venue name (free text)"`
		StartsAt        string            `json:"starts_at" required:"true" doc:"Start time in RFC3339 format"`
		DurationMinutes int               `json:"duration_minutes" required:"true" doc:"Duration in minutes (must be multiple of 15, max 300)" minimum:"15" maximum:"300"`
		Timezone        string            `json:"timezone" doc:"IANA timezone, e.g. Asia/Singapore" default:"Asia/Singapore"`
		GameType        *model.GameType   `json:"game_type,omitempty" doc:"Game type" enum:"doubles,singles,mixed_doubles,"`
		LevelMin        *string           `json:"level_min,omitempty" doc:"Minimum level code"`
		LevelMax        *string           `json:"level_max,omitempty" doc:"Maximum level code"`
		Fee             *int64            `json:"fee,omitempty" doc:"Fee in cents"`
		Currency        string            `json:"currency" doc:"Currency code" default:"SGD"`
		MaxPlayers      *int64            `json:"max_players,omitempty" doc:"Maximum number of players"`
		SlotsLeft       *int64            `json:"slots_left,omitempty" doc:"Available slots"`
		Courts          *int64            `json:"courts,omitempty" doc:"Number of courts"`
		Contacts        model.Contacts    `json:"contacts,omitempty" doc:"Contact methods"`
		GenderPref      *model.GenderPref `json:"gender_pref,omitempty" doc:"Gender preference" enum:"all,male_only,female_only,"`
	}
}

type CreateOutput struct {
	Body PlayPublic
}

// RegisterCreate registers POST /plays/ (protected via per-operation middleware).
func RegisterCreate(api huma.API, store CreatePlayStore, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "create-play",
		Summary:     "Create a play",
		Description: "Create a new play session. Requires authentication.",
		Method:      http.MethodPost,
		Path:        "/",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *CreateInput) (*CreateOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		startsAt, err := time.Parse(time.RFC3339, input.Body.StartsAt)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid starts_at: must be RFC3339")
		}
		startsAt = startsAt.UTC() // store in UTC for consistent comparison

		// Validate: starts_at must be in the future
		if startsAt.Before(time.Now().UTC()) {
			return nil, huma.Error422UnprocessableEntity("starts_at must be in the future")
		}

		// Validate duration: must be multiple of 15 minutes
		if input.Body.DurationMinutes%15 != 0 {
			return nil, huma.Error422UnprocessableEntity("duration_minutes must be a multiple of 15")
		}

		// Compute ends_at from starts_at + duration
		endsAt := startsAt.Add(time.Duration(input.Body.DurationMinutes) * time.Minute)

		// Compute level ordinals if provided
		var levelMinOrd, levelMaxOrd *int64
		if input.Body.LevelMin != nil {
			if ord := model.LevelOrd(input.Body.Sport, *input.Body.LevelMin); ord != nil {
				v := int64(*ord)
				levelMinOrd = &v
			}
		}
		if input.Body.LevelMax != nil {
			if ord := model.LevelOrd(input.Body.Sport, *input.Body.LevelMax); ord != nil {
				v := int64(*ord)
				levelMaxOrd = &v
			}
		}

		play, err := store.CreatePlay(ctx, db.CreatePlayParams{
			ListingType: model.ListingPlay,
			Sport:       input.Body.Sport,
			GameType:    input.Body.GameType,
			HostName:    user.DisplayName,
			StartsAt:    startsAt,
			EndsAt:      endsAt,
			Timezone:    input.Body.Timezone,
			Venue:       input.Body.Venue,
			LevelMin:    input.Body.LevelMin,
			LevelMax:    input.Body.LevelMax,
			LevelMinOrd: levelMinOrd,
			LevelMaxOrd: levelMaxOrd,
			Fee:         input.Body.Fee,
			Currency:    input.Body.Currency,
			MaxPlayers:  input.Body.MaxPlayers,
			SlotsLeft:   input.Body.SlotsLeft,
			Courts:      input.Body.Courts,
			Contacts:    input.Body.Contacts,
			GenderPref:  input.Body.GenderPref,
			CreatedBy:   &user.ID,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create play")
		}

		out := &CreateOutput{
			Body: PlayPublic{
				ID:          play.ID,
				CreatedAt:   play.CreatedAt.Format(time.RFC3339),
				UpdatedAt:   play.UpdatedAt.Format(time.RFC3339),
				ListingType: play.ListingType,
				Sport:       play.Sport,
				GameType:    play.GameType,
				HostName:    play.HostName,
				StartsAt:    play.StartsAt.Format(time.RFC3339),
				EndsAt:      play.EndsAt.Format(time.RFC3339),
				Timezone:    play.Timezone,
				Venue:       play.Venue,
				VenueName:   play.Venue,
				LevelMin:    play.LevelMin,
				LevelMax:    play.LevelMax,
				Fee:         play.Fee,
				Currency:    play.Currency,
				MaxPlayers:  play.MaxPlayers,
				SlotsLeft:   play.SlotsLeft,
				Courts:      play.Courts,
				Contacts:    play.Contacts,
				GenderPref:  play.GenderPref,
				Meta:        play.Meta,
				CreatedBy:   play.CreatedBy,
			},
		}
		return out, nil
	})
}

// Ensure *db.Queries satisfies CreatePlayStore
var _ CreatePlayStore = (*db.Queries)(nil)
