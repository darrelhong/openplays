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
	"openplays/server/internal/api/param"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/sqliteutils"
)

type ListInput struct {
	ListingType string                  `query:"listing_type" doc:"Filter by listing type" enum:"play,sell_booking,"`
	Sport       string                  `query:"sport" doc:"Filter by sport" enum:"badminton,tennis,football,pickleball,"`
	VenueID     int64                   `query:"venue_id" doc:"Filter by venue ID"`
	StartsAfter string                  `query:"starts_after" doc:"Only include plays starting on or after this date (YYYY-MM-DD)"`
	Lat         param.Optional[float64] `query:"lat" doc:"Reference latitude for distance sorting"`
	Lng         param.Optional[float64] `query:"lng" doc:"Reference longitude for distance sorting"`
	Cursor      string                  `query:"cursor" doc:"Opaque cursor from previous page"`
	Limit       int64                   `query:"limit" default:"20" minimum:"1" maximum:"100" doc:"Number of results per page"`
}

type ListOutput struct {
	Body pagination.Page[PlayPublic]
}

// --- Time-based cursor (starts_at, id) ---

// encodeTimeCursor encodes a (starts_at, id) pair into an opaque cursor string.
func encodeTimeCursor(startsAtRFC3339 string, id int64) string {
	t, err := time.Parse(time.RFC3339, startsAtRFC3339)
	if err != nil {
		return fmt.Sprintf("%s,%d", startsAtRFC3339, id)
	}
	return fmt.Sprintf("%s,%d", t.UTC().Format(time.RFC3339), id)
}

// decodeTimeCursor decodes an opaque cursor string into (starts_at, id).
func decodeTimeCursor(cursor string) (startsAt string, id int64, ok bool) {
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
	return t.UTC().Format(sqliteutils.DateTimeFormat), true
}

// --- Distance-based cursor (distance_km, id) ---

// encodeDistanceCursor encodes a (distance_km, id) pair into an opaque cursor string.
func encodeDistanceCursor(distanceKm float64, id int64) string {
	return fmt.Sprintf("%f,%d", distanceKm, id)
}

// decodeDistanceCursor decodes an opaque cursor string into (distance_km, id).
func decodeDistanceCursor(cursor string) (distance float64, id int64, ok bool) {
	if cursor == "" {
		return 0, 0, false
	}
	parts := strings.SplitN(cursor, ",", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	dist, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, false
	}
	parsed, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return dist, parsed, true
}

// --- Shared filter helpers ---

// filters holds the common nullable filter values for sqlc queries.
type filters struct {
	listingType interface{}
	sport       interface{}
	venueID     interface{}
	startsAfter interface{}
}

func buildFilters(input *ListInput) filters {
	var f filters
	if input.ListingType != "" {
		f.listingType = input.ListingType
	}
	if input.Sport != "" {
		f.sport = input.Sport
	}
	if input.VenueID != 0 {
		f.venueID = input.VenueID
	}
	if input.StartsAfter != "" {
		// Parse YYYY-MM-DD and convert to SQLite datetime at start of day UTC
		t, err := time.Parse("2006-01-02", input.StartsAfter)
		if err == nil {
			f.startsAfter = t.UTC().Format(sqliteutils.DateTimeFormat)
		}
	}
	return f
}

// sortByDistance returns true when both lat and lng query params are provided.
func (input *ListInput) sortByDistance() bool {
	return input.Lat.Set && input.Lng.Set
}

// --- Row mapping ---

func mapTimeRow(r db.ListUpcomingPlaysRow) PlayPublic {
	return PlayPublic{
		ID:               r.ID,
		CreatedAt:        r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        r.UpdatedAt.Format(time.RFC3339),
		ListingType:      r.ListingType,
		Sport:            r.Sport,
		GameType:         r.GameType,
		HostName:         r.HostName,
		StartsAt:         r.StartsAt.Format(time.RFC3339),
		EndsAt:           r.EndsAt.Format(time.RFC3339),
		Timezone:         r.Timezone,
		Venue:            r.Venue,
		VenueName:        r.VenueName,
		VenueID:          r.VenueID,
		VenuePostalCode:  r.VenuePostalCode,
		VenueLatitude:    r.VenueLatitude,
		VenueLongitude:   r.VenueLongitude,
		LevelMin:         r.LevelMin,
		LevelMax:         r.LevelMax,
		Fee:              r.Fee,
		Currency:         r.Currency,
		MaxPlayers:       r.MaxPlayers,
		SlotsLeft:        r.SlotsLeft,
		Courts:           r.Courts,
		Contacts:         r.Contacts,
		GenderPref:       r.GenderPref,
		Meta:             r.Meta,
		Source:           r.Source,
		SourceSenderLink: buildSenderLink(r.Source, r.SourceSenderUsername),
		SourceMessageID:  r.SourceMessageID,
		SourceGroup:      r.SourceGroup,
		SourceLink:       buildSourceLink(r.Source, r.SourceGroup, r.SourceMessageID),
	}
}

func mapDistanceRow(r db.ListUpcomingPlaysByDistanceRow) PlayPublic {
	return PlayPublic{
		ID:               r.ID,
		CreatedAt:        r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        r.UpdatedAt.Format(time.RFC3339),
		ListingType:      r.ListingType,
		Sport:            r.Sport,
		GameType:         r.GameType,
		HostName:         r.HostName,
		StartsAt:         r.StartsAt.Format(time.RFC3339),
		EndsAt:           r.EndsAt.Format(time.RFC3339),
		Timezone:         r.Timezone,
		Venue:            r.Venue,
		VenueName:        r.VenueName,
		VenueID:          r.VenueID,
		VenuePostalCode:  r.VenuePostalCode,
		VenueLatitude:    &r.VenueLatitude,
		VenueLongitude:   &r.VenueLongitude,
		LevelMin:         r.LevelMin,
		LevelMax:         r.LevelMax,
		Fee:              r.Fee,
		Currency:         r.Currency,
		MaxPlayers:       r.MaxPlayers,
		SlotsLeft:        r.SlotsLeft,
		Courts:           r.Courts,
		Contacts:         r.Contacts,
		GenderPref:       r.GenderPref,
		Meta:             r.Meta,
		Source:           r.Source,
		SourceSenderLink: buildSenderLink(r.Source, r.SourceSenderUsername),
		SourceMessageID:  r.SourceMessageID,
		SourceGroup:      r.SourceGroup,
		SourceLink:       buildSourceLink(r.Source, r.SourceGroup, r.SourceMessageID),
		distanceKm:       r.DistanceKm, // internal, for cursor encoding
	}
}

// --- Query dispatching ---

func listByTime(ctx context.Context, queries *db.Queries, input *ListInput, f filters) ([]PlayPublic, int64, error) {
	pageSize := input.Limit + 1

	var cursorStartsAt interface{}
	var cursorID *int64
	if startsAt, id, ok := decodeTimeCursor(input.Cursor); ok {
		if dbStartsAt, ok := cursorStartsAtForDB(startsAt); ok {
			cursorStartsAt = dbStartsAt
			cursorID = &id
		}
	}

	rows, err := queries.ListUpcomingPlays(ctx, db.ListUpcomingPlaysParams{
		ListingType:    f.listingType,
		Sport:          f.sport,
		VenueID:        f.venueID,
		StartsAfter:    f.startsAfter,
		CursorStartsAt: cursorStartsAt,
		CursorID:       cursorID,
		PageSize:       pageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list plays: %w", err)
	}

	total, err := queries.CountUpcomingPlays(ctx, db.CountUpcomingPlaysParams{
		ListingType: f.listingType,
		Sport:       f.sport,
		VenueID:     f.venueID,
		StartsAfter: f.startsAfter,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count plays: %w", err)
	}

	items := make([]PlayPublic, len(rows))
	for i, r := range rows {
		items[i] = mapTimeRow(r)
	}

	return items, total, nil
}

func listByDistance(ctx context.Context, queries *db.Queries, input *ListInput, f filters) ([]PlayPublic, int64, error) {
	pageSize := input.Limit + 1

	var cursorDistance interface{}
	var cursorID *int64
	if dist, id, ok := decodeDistanceCursor(input.Cursor); ok {
		cursorDistance = dist
		cursorID = &id
	}

	rows, err := queries.ListUpcomingPlaysByDistance(ctx, db.ListUpcomingPlaysByDistanceParams{
		RefLat:         input.Lat.Value,
		RefLng:         input.Lng.Value,
		ListingType:    f.listingType,
		Sport:          f.sport,
		VenueID:        f.venueID,
		StartsAfter:    f.startsAfter,
		CursorDistance: cursorDistance,
		CursorID:       cursorID,
		PageSize:       pageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list plays by distance: %w", err)
	}

	total, err := queries.CountUpcomingPlaysByDistance(ctx, db.CountUpcomingPlaysByDistanceParams{
		ListingType: f.listingType,
		Sport:       f.sport,
		VenueID:     f.venueID,
		StartsAfter: f.startsAfter,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count plays by distance: %w", err)
	}

	items := make([]PlayPublic, len(rows))
	for i, r := range rows {
		items[i] = mapDistanceRow(r)
	}

	return items, total, nil
}

// --- Handler ---

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

		f := buildFilters(input)

		var items []PlayPublic
		var total int64
		var err error

		if input.sortByDistance() {
			items, total, err = listByDistance(ctx, queries, input, f)
		} else {
			items, total, err = listByTime(ctx, queries, input, f)
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list plays", err)
		}

		var getCursor func(PlayPublic) string
		if input.sortByDistance() {
			getCursor = func(p PlayPublic) string {
				return encodeDistanceCursor(p.distanceKm, p.ID)
			}
		} else {
			getCursor = func(p PlayPublic) string {
				return encodeTimeCursor(p.StartsAt, p.ID)
			}
		}

		page := pagination.Paginate(items, input.Limit, total, getCursor)

		return &ListOutput{Body: page}, nil
	})
}
