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

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/pagination"
	"openplays/server/internal/api/param"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/sqliteutils"
)

type ListInput struct {
	ListingType  string                  `query:"listing_type" doc:"Filter by listing type" enum:"play,sell_booking,"`
	Sport        string                  `query:"sport" doc:"Filter by sport" enum:"badminton,tennis,football,pickleball,"`
	VenueID      int64                   `query:"venue_id" doc:"Filter by venue ID"`
	LevelMin     string                  `query:"level_min" doc:"Minimum level code (e.g. HB). Shows plays overlapping this range."`
	LevelMax     string                  `query:"level_max" doc:"Maximum level code (e.g. LI). Defaults to level_min if only level_min is set."`
	StartsAfter  string                  `query:"starts_after" doc:"Only include plays starting on or after this date (YYYY-MM-DD)"`
	StartsBefore string                  `query:"starts_before" doc:"Only include plays starting on or before this date (YYYY-MM-DD)"`
	Timezone     string                  `query:"timezone" doc:"IANA timezone for date filters, e.g. Asia/Singapore. Defaults to UTC."`
	Lat          param.Optional[float64] `query:"lat" doc:"Reference latitude for distance sorting"`
	Lng          param.Optional[float64] `query:"lng" doc:"Reference longitude for distance sorting"`
	Cursor       string                  `query:"cursor" doc:"Opaque cursor from previous page"`
	Limit        int64                   `query:"limit" default:"20" minimum:"1" maximum:"100" doc:"Number of results per page"`
}

type ListOutput struct {
	Body pagination.Page[PlayPublic]
}

// --- Time-based cursor (starts_at, id) ---

// encodeTimeCursor encodes a (starts_at, id) pair into an opaque cursor string.
func encodeTimeCursor(startsAtRFC3339 string, id string) string {
	t, err := time.Parse(time.RFC3339, startsAtRFC3339)
	if err != nil {
		return fmt.Sprintf("%s,%s", startsAtRFC3339, id)
	}
	return fmt.Sprintf("%s,%s", t.UTC().Format(time.RFC3339), id)
}

// decodeTimeCursor decodes an opaque cursor string into (starts_at, id).
func decodeTimeCursor(cursor string) (startsAt string, id string, ok bool) {
	if cursor == "" {
		return "", "", false
	}
	parts := strings.SplitN(cursor, ",", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
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
func encodeDistanceCursor(distanceKm float64, id string) string {
	return fmt.Sprintf("%f,%s", distanceKm, id)
}

// decodeDistanceCursor decodes an opaque cursor string into (distance_km, id).
func decodeDistanceCursor(cursor string) (distance float64, id string, ok bool) {
	if cursor == "" {
		return 0, "", false
	}
	parts := strings.SplitN(cursor, ",", 2)
	if len(parts) != 2 || parts[1] == "" {
		return 0, "", false
	}
	dist, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, "", false
	}
	return dist, parts[1], true
}

// --- Shared filter helpers ---

// filters holds the common nullable filter values for sqlc queries.
type filters struct {
	listingType       interface{}
	sport             interface{}
	venueID           interface{}
	startsAfter       interface{}
	startsBefore      interface{}
	filterLevelMinOrd interface{}
	filterLevelMaxOrd interface{}
}

func dateToUTCBound(dateStr string, loc *time.Location, isExclusiveEnd bool) (string, bool) {
	t, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return "", false
	}
	if isExclusiveEnd {
		t = t.AddDate(0, 0, 1)
	}
	return t.UTC().Format(sqliteutils.DateTimeFormat), true
}

func buildFilters(input *ListInput) filters {
	var f filters
	tz := strings.TrimSpace(input.Timezone)
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

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
		if bound, ok := dateToUTCBound(input.StartsAfter, loc, false); ok {
			f.startsAfter = bound
		}
	}
	if input.StartsBefore != "" {
		if bound, ok := dateToUTCBound(input.StartsBefore, loc, true); ok {
			f.startsBefore = bound
		}
	}
	if input.LevelMin != "" {
		sport := model.SportBadminton
		if input.Sport != "" {
			sport = model.Sport(input.Sport)
		}
		if ord := model.LevelOrd(sport, input.LevelMin); ord != nil {
			f.filterLevelMinOrd = *ord
		}
	}
	if input.LevelMax != "" {
		sport := model.SportBadminton
		if input.Sport != "" {
			sport = model.Sport(input.Sport)
		}
		if ord := model.LevelOrd(sport, input.LevelMax); ord != nil {
			f.filterLevelMaxOrd = *ord
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
	createdAt, updatedAt := publicPlayTimestamps(r.CreatedBy, r.CreatedAt, r.UpdatedAt)
	return PlayPublic{
		ID:                 r.ID,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		ListingType:        r.ListingType,
		Sport:              r.Sport,
		GameType:           r.GameType,
		HostName:           r.HostName,
		Name:               r.Name,
		Description:        r.Description,
		StartsAt:           r.StartsAt.Format(time.RFC3339),
		EndsAt:             r.EndsAt.Format(time.RFC3339),
		Timezone:           r.Timezone,
		CancelledAt:        publicOptionalTimestamp(r.CancelledAt),
		Venue:              r.Venue,
		VenueName:          r.VenueName,
		VenueID:            r.VenueID,
		VenuePostalCode:    r.VenuePostalCode,
		VenueLatitude:      r.VenueLatitude,
		VenueLongitude:     r.VenueLongitude,
		VenueGooglePlaceID: r.VenueGooglePlaceID,
		LevelMin:           r.LevelMin,
		LevelMax:           r.LevelMax,
		Fee:                r.Fee,
		Currency:           r.Currency,
		MaxPlayers:         r.MaxPlayers,
		SlotsLeft:          r.SlotsLeft,
		Courts:             r.Courts,
		Contacts:           r.Contacts,
		GenderPref:         r.GenderPref,
		Meta:               r.Meta,
		Source:             r.Source,
		SourceSenderLink:   buildSenderLink(r.Source, r.SourceSenderUsername),
		SourceMessageID:    r.SourceMessageID,
		SourceGroup:        r.SourceGroup,
		SourceLink:         buildSourceLink(r.Source, r.SourceGroup, r.SourceMessageID),
		CreatedBy:          r.CreatedBy,
		CreatorDisplayName: r.CreatorDisplayName,
		CreatorUsername:    r.CreatorUsername,
		CreatorPhotoURL:    r.CreatorPhotoUrl,
	}
}

func mapDistanceRow(r db.ListUpcomingPlaysByDistanceRow) PlayPublic {
	createdAt, updatedAt := publicPlayTimestamps(r.CreatedBy, r.CreatedAt, r.UpdatedAt)
	return PlayPublic{
		ID:                 r.ID,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		ListingType:        r.ListingType,
		Sport:              r.Sport,
		GameType:           r.GameType,
		HostName:           r.HostName,
		Name:               r.Name,
		Description:        r.Description,
		StartsAt:           r.StartsAt.Format(time.RFC3339),
		EndsAt:             r.EndsAt.Format(time.RFC3339),
		Timezone:           r.Timezone,
		CancelledAt:        publicOptionalTimestamp(r.CancelledAt),
		Venue:              r.Venue,
		VenueName:          r.VenueName,
		VenueID:            r.VenueID,
		VenuePostalCode:    r.VenuePostalCode,
		VenueLatitude:      &r.VenueLatitude,
		VenueLongitude:     &r.VenueLongitude,
		VenueGooglePlaceID: r.VenueGooglePlaceID,
		LevelMin:           r.LevelMin,
		LevelMax:           r.LevelMax,
		Fee:                r.Fee,
		Currency:           r.Currency,
		MaxPlayers:         r.MaxPlayers,
		SlotsLeft:          r.SlotsLeft,
		Courts:             r.Courts,
		Contacts:           r.Contacts,
		GenderPref:         r.GenderPref,
		Meta:               r.Meta,
		Source:             r.Source,
		SourceSenderLink:   buildSenderLink(r.Source, r.SourceSenderUsername),
		SourceMessageID:    r.SourceMessageID,
		SourceGroup:        r.SourceGroup,
		SourceLink:         buildSourceLink(r.Source, r.SourceGroup, r.SourceMessageID),
		CreatedBy:          r.CreatedBy,
		CreatorDisplayName: r.CreatorDisplayName,
		CreatorUsername:    r.CreatorUsername,
		CreatorPhotoURL:    r.CreatorPhotoUrl,
		distanceKm:         r.DistanceKm,
	}
}

// --- Query dispatching ---

func listByTime(ctx context.Context, queries *db.Queries, input *ListInput, f filters) ([]PlayPublic, int64, error) {
	pageSize := input.Limit + 1

	var cursorStartsAt interface{}
	var cursorID *string
	if startsAt, id, ok := decodeTimeCursor(input.Cursor); ok {
		if dbStartsAt, ok := cursorStartsAtForDB(startsAt); ok {
			cursorStartsAt = dbStartsAt
			cursorID = &id
		}
	}

	rows, err := queries.ListUpcomingPlays(ctx, db.ListUpcomingPlaysParams{
		ListingType:       f.listingType,
		Sport:             f.sport,
		VenueID:           f.venueID,
		StartsAfter:       f.startsAfter,
		StartsBefore:      f.startsBefore,
		FilterLevelMinOrd: f.filterLevelMinOrd,
		FilterLevelMaxOrd: f.filterLevelMaxOrd,
		CursorStartsAt:    cursorStartsAt,
		CursorID:          cursorID,
		PageSize:          pageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list plays: %w", err)
	}

	total, err := queries.CountUpcomingPlays(ctx, db.CountUpcomingPlaysParams{
		ListingType:       f.listingType,
		Sport:             f.sport,
		VenueID:           f.venueID,
		StartsAfter:       f.startsAfter,
		StartsBefore:      f.startsBefore,
		FilterLevelMinOrd: f.filterLevelMinOrd,
		FilterLevelMaxOrd: f.filterLevelMaxOrd,
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
	var cursorID *string
	if dist, id, ok := decodeDistanceCursor(input.Cursor); ok {
		cursorDistance = dist
		cursorID = &id
	}

	rows, err := queries.ListUpcomingPlaysByDistance(ctx, db.ListUpcomingPlaysByDistanceParams{
		RefLat:            input.Lat.Value,
		RefLng:            input.Lng.Value,
		ListingType:       f.listingType,
		Sport:             f.sport,
		VenueID:           f.venueID,
		StartsAfter:       f.startsAfter,
		StartsBefore:      f.startsBefore,
		FilterLevelMinOrd: f.filterLevelMinOrd,
		FilterLevelMaxOrd: f.filterLevelMaxOrd,
		CursorDistance:    cursorDistance,
		CursorID:          cursorID,
		PageSize:          pageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list plays by distance: %w", err)
	}

	total, err := queries.CountUpcomingPlaysByDistance(ctx, db.CountUpcomingPlaysByDistanceParams{
		ListingType:       f.listingType,
		Sport:             f.sport,
		VenueID:           f.venueID,
		StartsAfter:       f.startsAfter,
		StartsBefore:      f.startsBefore,
		FilterLevelMinOrd: f.filterLevelMinOrd,
		FilterLevelMaxOrd: f.filterLevelMaxOrd,
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

func RegisterList(api huma.API, queries *db.Queries, optionalAuthMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "list-plays",
		Summary:     "List upcoming plays",
		Method:      http.MethodGet,
		Path:        "/",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{optionalAuthMiddleware},
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
		if viewer := authmw.UserFromContext(ctx); viewer != nil {
			if err := hydrateViewerStates(ctx, queries, page.Items, viewer.ID); err != nil {
				return nil, huma.Error500InternalServerError("failed to list viewer states", err)
			}
			if err := hydrateFavouriteStates(ctx, queries, page.Items, viewer.ID); err != nil {
				return nil, huma.Error500InternalServerError("failed to list favourite states", err)
			}
		}
		if err := hydrateParticipantPreviews(ctx, queries, page.Items, false); err != nil {
			return nil, huma.Error500InternalServerError("failed to list participant previews", err)
		}

		return &ListOutput{Body: page}, nil
	})
}
