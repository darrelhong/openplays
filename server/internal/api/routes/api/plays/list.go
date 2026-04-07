package plays

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/pagination"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

type ListInput struct {
	Sport   string `query:"sport" doc:"Filter by sport" enum:"badminton,tennis,football,pickleball,"`
	VenueID int64  `query:"venue_id" doc:"Filter by venue ID"`
	Cursor  int64  `query:"cursor" doc:"Cursor for pagination (play ID)"`
	Limit   int64  `query:"limit" default:"20" minimum:"1" maximum:"100" doc:"Number of results per page"`
}

type ListOutput struct {
	Body pagination.Page[PlayPublic]
}

func RegisterList(api huma.API, queries *db.Queries) {
	huma.Register(api, huma.Operation{
		OperationID: "list-plays",
		Summary:     "List upcoming plays",
		Method:      http.MethodGet,
		Path:        "/",
		Tags:        []string{"Plays"},
	}, func(ctx context.Context, input *ListInput) (*ListOutput, error) {
		if input.Sport != "" && !slices.Contains(model.SportValues, input.Sport) {
			return nil, huma.Error422UnprocessableEntity(
				fmt.Sprintf("invalid sport: must be one of %s", strings.Join(model.SportValues, ", ")))
		}

		pageSize := input.Limit + 1

		var sport interface{}
		if input.Sport != "" {
			sport = input.Sport
		}
		var venueID interface{}
		if input.VenueID != 0 {
			venueID = input.VenueID
		}
		var cursor interface{}
		if input.Cursor != 0 {
			cursor = input.Cursor
		}

		rows, err := queries.ListUpcomingPlays(ctx, db.ListUpcomingPlaysParams{
			Sport:    sport,
			VenueID:  venueID,
			Cursor:   cursor,
			PageSize: pageSize,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list plays", err)
		}

		total, err := queries.CountUpcomingPlays(ctx, db.CountUpcomingPlaysParams{
			Sport:   sport,
			VenueID: venueID,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count plays", err)
		}

		items := make([]PlayPublic, len(rows))
		for i, r := range rows {
			items[i] = PlayPublic{
				ID:              r.ID,
				Sport:           string(r.Sport),
				GameType:        gameTypeStr(r.GameType),
				HostName:        r.HostName,
				StartsAt:        r.StartsAt.Format(time.RFC3339),
				EndsAt:          r.EndsAt.Format(time.RFC3339),
				Timezone:        r.Timezone,
				Venue:           r.Venue,
				VenueName:       r.VenueName,
				VenueID:         r.VenueID,
				VenuePostalCode: r.VenuePostalCode,
				VenueLatitude:   r.VenueLatitude,
				VenueLongitude:  r.VenueLongitude,
				LevelMin:        r.LevelMin,
				LevelMax:        r.LevelMax,
				Fee:             r.Fee,
				Currency:        r.Currency,
				MaxPlayers:      r.MaxPlayers,
				SlotsLeft:       r.SlotsLeft,
				Courts:          r.Courts,
				Contacts:        r.Contacts,
				GenderPref:      r.GenderPref,
				Meta:            r.Meta,
			}
		}

		page := pagination.Paginate(items, input.Limit, total, func(p PlayPublic) int64 {
			return p.ID
		})

		return &ListOutput{Body: page}, nil
	})
}

func gameTypeStr(gt *model.GameType) *string {
	if gt == nil {
		return nil
	}
	s := string(*gt)
	return &s
}
