package venue

import (
	"context"
	"database/sql"
	"log"
	"strings"

	"openplays/server/internal/db"
	"openplays/server/internal/geo"
)

// Resolved holds the venue data needed by the caller.
type Resolved struct {
	PostalCode string
	Name       string
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
		return &Resolved{PostalCode: v.PostalCode, Name: v.Name}
	} else if err != sql.ErrNoRows {
		log.Printf("venue: alias lookup error: %v", err)
	}

	// 2. Expanded alias lookup
	expanded := ExpandAndNormalise(*rawVenue)
	if expanded != alias {
		if v, err := r.store.GetVenueByAlias(ctx, expanded); err == nil {
			r.upsertAlias(ctx, alias, v.PostalCode)
			return &Resolved{PostalCode: v.PostalCode, Name: v.Name}
		}
	}

	// 3. Fuzzy match against all venue names
	if candidates := r.loadCandidates(ctx); len(candidates) > 0 {
		if m := FuzzyMatch(*rawVenue, candidates); m != nil {
			log.Printf("venue: fuzzy matched %q → %s (%s) [score=%.0f%%]",
				*rawVenue, m.Name, m.PostalCode, m.Score*100)
			r.upsertAlias(ctx, alias, m.PostalCode)
			return &Resolved{PostalCode: m.PostalCode, Name: m.Name}
		}
	}

	// 4. Geocoder fallback
	if r.geocoder == nil {
		return nil
	}

	result, err := r.geocoder.Geocode(ctx, *rawVenue)
	if err != nil {
		log.Printf("venue: geocode error for %q: %v", *rawVenue, err)
		return nil
	}
	if result == nil {
		log.Printf("venue: geocode no results for %q", *rawVenue)
		return nil
	}

	searchTerm := *rawVenue
	v, err := r.store.UpsertVenue(ctx, db.UpsertVenueParams{
		PostalCode: result.Postal,
		Name:       result.Name,
		Address:    result.Address,
		Latitude:   result.Latitude,
		Longitude:  result.Longitude,
		Source:     result.Source,
		SearchTerm: &searchTerm,
	})
	if err != nil {
		log.Printf("venue: error upserting venue %q: %v", result.Postal, err)
		return nil
	}

	r.upsertAlias(ctx, alias, v.PostalCode)
	log.Printf("venue: geocoded %q → %s (%s)", *rawVenue, v.Name, v.PostalCode)

	return &Resolved{PostalCode: v.PostalCode, Name: v.Name}
}

func (r *Resolver) loadCandidates(ctx context.Context) []Candidate {
	rows, err := r.store.ListVenueNames(ctx)
	if err != nil {
		log.Printf("venue: error listing venue names: %v", err)
		return nil
	}
	candidates := make([]Candidate, len(rows))
	for i, row := range rows {
		candidates[i] = Candidate{PostalCode: row.PostalCode, Name: row.Name}
	}
	return candidates
}

func (r *Resolver) upsertAlias(ctx context.Context, alias, postalCode string) {
	if err := r.store.UpsertVenueAlias(ctx, db.UpsertVenueAliasParams{
		Alias:           alias,
		VenuePostalCode: postalCode,
	}); err != nil {
		log.Printf("venue: error upserting alias %q: %v", alias, err)
	}
}
