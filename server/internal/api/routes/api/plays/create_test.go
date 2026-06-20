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
	play         db.Play
	err          error
	lastArgs     db.CreatePlayParams // captures last call args for assertions
	lastHostArgs db.CreatePlayHostParams
	lastEventArg db.CreatePlayEventParams
}

func (f *fakeCreatePlayStore) CreatePlay(_ context.Context, arg db.CreatePlayParams) (db.Play, error) {
	f.lastArgs = arg
	if f.err != nil {
		return db.Play{}, f.err
	}
	p := f.play
	p.ID = arg.ID
	p.Sport = arg.Sport
	p.Venue = arg.Venue
	p.HostName = arg.HostName
	p.Name = arg.Name
	p.Description = arg.Description
	if visibility, ok := arg.Visibility.(model.PlayVisibility); ok {
		p.Visibility = visibility
	}
	if p.Visibility == "" {
		p.Visibility = model.PlayVisibilityPublic
	}
	p.StartsAt = arg.StartsAt
	p.EndsAt = arg.EndsAt
	p.CreatedBy = arg.CreatedBy
	return p, nil
}

func (f *fakeCreatePlayStore) CreatePlayHost(_ context.Context, arg db.CreatePlayHostParams) (db.PlayHost, error) {
	f.lastHostArgs = arg
	return db.PlayHost{PlayID: arg.PlayID, UserID: arg.UserID}, nil
}

func (f *fakeCreatePlayStore) CreatePlayParticipant(_ context.Context, _ db.CreatePlayParticipantParams) (db.PlayParticipant, error) {
	return db.PlayParticipant{}, nil
}

func (f *fakeCreatePlayStore) CreatePlayEvent(_ context.Context, arg db.CreatePlayEventParams) (db.PlayEvent, error) {
	f.lastEventArg = arg
	return db.PlayEvent{PlayID: arg.PlayID, EventType: arg.EventType}, nil
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

// futureStart returns a stable start time safely in the future, on a whole hour
// in UTC. The create handler rejects non-future starts_at, so request bodies
// must be built relative to now rather than a hardcoded date.
func futureStart() time.Time {
	return time.Now().UTC().Add(24 * time.Hour).Truncate(time.Hour)
}

func TestCreatePlay_Success(t *testing.T) {
	now := time.Now()
	playStore := &fakeCreatePlayStore{
		play: db.Play{
			ID: "play-1", ListingType: model.ListingPlay, Currency: "SGD",
			Timezone: "Asia/Singapore", CreatedAt: now, UpdatedAt: now,
		},
	}
	ts := setupCreateTest(activeSession(), playStore)
	defer ts.Close()

	body := fmt.Sprintf(`{"sport":"badminton","venue":"SBH","name":"  Saturday Social  ","description":"  Bring water  ","starts_at":"%s","duration_minutes":120,"timezone":"Asia/Singapore","currency":"SGD","max_players":4}`,
		futureStart().Format(time.RFC3339))
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
	if playStore.lastHostArgs.UserID != "user-1" || playStore.lastHostArgs.PlayID == "" {
		t.Fatalf("CreatePlayHost args = %+v, want user-1 host", playStore.lastHostArgs)
	}
	if playStore.lastArgs.Name == nil || *playStore.lastArgs.Name != "Saturday Social" {
		t.Fatalf("name = %v, want trimmed custom name", playStore.lastArgs.Name)
	}
	if playStore.lastArgs.Description == nil || *playStore.lastArgs.Description != "Bring water" {
		t.Fatalf("description = %v, want trimmed description", playStore.lastArgs.Description)
	}
	if playStore.lastArgs.Visibility != model.PlayVisibilityPublic {
		t.Fatalf("visibility = %v, want public", playStore.lastArgs.Visibility)
	}
	var out plays.PlayPublic
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Name == nil || *out.Name != "Saturday Social" {
		t.Fatalf("response name = %v, want Saturday Social", out.Name)
	}
	if out.Description == nil || *out.Description != "Bring water" {
		t.Fatalf("response description = %v, want Bring water", out.Description)
	}
	if out.Visibility != model.PlayVisibilityPublic {
		t.Fatalf("response visibility = %q, want public", out.Visibility)
	}
}

func TestCreatePlay_CanCreateUnlisted(t *testing.T) {
	now := time.Now()
	playStore := &fakeCreatePlayStore{
		play: db.Play{
			ID: "play-1", ListingType: model.ListingPlay, Currency: "SGD",
			Timezone: "Asia/Singapore", CreatedAt: now, UpdatedAt: now,
		},
	}
	ts := setupCreateTest(activeSession(), playStore)
	defer ts.Close()

	body := fmt.Sprintf(`{"sport":"badminton","venue":"SBH","visibility":"unlisted","starts_at":"%s","duration_minutes":120,"timezone":"Asia/Singapore","currency":"SGD","max_players":4}`,
		futureStart().Format(time.RFC3339))
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
	if playStore.lastArgs.Visibility != model.PlayVisibilityUnlisted {
		t.Fatalf("visibility = %v, want unlisted", playStore.lastArgs.Visibility)
	}
	var out plays.PlayPublic
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Visibility != model.PlayVisibilityUnlisted {
		t.Fatalf("response visibility = %q, want unlisted", out.Visibility)
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

	body := `{"sport":"badminton","venue":"SBH","starts_at":"2026-06-01T10:00:00+08:00","duration_minutes":50,"timezone":"Asia/Singapore","currency":"SGD","max_players":4}`
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
	body := `{"sport":"badminton","venue":"SBH","starts_at":"2026-06-01T10:00:00+08:00","duration_minutes":315,"timezone":"Asia/Singapore","currency":"SGD","max_players":4}`
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
	body := `{"sport":"badminton","venue":"SBH","starts_at":"2026-06-01T10:00:00+08:00","duration_minutes":10,"timezone":"Asia/Singapore","currency":"SGD","max_players":4}`
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

func TestCreatePlay_RejectsTooLongCustomText(t *testing.T) {
	ts := setupCreateTest(activeSession(), &fakeCreatePlayStore{})
	defer ts.Close()

	body := fmt.Sprintf(`{"sport":"badminton","venue":"SBH","name":%q,"starts_at":"%s","duration_minutes":120,"timezone":"Asia/Singapore","currency":"SGD","max_players":4}`,
		strings.Repeat("a", 81),
		futureStart().Format(time.RFC3339),
	)
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
			ID: "play-1", ListingType: model.ListingPlay, Currency: "SGD",
			Timezone: "Asia/Singapore", CreatedAt: now, UpdatedAt: now,
		},
	}
	ts := setupCreateTest(activeSession(), playStore)
	defer ts.Close()

	// Send time with +08:00 offset
	start := futureStart()
	body := fmt.Sprintf(`{"sport":"badminton","venue":"SBH","starts_at":"%s","duration_minutes":120,"timezone":"Asia/Singapore","currency":"SGD","max_players":4}`,
		start.In(time.FixedZone("SGT", 8*60*60)).Format(time.RFC3339))
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

	// Sent with a +08:00 offset; handler must normalise and store as UTC.
	if playStore.lastArgs.StartsAt.Location() != time.UTC {
		t.Errorf("starts_at location = %v, want UTC", playStore.lastArgs.StartsAt.Location())
	}
	if !playStore.lastArgs.StartsAt.Equal(start) {
		t.Errorf("starts_at = %v, want %v", playStore.lastArgs.StartsAt, start)
	}
}

func TestCreatePlay_EndsAtComputedFromDuration(t *testing.T) {
	now := time.Now()
	playStore := &fakeCreatePlayStore{
		play: db.Play{
			ID: "play-1", ListingType: model.ListingPlay, Currency: "SGD",
			Timezone: "Asia/Singapore", CreatedAt: now, UpdatedAt: now,
		},
	}
	ts := setupCreateTest(activeSession(), playStore)
	defer ts.Close()

	// 90 minutes duration
	start := futureStart()
	body := fmt.Sprintf(`{"sport":"badminton","venue":"SBH","starts_at":"%s","duration_minutes":90,"timezone":"Asia/Singapore","currency":"SGD","max_players":4}`,
		start.Format(time.RFC3339))
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

	// ends_at = starts_at + 90 minutes
	expectedEnd := start.Add(90 * time.Minute)
	if !playStore.lastArgs.EndsAt.Equal(expectedEnd) {
		t.Errorf("ends_at = %v, want %v", playStore.lastArgs.EndsAt, expectedEnd)
	}
}

func TestCreatePlay_TennisLevelsMappedToOrdinals(t *testing.T) {
	now := time.Now()
	playStore := &fakeCreatePlayStore{
		play: db.Play{
			ID: "play-1", ListingType: model.ListingPlay, Currency: "SGD",
			Timezone: "Asia/Singapore", CreatedAt: now, UpdatedAt: now,
		},
	}
	ts := setupCreateTest(activeSession(), playStore)
	defer ts.Close()

	body := fmt.Sprintf(`{"sport":"tennis","venue":"Kallang Tennis Centre","starts_at":"%s","duration_minutes":90,"timezone":"Asia/Singapore","currency":"SGD","max_players":4,"level_min":"3.0","level_max":"4.0"}`,
		futureStart().Format(time.RFC3339))
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

	if playStore.lastArgs.LevelMinOrd == nil || *playStore.lastArgs.LevelMinOrd != 30 {
		t.Fatalf("level_min_ord = %v, want 30", playStore.lastArgs.LevelMinOrd)
	}
	if playStore.lastArgs.LevelMaxOrd == nil || *playStore.lastArgs.LevelMaxOrd != 40 {
		t.Fatalf("level_max_ord = %v, want 40", playStore.lastArgs.LevelMaxOrd)
	}
}

func TestCreatePlay_PastStartTime_Returns422(t *testing.T) {
	ts := setupCreateTest(activeSession(), &fakeCreatePlayStore{})
	defer ts.Close()

	// Use a date clearly in the past
	body := `{"sport":"badminton","venue":"SBH","starts_at":"2020-01-01T10:00:00+08:00","duration_minutes":60,"timezone":"Asia/Singapore","currency":"SGD","max_players":4}`
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
	body := fmt.Sprintf(`{"sport":"badminton","venue":"Beatty Secondary School","starts_at":"%s","duration_minutes":120,"timezone":"Asia/Singapore","currency":"SGD","max_players":4}`,
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
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.ID == "" {
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

func TestCreatePlay_MaxPlayersRequired_Returns422(t *testing.T) {
	ts := setupCreateTest(activeSession(), &fakeCreatePlayStore{})
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

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestCreatePlay_SeedsCreatorAndDerivesSlotsLeft(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)

	_, err := queries.UpsertUserByGoogleID(context.Background(), db.UpsertUserByGoogleIDParams{
		ID:          "user-1",
		Email:       "user-1@example.com",
		DisplayName: "Test User",
		GoogleID:    ptrString("google-user-1"),
	})
	if err != nil {
		t.Fatalf("upsert user: %v", err)
	}

	ts := setupCreateTest(activeSession(), queries)
	defer ts.Close()

	startsAt := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	body := fmt.Sprintf(`{"sport":"badminton","venue":"SBH","starts_at":"%s","duration_minutes":120,"timezone":"Asia/Singapore","currency":"SGD","max_players":4}`,
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
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	participants, err := queries.ListPlayParticipantsByPlay(context.Background(), out.ID)
	if err != nil {
		t.Fatalf("ListPlayParticipantsByPlay: %v", err)
	}
	if len(participants) != 1 {
		t.Fatalf("participants = %d, want 1", len(participants))
	}
	if participants[0].UserID == nil || *participants[0].UserID != "user-1" {
		t.Fatalf("participant user_id = %v, want user-1", participants[0].UserID)
	}
	if participants[0].Status != model.ParticipantConfirmed {
		t.Fatalf("participant status = %q, want confirmed", participants[0].Status)
	}

	if _, err := queries.GetPlayHost(context.Background(), db.GetPlayHostParams{PlayID: out.ID, UserID: "user-1"}); err != nil {
		t.Fatalf("GetPlayHost: %v", err)
	}
	row, err := queries.GetPlayByID(context.Background(), out.ID)
	if err != nil {
		t.Fatalf("GetPlayByID: %v", err)
	}
	if row.SlotsLeft == nil || *row.SlotsLeft != 3 {
		t.Fatalf("slots_left = %v, want 3", row.SlotsLeft)
	}
}

func ptrString(v string) *string { return &v }
