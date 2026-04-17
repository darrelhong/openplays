package venue

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"

	"openplays/server/internal/db"
	"openplays/server/internal/geo"
)

// Resolved holds the venue data needed by the caller.
type Resolved struct {
	ID   int64
	Name string
}

// Store is the subset of db.Queries that the Resolver needs.
type Store interface {
	GetVenueByAlias(ctx context.Context, alias string) (db.Venue, error)
	UpsertVenue(ctx context.Context, arg db.UpsertVenueParams) (db.Venue, error)
	UpsertVenueAlias(ctx context.Context, arg db.UpsertVenueAliasParams) error
	ListVenueNames(ctx context.Context) ([]db.ListVenueNamesRow, error)
}

// Resolver resolves raw venue strings to canonical venues using alias
// lookups, abbreviation expansion, fuzzy matching, and geocoder fallback.
// See the package doc for the full resolution flow.
type Resolver struct {
	store    Store
	geocoder geo.Coder // nil to disable geocoding
}

// NewResolver creates a new venue resolver. Pass nil for geocoder to
// disable the geocoder fallback (venue resolution still works via
// aliases and fuzzy matching).
func NewResolver(store Store, geocoder geo.Coder) *Resolver {
	return &Resolver{store: store, geocoder: geocoder}
}

// Resolve resolves a raw venue string to a canonical venue.
// Returns nil if the venue cannot be resolved.
func (r *Resolver) Resolve(ctx context.Context, rawVenue *string) *Resolved {
	if rawVenue == nil || *rawVenue == "" {
		return nil
	}

	alias := strings.ToLower(strings.TrimSpace(*rawVenue))

	// 1. Exact alias lookup
	if v, err := r.store.GetVenueByAlias(ctx, alias); err == nil {
		return &Resolved{ID: v.ID, Name: v.Name}
	} else if err != sql.ErrNoRows {
		slog.Error("alias lookup error", "error", err)
	}

	// 2. Expanded alias lookup
	expanded := ExpandAndNormalise(*rawVenue)
	if expanded != alias {
		if v, err := r.store.GetVenueByAlias(ctx, expanded); err == nil {
			r.upsertAlias(ctx, alias, v.ID)
			return &Resolved{ID: v.ID, Name: v.Name}
		}
	}

	// 3. Fuzzy match against all venue names
	if candidates := r.loadCandidates(ctx); len(candidates) > 0 {
		if m := FuzzyMatch(*rawVenue, candidates); m != nil {
			slog.Info("venue fuzzy matched", "raw", *rawVenue, "matched", m.Name, "venue_id", m.ID, "score_pct", m.Score*100)
			r.upsertAlias(ctx, alias, m.ID)
			return &Resolved{ID: m.ID, Name: m.Name}
		}
	}

	// 4. Geocoder fallback
	if r.geocoder == nil {
		return nil
	}

	result, err := r.geocoder.Geocode(ctx, *rawVenue)
	if err != nil {
		slog.Error("geocode error", "venue", *rawVenue, "error", err)
		return nil
	}
	if result == nil {
		slog.Warn("geocode no results", "venue", *rawVenue)
		return nil
	}

	searchTerm := *rawVenue
	var postalCode *string
	if result.Postal != "" {
		postalCode = &result.Postal
	}

	v, err := r.store.UpsertVenue(ctx, db.UpsertVenueParams{
		PostalCode: postalCode,
		Name:       result.Name,
		Address:    result.Address,
		Latitude:   result.Latitude,
		Longitude:  result.Longitude,
		Source:     result.Source,
		SearchTerm: &searchTerm,
	})
	if err != nil {
		slog.Error("error upserting venue", "error", err)
		return nil
	}

	r.upsertAlias(ctx, alias, v.ID)
	slog.Info("venue geocoded", "raw", *rawVenue, "name", v.Name, "venue_id", v.ID)

	return &Resolved{ID: v.ID, Name: v.Name}
}

func (r *Resolver) loadCandidates(ctx context.Context) []Candidate {
	rows, err := r.store.ListVenueNames(ctx)
	if err != nil {
		slog.Error("error listing venue names", "error", err)
		return nil
	}
	candidates := make([]Candidate, len(rows))
	for i, row := range rows {
		candidates[i] = Candidate{ID: row.ID, Name: row.Name}
	}
	return candidates
}

func (r *Resolver) upsertAlias(ctx context.Context, alias string, venueID int64) {
	if err := r.store.UpsertVenueAlias(ctx, db.UpsertVenueAliasParams{
		Alias:   alias,
		VenueID: venueID,
	}); err != nil {
		slog.Error("error upserting alias", "alias", alias, "error", err)
	}
}
