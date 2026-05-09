package plays_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/plays"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

type fakeCreatePlayStore struct {
	play     db.Play
	err      error
	lastArgs db.CreatePlayParams // captures last call args for assertions
}

func (f *fakeCreatePlayStore) CreatePlay(_ context.Context, arg db.CreatePlayParams) (db.Play, error) {
	f.lastArgs = arg
	if f.err != nil {
		return db.Play{}, f.err
	}
	p := f.play
	p.Sport = arg.Sport
	p.Venue = arg.Venue
	p.HostName = arg.HostName
	p.StartsAt = arg.StartsAt
	p.EndsAt = arg.EndsAt
	p.CreatedBy = arg.CreatedBy
	return p, nil
}

type fakeAuthStore struct {
	sessionRow db.GetSessionWithUserRow
	sessionErr error
}

func (f *fakeAuthStore) UpsertUserByGoogleID(_ context.Context, _ db.UpsertUserByGoogleIDParams) (db.User, error) {
	return db.User{}, nil
}
func (f *fakeAuthStore) UpsertUserByFacebookID(_ context.Context, _ db.UpsertUserByFacebookIDParams) (db.User, error) {
	return db.User{}, nil
}
func (f *fakeAuthStore) LinkGoogleID(_ context.Context, _ db.LinkGoogleIDParams) (db.User, error) {
	return db.User{}, nil
}
func (f *fakeAuthStore) LinkFacebookID(_ context.Context, _ db.LinkFacebookIDParams) (db.User, error) {
	return db.User{}, nil
}
func (f *fakeAuthStore) GetSessionWithUser(_ context.Context, _ string) (db.GetSessionWithUserRow, error) {
	return f.sessionRow, f.sessionErr
}
func (f *fakeAuthStore) CreateSession(_ context.Context, _ db.CreateSessionParams) error { return nil }
func (f *fakeAuthStore) DeleteSession(_ context.Context, _ string) error                 { return nil }
func (f *fakeAuthStore) RefreshSession(_ context.Context, _ db.RefreshSessionParams) error {
	return nil
}

func setupCreateTest(authStore *fakeAuthStore, playStore plays.CreatePlayStore) *httptest.Server {
	svc := auth.NewService(authStore)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/plays")
	plays.RegisterCreate(grp, playStore, authmw.RequireAuth(api, svc))

	return httptest.NewServer(r)
}

func activeSession() *fakeAuthStore {
	now := time.Now()
	return &fakeAuthStore{
		sessionRow: db.GetSessionWithUserRow{
			Token: "tok", UserID: "user-1", ExpiresAt: now.Add(time.Hour),
			UserID2: "user-1", Email: "test@test.com", DisplayName: "Test User",
			Status: "active", CreatedAt: now, UpdatedAt: now,
		},
	}
}

func TestCreatePlay_Success(t *testing.T) {
	now := time.Now()
	playStore := &fakeCreatePlayStore{
		play: db.Play{
			ID: 1, ListingType: model.ListingPlay, Currency: "SGD",
			Timezone: "Asia/Singapore", CreatedAt: now, UpdatedAt: now,
		},
	}
	ts := setupCreateTest(activeSession(), playStore)
	defer ts.Close()

	body := `{"sport":"badminton","venue":"SBH","starts_at":"2026-06-01T10:00:00+08:00","duration_minutes":120,"timezone":"Asia/Singapore","currency":"SGD"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/plays/", strings.NewReader(body))
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
}

func TestCreatePlay_NoAuth_Returns401(t *testing.T) {
	ts := setupCreateTest(&fakeAuthStore{sessionErr: sql.ErrNoRows}, &fakeCreatePlayStore{})
	defer ts.Close()

	body := `{"sport":"badminton","venue":"SBH","starts_at":"2026-06-01T10:00:00+08:00","duration_minutes":60}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/plays/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestCreatePlay_DurationNot15MinIncrement_Returns422(t *testing.T) {
	ts := setupCreateTest(activeSession(), &fakeCreatePlayStore{})
	defer ts.Close()

	body := `{"sport":"badminton","venue":"SBH","starts_at":"2026-06-01T10:00:00+08:00","duration_minutes":50,"timezone":"Asia/Singapore","currency":"SGD"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/plays/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestCreatePlay_DurationTooLong_Returns422(t *testing.T) {
	ts := setupCreateTest(activeSession(), &fakeCreatePlayStore{})
	defer ts.Close()

	// 315 minutes > 300 max — huma validates maximum:"300" on the field
	body := `{"sport":"badminton","venue":"SBH","starts_at":"2026-06-01T10:00:00+08:00","duration_minutes":315,"timezone":"Asia/Singapore","currency":"SGD"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/plays/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestCreatePlay_DurationTooShort_Returns422(t *testing.T) {
	ts := setupCreateTest(activeSession(), &fakeCreatePlayStore{})
	defer ts.Close()

	// 10 minutes < 15 min — huma validates minimum:"15" on the field
	body := `{"sport":"badminton","venue":"SBH","starts_at":"2026-06-01T10:00:00+08:00","duration_minutes":10,"timezone":"Asia/Singapore","currency":"SGD"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/plays/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestCreatePlay_StartsAtStoredAsUTC(t *testing.T) {
	now := time.Now()
	playStore := &fakeCreatePlayStore{
		play: db.Play{
			ID: 1, ListingType: model.ListingPlay, Currency: "SGD",
			Timezone: "Asia/Singapore", CreatedAt: now, UpdatedAt: now,
		},
	}
	ts := setupCreateTest(activeSession(), playStore)
	defer ts.Close()

	// Send time with +08:00 offset
	body := `{"sport":"badminton","venue":"SBH","starts_at":"2026-06-01T10:00:00+08:00","duration_minutes":120,"timezone":"Asia/Singapore","currency":"SGD"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/plays/", strings.NewReader(body))
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

	// Verify stored time is UTC (02:00 UTC = 10:00 +08:00)
	if playStore.lastArgs.StartsAt.Location() != time.UTC {
		t.Errorf("starts_at location = %v, want UTC", playStore.lastArgs.StartsAt.Location())
	}
	expected := time.Date(2026, 6, 1, 2, 0, 0, 0, time.UTC)
	if !playStore.lastArgs.StartsAt.Equal(expected) {
		t.Errorf("starts_at = %v, want %v", playStore.lastArgs.StartsAt, expected)
	}
}

func TestCreatePlay_EndsAtComputedFromDuration(t *testing.T) {
	now := time.Now()
	playStore := &fakeCreatePlayStore{
		play: db.Play{
			ID: 1, ListingType: model.ListingPlay, Currency: "SGD",
			Timezone: "Asia/Singapore", CreatedAt: now, UpdatedAt: now,
		},
	}
	ts := setupCreateTest(activeSession(), playStore)
	defer ts.Close()

	// 90 minutes duration
	body := `{"sport":"badminton","venue":"SBH","starts_at":"2026-06-01T10:00:00+08:00","duration_minutes":90,"timezone":"Asia/Singapore","currency":"SGD"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/plays/", strings.NewReader(body))
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

	// starts_at: 02:00 UTC, duration: 90 min → ends_at: 03:30 UTC
	expectedEnd := time.Date(2026, 6, 1, 3, 30, 0, 0, time.UTC)
	if !playStore.lastArgs.EndsAt.Equal(expectedEnd) {
		t.Errorf("ends_at = %v, want %v", playStore.lastArgs.EndsAt, expectedEnd)
	}
}

func TestCreatePlay_PastStartTime_Returns422(t *testing.T) {
	ts := setupCreateTest(activeSession(), &fakeCreatePlayStore{})
	defer ts.Close()

	// Use a date clearly in the past
	body := `{"sport":"badminton","venue":"SBH","starts_at":"2020-01-01T10:00:00+08:00","duration_minutes":60,"timezone":"Asia/Singapore","currency":"SGD"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/plays/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestCreatePlay_ResolvesVenueID_ForKnownVenueName(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)

	venue, err := queries.UpsertVenue(context.Background(), db.UpsertVenueParams{
		PostalCode: ptrString("359844"),
		Name:       "Beatty Secondary School",
		Address:    "1 Toa Payoh Lor 3, Singapore 319795",
		Latitude:   1.3285,
		Longitude:  103.8571,
		Source:     "manual",
		SearchTerm: ptrString("beatty secondary school"),
	})
	if err != nil {
		t.Fatalf("upsert venue: %v", err)
	}

	ts := setupCreateTest(activeSession(), queries)
	defer ts.Close()

	startsAt := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"sport":"badminton","venue":"Beatty Secondary School","starts_at":"%s","duration_minutes":120,"timezone":"Asia/Singapore","currency":"SGD"}`,
		startsAt,
	)

	req, _ := http.NewRequest("POST", ts.URL+"/api/plays/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200, body=%s", resp.StatusCode, string(b))
	}

	var out struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.ID == 0 {
		t.Fatalf("expected created play id in response")
	}

	row, err := queries.GetPlayByID(context.Background(), out.ID)
	if err != nil {
		t.Fatalf("GetPlayByID: %v", err)
	}
	if row.VenueID == nil {
		t.Fatalf("venue_id = nil, want %d", venue.ID)
	}
	if *row.VenueID != venue.ID {
		t.Fatalf("venue_id = %d, want %d", *row.VenueID, venue.ID)
	}
}

func ptrString(v string) *string { return &v }
