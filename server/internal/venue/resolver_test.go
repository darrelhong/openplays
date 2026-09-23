package venue

import (
	"context"
	"database/sql"
	"testing"

	"openplays/server/internal/db"
)

// stubStore is a minimal in-memory store for testing the Resolver.
type stubStore struct {
	aliases map[string]db.Venue // alias → venue
	venues  []db.ListVenueNamesRow
	upserts []db.UpsertVenueAliasParams
}

func (s *stubStore) GetVenueByAlias(_ context.Context, alias string) (db.Venue, error) {
	if v, ok := s.aliases[alias]; ok {
		return v, nil
	}
	return db.Venue{}, sql.ErrNoRows
}

func (s *stubStore) UpsertVenue(_ context.Context, arg db.UpsertVenueParams) (db.Venue, error) {
	return db.Venue{ID: 99, PostalCode: arg.PostalCode, Name: arg.Name}, nil
}

func (s *stubStore) UpsertVenueByGooglePlaceID(_ context.Context, arg db.UpsertVenueByGooglePlaceIDParams) (db.Venue, error) {
	return db.Venue{ID: 99, PostalCode: arg.PostalCode, Name: arg.Name, GooglePlaceID: arg.GooglePlaceID}, nil
}

func (s *stubStore) UpsertVenueAlias(_ context.Context, arg db.UpsertVenueAliasParams) error {
	s.upserts = append(s.upserts, arg)
	return nil
}

func (s *stubStore) ListVenueNames(_ context.Context) ([]db.ListVenueNamesRow, error) {
	return s.venues, nil
}

func TestResolve_ExactAlias(t *testing.T) {
	store := &stubStore{
		aliases: map[string]db.Venue{
			"hougang cc": {ID: 1, Name: "Hougang Community Club"},
		},
	}
	r := NewResolver(store, nil)
	raw := "Hougang CC"
	got := r.Resolve(context.Background(), &raw)
	if got == nil || got.ID != 1 {
		t.Fatalf("Resolve(%q) = %v, want id 1", raw, got)
	}
}

func TestResolve_ExpandedAlias(t *testing.T) {
	store := &stubStore{
		aliases: map[string]db.Venue{
			"hougang community club": {ID: 1, Name: "Hougang Community Club"},
		},
	}
	r := NewResolver(store, nil)
	raw := "Hougang CC"
	got := r.Resolve(context.Background(), &raw)
	if got == nil || got.ID != 1 {
		t.Fatalf("Resolve(%q) = %v, want id 1", raw, got)
	}
	if len(store.upserts) != 1 || store.upserts[0].Alias != "hougang cc" {
		t.Errorf("expected alias 'hougang cc' to be cached, got %v", store.upserts)
	}
}

func TestResolve_ExpandedAlias_SecSch(t *testing.T) {
	store := &stubStore{
		venues: []db.ListVenueNamesRow{
			{ID: 9, Name: "Hougang Secondary School"},
		},
	}
	r := NewResolver(store, nil)
	raw := "Hougang Sec"
	got := r.Resolve(context.Background(), &raw)
	if got == nil || got.ID != 9 {
		t.Fatalf("Resolve(%q) = %v, want id 9", raw, got)
	}
	if len(store.upserts) != 1 || store.upserts[0].Alias != "hougang sec" {
		t.Errorf("expected alias 'hougang sec' to be cached, got %v", store.upserts)
	}
}

func TestResolve_FuzzyMatch(t *testing.T) {
	store := &stubStore{
		venues: []db.ListVenueNamesRow{
			{ID: 5, Name: "Bukit Canberra Sports Hall"},
		},
	}
	r := NewResolver(store, nil)
	raw := "Canberra Sport Hall"
	got := r.Resolve(context.Background(), &raw)
	if got == nil || got.ID != 5 {
		t.Fatalf("Resolve(%q) = %v, want id 5", raw, got)
	}
}

// TestResolve_AmbiguousFuzzyMatchNotCached is the regression test for the
// mis-filing. When several venues fit the input equally well the old code
// cached whichever row came first, permanently pinning the raw string to an
// arbitrary venue. An unresolved tie must not become an alias.
//
// The fixture mirrors real data: the venues table genuinely contains duplicate
// rows for generic locations ("Simei" exists twice), so a bare "Simei" is
// ambiguous and must not be guessed.
func TestResolve_AmbiguousFuzzyMatchNotCached(t *testing.T) {
	store := &stubStore{
		venues: []db.ListVenueNamesRow{
			{ID: 44, Name: "Simei"},
			{ID: 239, Name: "Simei"},
		},
	}
	r := NewResolver(store, nil) // nil geocoder: a tie falls through to unresolved
	raw := "Simei"

	got := r.Resolve(context.Background(), &raw)
	if got != nil {
		t.Errorf("Resolve(%q) = %v, want nil (ambiguous, must not guess)", raw, got)
	}
	for _, u := range store.upserts {
		if u.Alias == "simei" {
			t.Errorf("ambiguous match was cached as an alias: %v", u)
		}
	}
}

// TestResolve_UniqueFuzzyMatchIsCached is the counterpart: a clear winner is
// still cached, so we do not lose speed on the cases the matcher gets right.
func TestResolve_UniqueFuzzyMatchIsCached(t *testing.T) {
	store := &stubStore{
		venues: []db.ListVenueNamesRow{
			{ID: 30, Name: "Sengkang Sport Hall"},
			{ID: 99, Name: "Clementi Sport Hall"},
		},
	}
	r := NewResolver(store, nil)
	raw := "SENGKANG S/HALL"

	got := r.Resolve(context.Background(), &raw)
	if got == nil || got.ID != 30 {
		t.Fatalf("Resolve(%q) = %v, want id 30", raw, got)
	}
	var cached bool
	for _, u := range store.upserts {
		if u.Alias == "sengkang s/hall" && u.VenueID == 30 {
			cached = true
		}
	}
	if !cached {
		t.Errorf("expected alias 'sengkang s/hall' -> 30, upserts=%v", store.upserts)
	}
}

func TestResolve_Unresolved(t *testing.T) {
	store := &stubStore{}
	r := NewResolver(store, nil)
	raw := "SBH"
	got := r.Resolve(context.Background(), &raw)
	if got != nil {
		t.Errorf("Resolve(%q) = %v, want nil", raw, got)
	}
}

func TestResolve_Nil(t *testing.T) {
	r := NewResolver(&stubStore{}, nil)
	if got := r.Resolve(context.Background(), nil); got != nil {
		t.Errorf("Resolve(nil) = %v, want nil", got)
	}
	empty := ""
	if got := r.Resolve(context.Background(), &empty); got != nil {
		t.Errorf("Resolve('') = %v, want nil", got)
	}
}
