package plays

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

type UpdatePlayInput struct {
	ID   string `path:"id" doc:"Play ID"`
	Body struct {
		Name            *string               `json:"name,omitempty" doc:"Optional custom game name" maxLength:"80"`
		Description     *string               `json:"description,omitempty" doc:"Optional game description" maxLength:"1000"`
		Visibility      *model.PlayVisibility `json:"visibility,omitempty" doc:"Set to unlisted to hide from public discovery while keeping direct-link access" enum:"public,unlisted"`
		StartsAt        *string               `json:"starts_at,omitempty" doc:"Start time in RFC3339 format"`
		DurationMinutes *int                  `json:"duration_minutes,omitempty" doc:"Duration in minutes (must be multiple of 15, max 300)" minimum:"15" maximum:"300"`
		Timezone        *string               `json:"timezone,omitempty" doc:"IANA timezone, e.g. Asia/Singapore"`
		GameType        *model.GameType       `json:"game_type,omitempty" doc:"Game type" enum:"doubles,singles,mixed_doubles,"`
		LevelMin        *string               `json:"level_min,omitempty" doc:"Minimum level code"`
		LevelMax        *string               `json:"level_max,omitempty" doc:"Maximum level code"`
		Fee             *int64                `json:"fee,omitempty" doc:"Fee in cents" minimum:"0"`
		FeeClear        bool                  `json:"fee_clear,omitempty" doc:"Clear the fee"`
		MaxPlayers      *int64                `json:"max_players,omitempty" doc:"Maximum number of players" minimum:"1"`
		Courts          *int64                `json:"courts,omitempty" doc:"Number of courts" minimum:"1"`
		CourtsClear     bool                  `json:"courts_clear,omitempty" doc:"Clear the court count"`
	}
}

type UpdatePlayOutput struct {
	Body PlayPublic
}

type UpdatePlayStore interface {
	GetPlayByID(ctx context.Context, id string) (db.GetPlayByIDRow, error)
	GetPlayHost(ctx context.Context, arg db.GetPlayHostParams) (db.PlayHost, error)
	CountReservedPlayParticipants(ctx context.Context, playID string) (int64, error)
	UpdateUserCreatedPlay(ctx context.Context, arg db.UpdateUserCreatedPlayParams) (db.Play, error)
	CreatePlayEvent(ctx context.Context, arg db.CreatePlayEventParams) (db.PlayEvent, error)
}

// RegisterUpdate registers PATCH /plays/{id}.
func RegisterUpdate(api huma.API, store UpdatePlayStore, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "update-play",
		Summary:     "Update a hosted play",
		Description: "Update host-managed fields for a user-created play. Requires the play host.",
		Method:      http.MethodPatch,
		Path:        "/{id}",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *UpdatePlayInput) (*UpdatePlayOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		play, err := store.GetPlayByID(ctx, input.ID)
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("play not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get play")
		}
		if play.CreatedBy == nil {
			return nil, huma.Error422UnprocessableEntity("cannot update imported plays")
		}
		if play.CancelledAt != nil {
			return nil, huma.Error409Conflict("play is cancelled")
		}
		if err := requirePlayHost(ctx, store, input.ID, user.ID); err != nil {
			return nil, err
		}
		changedFields := updatePlayChangedFields(input)
		name := play.Name
		if input.Body.Name != nil {
			name, err = model.CleanPlayName(input.Body.Name)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(err.Error())
			}
		}
		description := play.Description
		if input.Body.Description != nil {
			description, err = model.CleanPlayDescription(input.Body.Description)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(err.Error())
			}
		}
		visibility, err := cleanPlayVisibility(input.Body.Visibility, "")
		if err != nil {
			return nil, err
		}

		startsAt, endsAt, err := updatePlayTimes(play.StartsAt, play.EndsAt, input.Body.StartsAt, input.Body.DurationMinutes)
		if err != nil {
			return nil, err
		}

		timezone := play.Timezone
		if input.Body.Timezone != nil {
			if tz := strings.TrimSpace(*input.Body.Timezone); tz != "" {
				timezone = tz
			}
		}

		gameType := play.GameType
		if input.Body.GameType != nil {
			if strings.TrimSpace(string(*input.Body.GameType)) == "" {
				gameType = nil
			} else {
				gameType = input.Body.GameType
			}
		}

		levelMin, levelMax, levelMinOrd, levelMaxOrd, err := updatePlayLevels(play, input.Body.LevelMin, input.Body.LevelMax)
		if err != nil {
			return nil, err
		}

		fee := play.Fee
		if input.Body.FeeClear {
			fee = nil
		}
		if input.Body.Fee != nil {
			if *input.Body.Fee < 0 {
				return nil, huma.Error422UnprocessableEntity("fee must be at least 0")
			}
			fee = input.Body.Fee
		}

		maxPlayers := play.MaxPlayers
		if input.Body.MaxPlayers != nil {
			if *input.Body.MaxPlayers < 1 {
				return nil, huma.Error422UnprocessableEntity("max_players must be at least 1")
			}
			maxPlayers = input.Body.MaxPlayers
		}
		if maxPlayers == nil {
			return nil, huma.Error500InternalServerError("play is missing max_players")
		}

		reservedCount, err := store.CountReservedPlayParticipants(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count participants")
		}
		if *maxPlayers < reservedCount {
			return nil, huma.Error409Conflict("max_players cannot be less than reserved participants")
		}

		courts := play.Courts
		if input.Body.CourtsClear {
			courts = nil
		}
		if input.Body.Courts != nil {
			if *input.Body.Courts < 1 {
				return nil, huma.Error422UnprocessableEntity("courts must be at least 1")
			}
			courts = input.Body.Courts
		}

		updated, err := store.UpdateUserCreatedPlay(ctx, db.UpdateUserCreatedPlayParams{
			ID:          input.ID,
			Name:        name,
			Description: description,
			Visibility:  visibility,
			GameType:    gameType,
			StartsAt:    startsAt,
			EndsAt:      endsAt,
			Timezone:    timezone,
			LevelMin:    levelMin,
			LevelMax:    levelMax,
			LevelMinOrd: levelMinOrd,
			LevelMaxOrd: levelMaxOrd,
			Fee:         fee,
			MaxPlayers:  maxPlayers,
			Courts:      courts,
		})
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("play not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to update play")
		}
		if len(changedFields) > 0 {
			metadata, err := metadataJSON(playEventMetadata{
				"changed_fields": changedFields,
			})
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to record play event")
			}
			actorUserID, actorDisplayName := playEventActor(user)
			if err := recordPlayEvent(ctx, store, db.CreatePlayEventParams{
				PlayID:           input.ID,
				EventType:        model.PlayEventUpdated,
				ActorUserID:      actorUserID,
				ActorDisplayName: actorDisplayName,
				Metadata:         metadata,
			}); err != nil {
				return nil, huma.Error500InternalServerError("failed to record play event")
			}
		}

		return &UpdatePlayOutput{Body: publicPlayFromDB(updated)}, nil
	})
}

func updatePlayChangedFields(input *UpdatePlayInput) []string {
	fields := make([]string, 0, 11)
	if input.Body.Name != nil {
		fields = append(fields, "name")
	}
	if input.Body.Description != nil {
		fields = append(fields, "description")
	}
	if input.Body.Visibility != nil {
		fields = append(fields, "visibility")
	}
	if input.Body.StartsAt != nil {
		fields = append(fields, "starts_at")
	}
	if input.Body.DurationMinutes != nil {
		fields = append(fields, "duration_minutes")
	}
	if input.Body.Timezone != nil {
		fields = append(fields, "timezone")
	}
	if input.Body.GameType != nil {
		fields = append(fields, "game_type")
	}
	if input.Body.LevelMin != nil {
		fields = append(fields, "level_min")
	}
	if input.Body.LevelMax != nil {
		fields = append(fields, "level_max")
	}
	if input.Body.Fee != nil || input.Body.FeeClear {
		fields = append(fields, "fee")
	}
	if input.Body.MaxPlayers != nil {
		fields = append(fields, "max_players")
	}
	if input.Body.Courts != nil || input.Body.CourtsClear {
		fields = append(fields, "courts")
	}
	return fields
}

func updatePlayTimes(currentStart, currentEnd time.Time, startsAtInput *string, durationInput *int) (time.Time, time.Time, error) {
	startsAt := currentStart
	if startsAtInput != nil {
		parsed, err := time.Parse(time.RFC3339, *startsAtInput)
		if err != nil {
			return time.Time{}, time.Time{}, huma.Error422UnprocessableEntity("invalid starts_at: must be RFC3339")
		}
		startsAt = parsed.UTC()
	}
	if startsAt.Before(time.Now().UTC()) {
		return time.Time{}, time.Time{}, huma.Error422UnprocessableEntity("starts_at must be in the future")
	}

	duration := int(currentEnd.Sub(currentStart) / time.Minute)
	if durationInput != nil {
		duration = *durationInput
	}
	if duration < 15 || duration > 300 {
		return time.Time{}, time.Time{}, huma.Error422UnprocessableEntity("duration_minutes must be between 15 and 300")
	}
	if duration%15 != 0 {
		return time.Time{}, time.Time{}, huma.Error422UnprocessableEntity("duration_minutes must be a multiple of 15")
	}

	return startsAt, startsAt.Add(time.Duration(duration) * time.Minute), nil
}

func updatePlayLevels(play db.GetPlayByIDRow, levelMinInput, levelMaxInput *string) (*string, *string, *int64, *int64, error) {
	levelMin := play.LevelMin
	levelMax := play.LevelMax
	levelMinOrd := play.LevelMinOrd
	levelMaxOrd := play.LevelMaxOrd

	if levelMinInput != nil {
		levelMin = cleanedOptionalString(*levelMinInput)
		ord, err := levelOrdForUpdate(play.Sport, "level_min", levelMin)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		levelMinOrd = ord
	}
	if levelMaxInput != nil {
		levelMax = cleanedOptionalString(*levelMaxInput)
		ord, err := levelOrdForUpdate(play.Sport, "level_max", levelMax)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		levelMaxOrd = ord
	}
	if levelMinOrd != nil && levelMaxOrd != nil && *levelMinOrd > *levelMaxOrd {
		return nil, nil, nil, nil, huma.Error422UnprocessableEntity("level_min cannot be higher than level_max")
	}

	return levelMin, levelMax, levelMinOrd, levelMaxOrd, nil
}

func cleanedOptionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cleanPlayVisibility(input *model.PlayVisibility, fallback model.PlayVisibility) (model.PlayVisibility, error) {
	if input == nil {
		return fallback, nil
	}
	switch *input {
	case model.PlayVisibilityPublic, model.PlayVisibilityUnlisted:
		return *input, nil
	default:
		return "", huma.Error422UnprocessableEntity("visibility must be public or unlisted")
	}
}

func levelOrdForUpdate(sport model.Sport, field string, value *string) (*int64, error) {
	if value == nil {
		return nil, nil
	}
	ord := model.LevelOrd(sport, *value)
	if ord == nil {
		return nil, huma.Error422UnprocessableEntity(field + " is not valid for this sport")
	}
	v := int64(*ord)
	return &v, nil
}

func publicPlayFromDB(play db.Play) PlayPublic {
	createdAt, updatedAt := publicPlayTimestamps(play.CreatedBy, play.CreatedAt, play.UpdatedAt)
	return PlayPublic{
		ID:          play.ID,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		ListingType: play.ListingType,
		Sport:       play.Sport,
		GameType:    play.GameType,
		HostName:    play.HostName,
		Name:        play.Name,
		Description: play.Description,
		Visibility:  play.Visibility,
		StartsAt:    play.StartsAt.Format(time.RFC3339),
		EndsAt:      play.EndsAt.Format(time.RFC3339),
		Timezone:    play.Timezone,
		CancelledAt: publicOptionalTimestamp(play.CancelledAt),
		Venue:       play.Venue,
		VenueName:   play.Venue,
		VenueID:     play.VenueID,
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
		Source:      play.Source,
		CreatedBy:   play.CreatedBy,
	}
}
