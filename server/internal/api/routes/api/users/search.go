package users

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

type SearchInput struct {
	Query string `query:"q" doc:"Search by display name or username"`
	Sport string `query:"sport" doc:"Sport for rating snapshot" enum:"badminton,tennis,football,pickleball,"`
	Limit int64  `query:"limit" default:"10" minimum:"1" maximum:"25" doc:"Maximum users to return"`
}

type UserSummary struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Username    *string `json:"username,omitempty"`
	PhotoURL    *string `json:"photo_url,omitempty"`
	RatingCode  *string `json:"rating_code,omitempty"`
}

type SearchPage struct {
	Items []UserSummary `json:"items"`
}

type SearchOutput struct {
	Body SearchPage
}

func RegisterSearch(api huma.API, store SearchStore) {
	huma.Register(api, huma.Operation{
		OperationID: "search-users",
		Summary:     "Search users",
		Description: "Search active users by display name or username. Requires authentication.",
		Method:      http.MethodGet,
		Path:        "/search",
		Tags:        []string{"Users"},
	}, func(ctx context.Context, input *SearchInput) (*SearchOutput, error) {
		sport := strings.TrimSpace(input.Sport)
		if sport != "" && !slices.Contains(model.SportValues, sport) {
			return nil, huma.Error422UnprocessableEntity(
				fmt.Sprintf("invalid sport: must be one of %s", strings.Join(model.SportValues, ", ")))
		}

		rows, err := store.SearchActiveUsers(ctx, db.SearchActiveUsersParams{
			Query: strings.TrimSpace(input.Query),
			Limit: input.Limit,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to search users")
		}

		items := make([]UserSummary, 0, len(rows))
		for _, row := range rows {
			items = append(items, mapUserSummary(row, model.Sport(sport)))
		}
		return &SearchOutput{Body: SearchPage{Items: items}}, nil
	})
}

func mapUserSummary(row db.SearchActiveUsersRow, sport model.Sport) UserSummary {
	profile, _ := model.ParseSportsProfile(row.SportsProfile)
	return UserSummary{
		ID:          row.ID,
		DisplayName: row.DisplayName,
		Username:    row.Username,
		PhotoURL:    row.PhotoUrl,
		RatingCode:  profile.LevelFor(sport),
	}
}
