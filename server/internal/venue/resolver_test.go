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
