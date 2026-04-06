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
	return db.Venue{PostalCode: arg.PostalCode, Name: arg.Name}, nil
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
			"hougang cc": {PostalCode: "538840", Name: "Hougang Community Club"},
		},
	}
	r := NewResolver(store, nil)
	raw := "Hougang CC"
	got := r.Resolve(context.Background(), &raw)
	if got == nil || got.PostalCode != "538840" {
		t.Fatalf("Resolve(%q) = %v, want postal 538840", raw, got)
	}
}

func TestResolve_ExpandedAlias(t *testing.T) {
	// "hougang cc" is NOT an alias, but "hougang community club" IS.
	// The resolver should expand "hougang cc" → "hougang community club",
	// find the alias, and auto-add "hougang cc" as a new alias.
	store := &stubStore{
		aliases: map[string]db.Venue{
			"hougang community club": {PostalCode: "538840", Name: "Hougang Community Club"},
		},
	}
	r := NewResolver(store, nil)
	raw := "Hougang CC"
	got := r.Resolve(context.Background(), &raw)
	if got == nil || got.PostalCode != "538840" {
		t.Fatalf("Resolve(%q) = %v, want postal 538840", raw, got)
	}
	// Verify the original alias was cached
	if len(store.upserts) != 1 || store.upserts[0].Alias != "hougang cc" {
		t.Errorf("expected alias 'hougang cc' to be cached, got %v", store.upserts)
	}
}

func TestResolve_ExpandedAlias_SecSch(t *testing.T) {
	// "hougang sec" expands to "hougang secondary", which is looked up as alias.
	// If that fails, fuzzy match against venue names should catch it.
	store := &stubStore{
		venues: []db.ListVenueNamesRow{
			{PostalCode: "530540", Name: "Hougang Secondary School"},
		},
	}
	r := NewResolver(store, nil)
	raw := "Hougang Sec"
	got := r.Resolve(context.Background(), &raw)
	if got == nil || got.PostalCode != "530540" {
		t.Fatalf("Resolve(%q) = %v, want postal 530540", raw, got)
	}
	// Verify alias was cached
	if len(store.upserts) != 1 || store.upserts[0].Alias != "hougang sec" {
		t.Errorf("expected alias 'hougang sec' to be cached, got %v", store.upserts)
	}
}

func TestResolve_FuzzyMatch(t *testing.T) {
	// "Canberra Sport Hall" should fuzzy match "Bukit Canberra Sports Hall"
	store := &stubStore{
		venues: []db.ListVenueNamesRow{
			{PostalCode: "757716", Name: "Bukit Canberra Sports Hall"},
		},
	}
	r := NewResolver(store, nil)
	raw := "Canberra Sport Hall"
	got := r.Resolve(context.Background(), &raw)
	if got == nil || got.PostalCode != "757716" {
		t.Fatalf("Resolve(%q) = %v, want postal 757716", raw, got)
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
