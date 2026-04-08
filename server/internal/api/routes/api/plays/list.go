package plays

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
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
	Cursor  string `query:"cursor" doc:"Opaque cursor from previous page"`
	Limit   int64  `query:"limit" default:"20" minimum:"1" maximum:"100" doc:"Number of results per page"`
}

type ListOutput struct {
	Body pagination.Page[PlayPublic]
}

// sqliteTimeFormat is the format SQLite uses for datetime columns.
const sqliteTimeFormat = "2006-01-02 15:04:05+00:00"

// encodeCursor encodes a (starts_at, id) pair into an opaque cursor string.
// The cursor stays in RFC3339 format externally so API consumers never need to
// know about SQLite's internal timestamp representation.
func encodeCursor(startsAtRFC3339 string, id int64) string {
	t, err := time.Parse(time.RFC3339, startsAtRFC3339)
	if err != nil {
		return fmt.Sprintf("%s,%d", startsAtRFC3339, id)
	}
	return fmt.Sprintf("%s,%d", t.UTC().Format(time.RFC3339), id)
}

// decodeCursor decodes an opaque cursor string into (starts_at, id).
// Returns zero values if the cursor is empty or invalid.
func decodeCursor(cursor string) (startsAt string, id int64, ok bool) {
	if cursor == "" {
		return "", 0, false
	}
	parts := strings.SplitN(cursor, ",", 2)
	if len(parts) != 2 {
		return "", 0, false
	}
	parsed, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return parts[0], parsed, true
}

func cursorStartsAtForDB(startsAtRFC3339 string) (string, bool) {
	t, err := time.Parse(time.RFC3339, startsAtRFC3339)
	if err != nil {
		return "", false
	}
	return t.UTC().Format(sqliteTimeFormat), true
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

		var cursorStartsAt interface{}
		var cursorID *int64
		if startsAt, id, ok := decodeCursor(input.Cursor); ok {
			if dbStartsAt, ok := cursorStartsAtForDB(startsAt); ok {
				cursorStartsAt = dbStartsAt
				cursorID = &id
			}
		}

		rows, err := queries.ListUpcomingPlays(ctx, db.ListUpcomingPlaysParams{
			Sport:          sport,
			VenueID:        venueID,
			CursorStartsAt: cursorStartsAt,
			CursorID:       cursorID,
			PageSize:       pageSize,
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
		}

		page := pagination.Paginate(items, input.Limit, total, func(p PlayPublic) string {
			return encodeCursor(p.StartsAt, p.ID)
		})

		return &ListOutput{Body: page}, nil
	})
}
