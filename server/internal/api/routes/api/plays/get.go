package plays

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/db"
)

type GetInput struct {
	ID int64 `path:"id" doc:"Play ID"`
}

type GetOutput struct {
	Body PlayPublic
}

func RegisterGet(api huma.API, queries *db.Queries) {
	huma.Register(api, huma.Operation{
		OperationID: "get-play",
		Summary:     "Get a play by ID",
		Method:      http.MethodGet,
		Path:        "/{id}",
		Tags:        []string{"Plays"},
	}, func(ctx context.Context, input *GetInput) (*GetOutput, error) {
		r, err := queries.GetPlayByID(ctx, input.ID)
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("play not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get play", err)
		}

		item := PlayPublic{
			ID:              r.ID,
			CreatedAt:       r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:       r.UpdatedAt.Format(time.RFC3339),
			ListingType:     r.ListingType,
			Sport:           r.Sport,
			GameType:        r.GameType,
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
			Source:          r.Source,
			SourceMessageID: r.SourceMessageID,
			SourceGroup:     r.SourceGroup,
			SourceLink:      buildSourceLink(r.Source, r.SourceGroup, r.SourceMessageID),
		}

		return &GetOutput{Body: item}, nil
	})
}
