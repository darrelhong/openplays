package me

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/users"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

type SuggestedPlayersInput struct {
	Sport         string `query:"sport" doc:"Sport for rating snapshot" enum:"badminton,tennis,football,pickleball,"`
	Limit         int64  `query:"limit" default:"10" minimum:"1" maximum:"25" doc:"Maximum users to return"`
	ExcludePlayID string `query:"exclude_play_id" doc:"Exclude users already participating in this play"`
}

type SuggestedPlayersStore interface {
	ListSuggestedPlayers(ctx context.Context, arg db.ListSuggestedPlayersParams) ([]db.ListSuggestedPlayersRow, error)
}

type SuggestedPlayersOutput struct {
	Body users.SearchPage
}

// RegisterSuggestedPlayers registers GET /me/suggested-players.
func RegisterSuggestedPlayers(api huma.API, store SuggestedPlayersStore) {
	huma.Register(api, huma.Operation{
		OperationID: "list-suggested-players",
		Summary:     "List suggested players",
		Description: "Returns active players from the viewer's ended games, ranked by games played together. Requires authentication.",
		Method:      http.MethodGet,
		Path:        "/suggested-players",
		Tags:        []string{"Me"},
	}, func(ctx context.Context, input *SuggestedPlayersInput) (*SuggestedPlayersOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		sport := strings.TrimSpace(input.Sport)
		if sport != "" && !slices.Contains(model.SportValues, sport) {
			return nil, huma.Error422UnprocessableEntity(
				fmt.Sprintf("invalid sport: must be one of %s", strings.Join(model.SportValues, ", ")))
		}

		rows, err := store.ListSuggestedPlayers(ctx, db.ListSuggestedPlayersParams{
			UserID:        user.ID,
			ExcludePlayID: strings.TrimSpace(input.ExcludePlayID),
			Limit:         input.Limit,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list suggested players")
		}

		items := make([]users.UserSummary, 0, len(rows))
		for _, row := range rows {
			profile, _ := model.ParseSportsProfile(row.SportsProfile)
			items = append(items, users.UserSummary{
				ID:          row.ID,
				DisplayName: row.DisplayName,
				Username:    row.Username,
				PhotoURL:    row.PhotoUrl,
				RatingCode:  profile.LevelFor(model.Sport(sport)),
			})
		}

		return &SuggestedPlayersOutput{Body: users.SearchPage{Items: items}}, nil
	})
}
