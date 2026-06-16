package venues

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/db"
	"openplays/server/internal/geo"
)

const (
	minLocalVenueQueryRunes  = 2
	minGoogleVenueQueryRunes = 3
	minLocalVenueMatches     = 2
	maxVenueSearchLimit      = 8
)

type SearchInput struct {
	Query        string `query:"q" doc:"Venue name, address, postal code, or alias"`
	SessionToken string `query:"session_token" doc:"Google Places autocomplete session token"`
	Limit        int64  `query:"limit" default:"5" minimum:"1" maximum:"8"`
}

type SearchBody struct {
	Items []VenueSearchItem `json:"items"`
}

type SearchOutput struct {
	Body SearchBody
}

type ResolveInput struct {
	Body struct {
		GooglePlaceID string `json:"google_place_id" required:"true"`
		SessionToken  string `json:"session_token,omitempty"`
		Query         string `json:"query,omitempty"`
	}
}

type ResolveOutput struct {
	Body VenuePublic
}

func RegisterSearch(api huma.API, queries *db.Queries, places geo.PlaceProvider, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "search-venues",
		Summary:     "Search venues",
		Description: "Returns local venue matches first. Google Places is queried only when fewer than two local matches exist.",
		Method:      http.MethodGet,
		Path:        "/search",
		Tags:        []string{"Venues"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *SearchInput) (*SearchOutput, error) {
		query := strings.TrimSpace(input.Query)
		if len([]rune(query)) < minLocalVenueQueryRunes {
			return &SearchOutput{Body: SearchBody{Items: []VenueSearchItem{}}}, nil
		}

		limit := input.Limit
		if limit <= 0 || limit > maxVenueSearchLimit {
			limit = 5
		}
		localQueryLimit := limit
		if localQueryLimit < minLocalVenueMatches {
			localQueryLimit = minLocalVenueMatches
		}

		local, err := queries.SearchVenues(ctx, db.SearchVenuesParams{
			Query: query,
			Limit: localQueryLimit,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to search venues", err)
		}
		items := venueSearchItemsFromRows(local)
		if int64(len(items)) > limit {
			items = items[:limit]
		}
		if len(local) >= minLocalVenueMatches || places == nil || len([]rune(query)) < minGoogleVenueQueryRunes || int64(len(items)) >= limit {
			return &SearchOutput{Body: SearchBody{Items: items}}, nil
		}

		suggestions, err := places.Autocomplete(ctx, query, strings.TrimSpace(input.SessionToken))
		if err != nil {
			return nil, huma.Error502BadGateway("failed to search Google Places", err)
		}
		items = append(items, venueSearchItemsFromSuggestions(suggestions, limit-int64(len(items)), googlePlaceIDsFromItems(items))...)
		return &SearchOutput{Body: SearchBody{Items: items}}, nil
	})
}

func RegisterResolve(api huma.API, queries *db.Queries, places geo.PlaceProvider, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "resolve-google-venue",
		Summary:     "Resolve a Google venue",
		Description: "Stores a selected Google Places result as a local venue and returns the saved venue.",
		Method:      http.MethodPost,
		Path:        "/resolve-google",
		Tags:        []string{"Venues"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *ResolveInput) (*ResolveOutput, error) {
		placeID := strings.TrimSpace(input.Body.GooglePlaceID)
		if placeID == "" {
			return nil, huma.Error422UnprocessableEntity("google_place_id is required")
		}

		if venue, err := queries.GetVenueByGooglePlaceID(ctx, &placeID); err == nil {
			return &ResolveOutput{Body: venuePublicFromVenue(venue)}, nil
		} else if err != sql.ErrNoRows {
			return nil, huma.Error500InternalServerError("failed to lookup venue", err)
		}
		if places == nil {
			return nil, huma.Error503ServiceUnavailable("Google Places is not configured")
		}

		details, err := places.PlaceDetails(ctx, placeID, strings.TrimSpace(input.Body.SessionToken))
		if err != nil {
			return nil, huma.Error502BadGateway("failed to resolve Google Place", err)
		}
		if details == nil || strings.TrimSpace(details.Name) == "" || strings.TrimSpace(details.PlaceID) == "" {
			return nil, huma.Error502BadGateway("Google Place response was missing venue details")
		}

		venue, err := upsertGoogleVenue(ctx, queries, details, strings.TrimSpace(input.Body.Query))
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to store venue", err)
		}
		return &ResolveOutput{Body: venuePublicFromVenue(venue)}, nil
	})
}

func upsertGoogleVenue(ctx context.Context, queries *db.Queries, details *geo.Result, query string) (db.Venue, error) {
	placeID := strings.TrimSpace(details.PlaceID)
	name := strings.TrimSpace(details.Name)
	address := strings.TrimSpace(details.Address)

	var postalCode *string
	if postal := strings.TrimSpace(details.Postal); postal != "" {
		postalCode = &postal
	}
	searchTerm := query
	if searchTerm == "" {
		searchTerm = name
	}

	venue, err := queries.UpsertVenueByGooglePlaceID(ctx, db.UpsertVenueByGooglePlaceIDParams{
		GooglePlaceID: &placeID,
		PostalCode:    postalCode,
		Name:          name,
		Address:       address,
		Latitude:      details.Latitude,
		Longitude:     details.Longitude,
		Source:        "google",
		SearchTerm:    &searchTerm,
	})
	if err != nil {
		return db.Venue{}, err
	}

	cacheVenueAlias(ctx, queries, name, venue.ID)
	cacheVenueAlias(ctx, queries, query, venue.ID)
	return venue, nil
}

func cacheVenueAlias(ctx context.Context, queries *db.Queries, alias string, venueID int64) {
	alias = strings.ToLower(strings.TrimSpace(alias))
	if alias == "" {
		return
	}
	_ = queries.UpsertVenueAlias(ctx, db.UpsertVenueAliasParams{
		Alias:   alias,
		VenueID: venueID,
	})
}

func venueSearchItemsFromRows(rows []db.SearchVenuesRow) []VenueSearchItem {
	items := make([]VenueSearchItem, len(rows))
	for i, row := range rows {
		id := row.ID
		lat := row.Latitude
		lng := row.Longitude
		items[i] = VenueSearchItem{
			ID:            &id,
			Name:          row.Name,
			Address:       row.Address,
			PostalCode:    row.PostalCode,
			Latitude:      &lat,
			Longitude:     &lng,
			GooglePlaceID: row.GooglePlaceID,
		}
	}
	return items
}

func venueSearchItemsFromSuggestions(suggestions []geo.Suggestion, limit int64, seenPlaceIDs map[string]struct{}) []VenueSearchItem {
	if limit <= 0 {
		return []VenueSearchItem{}
	}
	items := make([]VenueSearchItem, 0, len(suggestions))
	for _, suggestion := range suggestions {
		placeID := strings.TrimSpace(suggestion.PlaceID)
		if placeID == "" {
			continue
		}
		if _, ok := seenPlaceIDs[placeID]; ok {
			continue
		}
		items = append(items, VenueSearchItem{
			Name:          suggestion.Name,
			Address:       suggestion.Address,
			GooglePlaceID: &placeID,
		})
		if int64(len(items)) >= limit {
			break
		}
	}
	return items
}

func googlePlaceIDsFromItems(items []VenueSearchItem) map[string]struct{} {
	ids := make(map[string]struct{})
	for _, item := range items {
		if item.GooglePlaceID == nil {
			continue
		}
		if placeID := strings.TrimSpace(*item.GooglePlaceID); placeID != "" {
			ids[placeID] = struct{}{}
		}
	}
	return ids
}

func venuePublicFromVenue(venue db.Venue) VenuePublic {
	postalCode := ""
	if venue.PostalCode != nil {
		postalCode = *venue.PostalCode
	}
	return VenuePublic{
		ID:            venue.ID,
		Name:          venue.Name,
		Address:       venue.Address,
		PostalCode:    postalCode,
		Latitude:      venue.Latitude,
		Longitude:     venue.Longitude,
		GooglePlaceID: venue.GooglePlaceID,
	}
}
