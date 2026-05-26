package plays_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/plays"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

func setupJoinLeaveTest(authStore *fakeAuthStore, store *db.Queries) *httptest.Server {
	svc := auth.NewService(authStore)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/plays")
	plays.RegisterJoin(grp, store, authmw.RequireAuth(api, svc))
	plays.RegisterLeave(grp, store, authmw.RequireAuth(api, svc))

	return httptest.NewServer(r)
}

func TestJoinPlay_AutoConfirmWhenRatingMatchesAndSpaceExists(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "creator-1")
	joinerID := createRouteTestUser(t, ctx, queries, "joiner-1")

	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	authStore := sessionWithProfile(joinerID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupJoinLeaveTest(authStore, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/plays/"+playID+"/join", nil)
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
		Status    string `json:"status"`
		SlotsLeft *int64 `json:"slots_left"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Status != string(model.ParticipantConfirmed) {
		t.Fatalf("status = %q, want confirmed", out.Status)
	}
	if out.SlotsLeft == nil || *out.SlotsLeft != 1 {
		t.Fatalf("slots_left = %v, want 1", out.SlotsLeft)
	}
}

func TestJoinPlay_AutoWaitlistWhenFull(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "creator-2")
	existingID := createRouteTestUser(t, ctx, queries, "existing-1")
	joinerID := createRouteTestUser(t, ctx, queries, "joiner-2")

	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedConfirmedParticipant(t, ctx, queries, playID, existingID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	authStore := sessionWithProfile(joinerID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupJoinLeaveTest(authStore, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/plays/"+playID+"/join", nil)
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
		Status    string `json:"status"`
		SlotsLeft *int64 `json:"slots_left"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Status != string(model.ParticipantWaitlisted) {
		t.Fatalf("status = %q, want waitlisted", out.Status)
	}
	if out.SlotsLeft == nil || *out.SlotsLeft != 0 {
		t.Fatalf("slots_left = %v, want 0", out.SlotsLeft)
	}
}

func TestJoinPlay_AutoWaitlistWhenRatingMissing(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "creator-3")
	joinerID := createRouteTestUser(t, ctx, queries, "joiner-3")

	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	authStore := sessionWithProfile(joinerID, nil)
	ts := setupJoinLeaveTest(authStore, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/plays/"+playID+"/join", nil)
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
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Status != string(model.ParticipantWaitlisted) {
		t.Fatalf("status = %q, want waitlisted", out.Status)
	}
}

func TestLeavePlay_RemovesParticipantAndFreesSlot(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "creator-4")
	joinerID := createRouteTestUser(t, ctx, queries, "joiner-4")

	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedConfirmedParticipant(t, ctx, queries, playID, joinerID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	authStore := sessionWithProfile(joinerID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupJoinLeaveTest(authStore, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/plays/"+playID+"/participants/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}

	_, err = queries.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{
		PlayID: playID,
		UserID: &joinerID,
	})
	if err == nil {
		t.Fatal("expected participant to be removed")
	}

	play, err := queries.GetPlayByID(ctx, playID)
	if err != nil {
		t.Fatalf("GetPlayByID: %v", err)
	}
	if play.SlotsLeft == nil || *play.SlotsLeft != 2 {
		t.Fatalf("slots_left = %v, want 2", play.SlotsLeft)
	}
}

func createUserPlay(t *testing.T, ctx context.Context, queries *db.Queries, creatorID string, maxPlayers int64, levelMin, levelMax *string) string {
	t.Helper()

	startsAt := time.Now().UTC().Add(24 * time.Hour)
	play, err := queries.CreatePlay(ctx, db.CreatePlayParams{
		ID:          uuid.NewString(),
		ListingType: model.ListingPlay,
		Sport:       model.SportBadminton,
		HostName:    "Host",
		StartsAt:    startsAt,
		EndsAt:      startsAt.Add(2 * time.Hour),
		Timezone:    "Asia/Singapore",
		Venue:       "SBH",
		LevelMin:    levelMin,
		LevelMax:    levelMax,
		LevelMinOrd: levelOrdPtr(model.SportBadminton, levelMin),
		LevelMaxOrd: levelOrdPtr(model.SportBadminton, levelMax),
		Currency:    "SGD",
		MaxPlayers:  &maxPlayers,
		SlotsLeft:   &maxPlayers,
		CreatedBy:   &creatorID,
	})
	if err != nil {
		t.Fatalf("CreatePlay: %v", err)
	}
	return play.ID
}

func seedConfirmedParticipant(t *testing.T, ctx context.Context, queries *db.Queries, playID, userID string) {
	t.Helper()

	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: playID,
		UserID: &userID,
		Status: model.ParticipantConfirmed,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant confirmed: %v", err)
	}
}

func sessionWithProfile(userID string, sportsProfile *string) *fakeAuthStore {
	now := time.Now()
	return &fakeAuthStore{
		sessionRow: db.GetSessionWithUserRow{
			Token: "tok", UserID: userID, ExpiresAt: now.Add(time.Hour),
			UserID2: userID, Email: fmt.Sprintf("%s@example.com", userID), DisplayName: "Test User",
			Status: "active", SportsProfile: sportsProfile, CreatedAt: now, UpdatedAt: now,
		},
	}
}

func levelOrdPtr(sport model.Sport, level *string) *int64 {
	if level == nil {
		return nil
	}
	ord := model.LevelOrd(sport, strings.TrimSpace(*level))
	if ord == nil {
		return nil
	}
	v := int64(*ord)
	return &v
}

func createRouteTestUser(t *testing.T, ctx context.Context, queries *db.Queries, id string) string {
	t.Helper()

	googleID := "google-" + id
	user, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID:          id,
		Email:       id + "@example.com",
		DisplayName: "Test " + id,
		GoogleID:    &googleID,
	})
	if err != nil {
		t.Fatalf("create test user %q: %v", id, err)
	}
	return user.ID
}
