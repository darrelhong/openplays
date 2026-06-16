package venues_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/venues"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/geo"
	"openplays/server/internal/testdb"
)

type fakePlaceProvider struct {
	autocompleteCalls int
	detailsCalls      int
	suggestions       []geo.Suggestion
	details           *geo.Result
}

func (f *fakePlaceProvider) Autocomplete(_ context.Context, _ string, _ string) ([]geo.Suggestion, error) {
	f.autocompleteCalls++
	return f.suggestions, nil
}

func (f *fakePlaceProvider) PlaceDetails(_ context.Context, _ string, _ string) (*geo.Result, error) {
	f.detailsCalls++
	return f.details, nil
}

func setupVenueSearchTest(t *testing.T, queries *db.Queries, places geo.PlaceProvider) *httptest.Server {
	t.Helper()

	svc := auth.NewService(queries)
	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/venues")
	authMiddleware := authmw.RequireAuth(api, svc)
	venues.RegisterSearch(grp, queries, places, authMiddleware)
	venues.RegisterResolve(grp, queries, places, authMiddleware)
	return httptest.NewServer(r)
}

func seedVenueSearchSession(t *testing.T, ctx context.Context, queries *db.Queries) {
	t.Helper()

	googleID := "google-venue-user"
	if _, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID:          "venue-user",
		Email:       "venue-user@example.com",
		DisplayName: "Venue User",
		GoogleID:    &googleID,
	}); err != nil {
		t.Fatalf("UpsertUserByGoogleID: %v", err)
	}
	if err := queries.CreateSession(ctx, db.CreateSessionParams{
		Token:     "tok",
		UserID:    "venue-user",
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
}

func TestSearchVenues_EnoughLocalMatchesDoesNotCallGoogle(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	seedVenueSearchSession(t, ctx, queries)

	seedVenue(t, ctx, queries, "319795", "Beatty Secondary School")
	seedVenue(t, ctx, queries, "319796", "Beatty Sports Hall")

	places := &fakePlaceProvider{
		suggestions: []geo.Suggestion{{PlaceID: "google-1", Name: "Google Hall"}},
	}
	ts := setupVenueSearchTest(t, queries, places)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/venues/search?q=beatty", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if places.autocompleteCalls != 0 {
		t.Fatalf("autocompleteCalls = %d, want 0 for enough local matches", places.autocompleteCalls)
	}

	var out struct {
		Items []struct {
			Source  *string `json:"source"`
			ID      *int64  `json:"id"`
			Name    string  `json:"name"`
			Address string  `json:"address"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("items len = %d, want 2 local results: %+v", len(out.Items), out.Items)
	}
	for _, item := range out.Items {
		if item.Source != nil {
			t.Fatalf("item source = %q, want source omitted: %+v", *item.Source, out.Items)
		}
		if item.ID == nil {
			t.Fatalf("item id was nil, want saved venue id: %+v", item)
		}
	}
	if out.Items[0].Address == "" {
		t.Fatal("address was empty, want street address in search result")
	}
}

func TestSearchVenues_FewLocalMatchesTopsUpWithGoogle(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	seedVenueSearchSession(t, ctx, queries)

	googlePlaceID := "ChIJ-local"
	if _, err := queries.UpsertVenue(ctx, db.UpsertVenueParams{
		PostalCode:    ptrString("319795"),
		Name:          "Beatty Secondary School",
		Address:       "1 Toa Payoh Lor 3, Singapore 319795",
		Latitude:      1.3285,
		Longitude:     103.8571,
		Source:        "google",
		GooglePlaceID: &googlePlaceID,
	}); err != nil {
		t.Fatalf("UpsertVenue: %v", err)
	}

	places := &fakePlaceProvider{
		suggestions: []geo.Suggestion{
			{PlaceID: "ChIJ-local", Name: "Duplicate Local Place", Address: "Duplicate Road"},
			{PlaceID: "ChIJ-remote", Name: "Beatty New Court", Address: "123 Test Road"},
		},
	}
	ts := setupVenueSearchTest(t, queries, places)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/venues/search?q=beatty&session_token=session-1", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if places.autocompleteCalls != 1 {
		t.Fatalf("autocompleteCalls = %d, want 1 for fewer than 2 local matches", places.autocompleteCalls)
	}

	var out struct {
		Items []struct {
			Source        *string `json:"source"`
			ID            *int64  `json:"id"`
			Name          string  `json:"name"`
			GooglePlaceID *string `json:"google_place_id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("items len = %d, want local plus one deduped Google result: %+v", len(out.Items), out.Items)
	}
	if out.Items[0].Source != nil || out.Items[0].ID == nil || out.Items[0].Name != "Beatty Secondary School" {
		t.Fatalf("first item = %+v, want saved DB match first without source", out.Items[0])
	}
	if out.Items[1].Source != nil || out.Items[1].ID != nil || out.Items[1].GooglePlaceID == nil || *out.Items[1].GooglePlaceID != "ChIJ-remote" {
		t.Fatalf("second item = %+v, want unresolved suggestion with place id and no source", out.Items[1])
	}
}

func TestSearchVenues_NoLocalMatchReturnsGoogleSuggestions(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	seedVenueSearchSession(t, ctx, queries)

	places := &fakePlaceProvider{
		suggestions: []geo.Suggestion{{
			PlaceID: "ChIJ123",
			Name:    "New Sports Hall",
			Address: "123 Test Road, Singapore",
		}},
	}
	ts := setupVenueSearchTest(t, queries, places)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/venues/search?q=new%20sports&session_token=session-1", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if places.autocompleteCalls != 1 {
		t.Fatalf("autocompleteCalls = %d, want 1", places.autocompleteCalls)
	}

	var out struct {
		Items []struct {
			Source        *string `json:"source"`
			ID            *int64  `json:"id"`
			Name          string  `json:"name"`
			Address       string  `json:"address"`
			GooglePlaceID *string `json:"google_place_id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(out.Items) != 1 || out.Items[0].Source != nil || out.Items[0].ID != nil || out.Items[0].GooglePlaceID == nil || *out.Items[0].GooglePlaceID != "ChIJ123" {
		t.Fatalf("items = %+v, want unresolved suggestion with place id and no source", out.Items)
	}
	if out.Items[0].Address != "123 Test Road, Singapore" {
		t.Fatalf("address = %q, want Google secondary text", out.Items[0].Address)
	}
}

func TestResolveGoogleVenue_StoresVenueAndAliases(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	seedVenueSearchSession(t, ctx, queries)

	places := &fakePlaceProvider{
		details: &geo.Result{
			PlaceID:   "ChIJ456",
			Name:      "Resolved Sports Hall",
			Address:   "456 Court Street, Singapore",
			Postal:    "123456",
			Latitude:  1.35,
			Longitude: 103.9,
			Source:    "google",
		},
	}
	ts := setupVenueSearchTest(t, queries, places)
	defer ts.Close()

	body := `{"google_place_id":"ChIJ456","session_token":"session-2","query":"resolved hall"}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/venues/resolve-google", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if places.detailsCalls != 1 {
		t.Fatalf("detailsCalls = %d, want 1", places.detailsCalls)
	}

	var out struct {
		ID            int64   `json:"id"`
		Name          string  `json:"name"`
		Address       string  `json:"address"`
		GooglePlaceID *string `json:"google_place_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.ID == 0 || out.Name != "Resolved Sports Hall" || out.GooglePlaceID == nil || *out.GooglePlaceID != "ChIJ456" {
		t.Fatalf("response = %+v, want stored venue with Google place id", out)
	}
	if out.Address != "456 Court Street, Singapore" {
		t.Fatalf("address = %q, want stored street address", out.Address)
	}

	venue, err := queries.GetVenueByGooglePlaceID(ctx, ptrString("ChIJ456"))
	if err != nil {
		t.Fatalf("GetVenueByGooglePlaceID: %v", err)
	}
	if venue.ID != out.ID {
		t.Fatalf("stored venue id = %d, want response id %d", venue.ID, out.ID)
	}
	if alias, err := queries.GetVenueByAlias(ctx, "resolved hall"); err != nil || alias.ID != out.ID {
		t.Fatalf("query alias = %+v, err=%v; want stored venue", alias, err)
	}
}

func TestResolveGoogleVenue_ReusesExistingPostalCodeVenue(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	seedVenueSearchSession(t, ctx, queries)

	postalCode := "123456"
	existing, err := queries.UpsertVenue(ctx, db.UpsertVenueParams{
		PostalCode: &postalCode,
		Name:       "Existing Manual Hall",
		Address:    "456 Court Street, Singapore 123456",
		Latitude:   1.35,
		Longitude:  103.9,
		Source:     "manual",
	})
	if err != nil {
		t.Fatalf("UpsertVenue existing: %v", err)
	}

	places := &fakePlaceProvider{
		details: &geo.Result{
			PlaceID:   "ChIJ-postal-collision",
			Name:      "Resolved Google Hall",
			Address:   "456 Court Street, Singapore 123456",
			Postal:    postalCode,
			Latitude:  1.3501,
			Longitude: 103.9001,
			Source:    "google",
		},
	}
	ts := setupVenueSearchTest(t, queries, places)
	defer ts.Close()

	body := `{"google_place_id":"ChIJ-postal-collision","session_token":"session-3","query":"google hall"}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/venues/resolve-google", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		ID            int64   `json:"id"`
		Name          string  `json:"name"`
		GooglePlaceID *string `json:"google_place_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.ID != existing.ID {
		t.Fatalf("id = %d, want existing postal venue id %d", out.ID, existing.ID)
	}
	if out.GooglePlaceID == nil || *out.GooglePlaceID != "ChIJ-postal-collision" {
		t.Fatalf("google_place_id = %v, want collision place id", out.GooglePlaceID)
	}
	if out.Name != "Resolved Google Hall" {
		t.Fatalf("name = %q, want Google details to refresh existing venue", out.Name)
	}
}

func ptrString(value string) *string {
	return &value
}

func seedVenue(t *testing.T, ctx context.Context, queries *db.Queries, postalCode string, name string) {
	t.Helper()

	if _, err := queries.UpsertVenue(ctx, db.UpsertVenueParams{
		PostalCode: &postalCode,
		Name:       name,
		Address:    name + ", Singapore " + postalCode,
		Latitude:   1.3285,
		Longitude:  103.8571,
		Source:     "manual",
	}); err != nil {
		t.Fatalf("UpsertVenue(%s): %v", name, err)
	}
}
