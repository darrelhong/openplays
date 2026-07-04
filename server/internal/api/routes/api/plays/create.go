package plays

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

type CreateInput struct {
	Body struct {
		Sport           model.Sport           `json:"sport" required:"true" doc:"Sport type" enum:"badminton,tennis,football,pickleball"`
		Venue           string                `json:"venue" required:"true" doc:"Venue name (free text)"`
		VenueID         *int64                `json:"venue_id,omitempty" doc:"Stored venue ID selected from venue search"`
		Name            *string               `json:"name,omitempty" doc:"Optional custom game name" maxLength:"80"`
		Description     *string               `json:"description,omitempty" doc:"Optional game description" maxLength:"1000"`
		Visibility      *model.PlayVisibility `json:"visibility,omitempty" doc:"Set to unlisted to hide from public discovery while keeping direct-link access" enum:"public,unlisted"`
		StartsAt        string                `json:"starts_at" required:"true" doc:"Start time in RFC3339 format"`
		DurationMinutes int                   `json:"duration_minutes" required:"true" doc:"Duration in minutes (must be multiple of 15, max 300)" minimum:"15" maximum:"300"`
		Timezone        string                `json:"timezone" doc:"IANA timezone, e.g. Asia/Singapore" default:"Asia/Singapore"`
		GameType        *model.GameType       `json:"game_type,omitempty" doc:"Game type" enum:"doubles,singles,mixed_doubles,"`
		LevelMin        *string               `json:"level_min,omitempty" doc:"Minimum level code"`
		LevelMax        *string               `json:"level_max,omitempty" doc:"Maximum level code"`
		Fee             *int64                `json:"fee,omitempty" doc:"Fee in cents"`
		Currency        string                `json:"currency" doc:"Currency code" default:"SGD"`
		MaxPlayers      *int64                `json:"max_players,omitempty" doc:"Maximum number of players"`
		SlotsLeft       *int64                `json:"slots_left,omitempty" doc:"Available slots"`
		Courts          *int64                `json:"courts,omitempty" doc:"Number of courts"`
		Contacts        model.Contacts        `json:"contacts,omitempty" doc:"Contact methods"`
		GenderPref      *model.GenderPref     `json:"gender_pref,omitempty" doc:"Gender preference" enum:"all,male_only,female_only,"`
		RequireWaitlist *bool                 `json:"require_waitlist,omitempty" doc:"When true, joiners request a spot and a host adds each player to the game or waitlist"`
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

		if input.Body.MaxPlayers == nil {
			return nil, huma.Error422UnprocessableEntity("max_players is required")
		}
		if *input.Body.MaxPlayers < 1 {
			return nil, huma.Error422UnprocessableEntity("max_players must be at least 1")
		}

		slotsLeft := *input.Body.MaxPlayers - 1
		name, err := model.CleanPlayName(input.Body.Name)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}
		description, err := model.CleanPlayDescription(input.Body.Description)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}
		visibility, err := cleanPlayVisibility(input.Body.Visibility, model.PlayVisibilityPublic)
		if err != nil {
			return nil, err
		}
		requireWaitlist := input.Body.RequireWaitlist != nil && *input.Body.RequireWaitlist

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

		venueName := strings.TrimSpace(input.Body.Venue)
		if venueName == "" {
			return nil, huma.Error422UnprocessableEntity("venue is required")
		}

		var resolvedVenueID *int64
		var resolvedVenue *db.Venue
		if queries, ok := store.(*db.Queries); ok {
			resolvedVenue, err = resolveVenue(ctx, queries, venueName, input.Body.VenueID)
			if err != nil {
				return nil, err
			}
			if resolvedVenue != nil {
				resolvedVenueID = &resolvedVenue.ID
				venueName = resolvedVenue.Name
			}
		}

		play, err := store.CreatePlay(ctx, db.CreatePlayParams{
			ID:              uuid.NewString(),
			ListingType:     model.ListingPlay,
			Sport:           input.Body.Sport,
			GameType:        input.Body.GameType,
			HostName:        user.DisplayName,
			Name:            name,
			Description:     description,
			Visibility:      visibility,
			StartsAt:        startsAt,
			EndsAt:          endsAt,
			Timezone:        input.Body.Timezone,
			Venue:           venueName,
			VenueID:         resolvedVenueID,
			LevelMin:        input.Body.LevelMin,
			LevelMax:        input.Body.LevelMax,
			LevelMinOrd:     levelMinOrd,
			LevelMaxOrd:     levelMaxOrd,
			Fee:             input.Body.Fee,
			Currency:        input.Body.Currency,
			MaxPlayers:      input.Body.MaxPlayers,
			SlotsLeft:       &slotsLeft,
			Courts:          input.Body.Courts,
			Contacts:        input.Body.Contacts,
			GenderPref:      input.Body.GenderPref,
			CreatedBy:       &user.ID,
			RequireWaitlist: requireWaitlist,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create play")
		}

		var hostRatingCode *string
		var hostRatingOrd *int64
		if user.SportsProfile != nil {
			hostRatingCode = user.SportsProfile.LevelFor(input.Body.Sport)
			if hostRatingCode != nil {
				if ord := model.LevelOrd(input.Body.Sport, *hostRatingCode); ord != nil {
					v := int64(*ord)
					hostRatingOrd = &v
				}
			}
		}

		if _, err := store.CreatePlayHost(ctx, db.CreatePlayHostParams{
			PlayID: play.ID,
			UserID: user.ID,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to seed play host")
		}

		if _, err := store.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
			PlayID:     play.ID,
			UserID:     &user.ID,
			RatingCode: hostRatingCode,
			RatingOrd:  hostRatingOrd,
			Status:     model.ParticipantConfirmed,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to seed creator participant")
		}
		actorUserID, actorDisplayName := playEventActor(user)
		if err := recordPlayEvent(ctx, store, db.CreatePlayEventParams{
			PlayID:           play.ID,
			EventType:        model.PlayEventCreated,
			ActorUserID:      actorUserID,
			ActorDisplayName: actorDisplayName,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to record play event")
		}
		play.SlotsLeft = &slotsLeft
		createdAt, updatedAt := publicPlayTimestamps(play.CreatedBy, play.CreatedAt, play.UpdatedAt)

		out := &CreateOutput{
			Body: PlayPublic{
				ID:                 play.ID,
				CreatedAt:          createdAt,
				UpdatedAt:          updatedAt,
				ListingType:        play.ListingType,
				Sport:              play.Sport,
				GameType:           play.GameType,
				HostName:           play.HostName,
				Name:               play.Name,
				Description:        play.Description,
				Visibility:         play.Visibility,
				RequireWaitlist:    play.RequireWaitlist,
				StartsAt:           play.StartsAt.Format(time.RFC3339),
				EndsAt:             play.EndsAt.Format(time.RFC3339),
				Timezone:           play.Timezone,
				Venue:              play.Venue,
				VenueName:          venueDisplayName(play.Venue, resolvedVenue),
				VenueID:            resolvedVenueID,
				VenuePostalCode:    venuePostalCode(resolvedVenue),
				VenueLatitude:      venueLatitude(resolvedVenue),
				VenueLongitude:     venueLongitude(resolvedVenue),
				VenueGooglePlaceID: venueGooglePlaceID(resolvedVenue),
				LevelMin:           play.LevelMin,
				LevelMax:           play.LevelMax,
				Fee:                play.Fee,
				Currency:           play.Currency,
				MaxPlayers:         play.MaxPlayers,
				SlotsLeft:          play.SlotsLeft,
				Courts:             play.Courts,
				Contacts:           play.Contacts,
				GenderPref:         play.GenderPref,
				Meta:               play.Meta,
				CreatedBy:          play.CreatedBy,
			},
		}
		return out, nil
	})
}

func resolveVenue(ctx context.Context, queries *db.Queries, rawVenue string, venueID *int64) (*db.Venue, error) {
	venue := strings.TrimSpace(rawVenue)
	if venue == "" {
		return nil, nil
	}

	if venueID != nil {
		v, err := queries.GetVenueByID(ctx, *venueID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("venue_id does not match a stored venue")
		}
		return &v, nil
	}

	if v, err := queries.GetVenueByAlias(ctx, venue); err == nil {
		return &v, nil
	}

	if lower := strings.ToLower(venue); lower != venue {
		if v, err := queries.GetVenueByAlias(ctx, lower); err == nil {
			return &v, nil
		}
	}

	names, err := queries.ListVenueNames(ctx)
	if err != nil {
		return nil, nil
	}

	for _, item := range names {
		if strings.EqualFold(strings.TrimSpace(item.Name), venue) {
			v, err := queries.GetVenueByID(ctx, item.ID)
			if err != nil {
				return nil, nil
			}
			return &v, nil
		}
	}

	return nil, nil
}

func venueDisplayName(rawVenue string, venue *db.Venue) string {
	if venue != nil {
		return venue.Name
	}
	return rawVenue
}

func venuePostalCode(venue *db.Venue) *string {
	if venue == nil {
		return nil
	}
	return venue.PostalCode
}

func venueLatitude(venue *db.Venue) *float64 {
	if venue == nil {
		return nil
	}
	return &venue.Latitude
}

func venueLongitude(venue *db.Venue) *float64 {
	if venue == nil {
		return nil
	}
	return &venue.Longitude
}

func venueGooglePlaceID(venue *db.Venue) *string {
	if venue == nil {
		return nil
	}
	return venue.GooglePlaceID
}

// Ensure *db.Queries satisfies CreatePlayStore
var _ CreatePlayStore = (*db.Queries)(nil)
