package plays_test

import (
	"context"
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

func setupHostPlayManagementTest(authStore *fakeAuthStore, store *db.Queries) *httptest.Server {
	svc := auth.NewService(authStore)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/plays")
	plays.RegisterUpdate(grp, store, authmw.RequireAuth(api, svc))
	plays.RegisterDelete(grp, store, authmw.RequireAuth(api, svc))

	return httptest.NewServer(r)
}

func TestUpdatePlay_HostCanUpdateEditableFields(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "update-host")
	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("MI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)

	startsAt := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	body := fmt.Sprintf(`{
		"starts_at": %q,
		"duration_minutes": 90,
		"timezone": "Asia/Singapore",
		"level_min": "HB",
		"level_max": "HI",
		"fee": 1350,
		"max_players": 4,
		"courts": 2
	}`, startsAt.Format(time.RFC3339))

	ts := setupHostPlayManagementTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/plays/"+playID, strings.NewReader(body))
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
	var out plays.PlayPublic
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.CreatedAt != nil || out.UpdatedAt != nil {
		t.Fatalf("timestamps = %v/%v, want omitted for user-created play", out.CreatedAt, out.UpdatedAt)
	}

	got, err := queries.GetPlayByID(ctx, playID)
	if err != nil {
		t.Fatalf("GetPlayByID: %v", err)
	}
	if got.StartsAt.Format(time.RFC3339) != startsAt.Format(time.RFC3339) {
		t.Fatalf("starts_at = %s, want %s", got.StartsAt.Format(time.RFC3339), startsAt.Format(time.RFC3339))
	}
	wantEnd := startsAt.Add(90 * time.Minute).Format(time.RFC3339)
	if got.EndsAt.Format(time.RFC3339) != wantEnd {
		t.Fatalf("ends_at = %s, want %s", got.EndsAt.Format(time.RFC3339), wantEnd)
	}
	if got.LevelMin == nil || *got.LevelMin != "HB" {
		t.Fatalf("level_min = %v, want HB", got.LevelMin)
	}
	if got.LevelMax == nil || *got.LevelMax != "HI" {
		t.Fatalf("level_max = %v, want HI", got.LevelMax)
	}
	if got.LevelMinOrd == nil || *got.LevelMinOrd != 30 {
		t.Fatalf("level_min_ord = %v, want 30", got.LevelMinOrd)
	}
	if got.Fee == nil || *got.Fee != 1350 {
		t.Fatalf("fee = %v, want 1350", got.Fee)
	}
	if got.MaxPlayers == nil || *got.MaxPlayers != 4 {
		t.Fatalf("max_players = %v, want 4", got.MaxPlayers)
	}
	if got.SlotsLeft == nil || *got.SlotsLeft != 3 {
		t.Fatalf("slots_left = %v, want 3", got.SlotsLeft)
	}
	if got.Courts == nil || *got.Courts != 2 {
		t.Fatalf("courts = %v, want 2", got.Courts)
	}
}

func TestUpdatePlay_ClearsOptionalFields(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "update-clear-host")
	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)

	play, err := queries.GetPlayByID(ctx, playID)
	if err != nil {
		t.Fatalf("GetPlayByID: %v", err)
	}
	doubles := model.GameDoubles
	fee := int64(1200)
	courts := int64(2)
	if _, err := queries.UpdateUserCreatedPlay(ctx, db.UpdateUserCreatedPlayParams{
		ID:          playID,
		GameType:    &doubles,
		StartsAt:    play.StartsAt,
		EndsAt:      play.EndsAt,
		Timezone:    play.Timezone,
		LevelMin:    play.LevelMin,
		LevelMax:    play.LevelMax,
		LevelMinOrd: play.LevelMinOrd,
		LevelMaxOrd: play.LevelMaxOrd,
		Fee:         &fee,
		MaxPlayers:  play.MaxPlayers,
		Courts:      &courts,
	}); err != nil {
		t.Fatalf("UpdateUserCreatedPlay seed: %v", err)
	}

	ts := setupHostPlayManagementTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	body := `{"game_type":"","level_min":"","level_max":"","fee_clear":true,"courts_clear":true}`
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/plays/"+playID, strings.NewReader(body))
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

	got, err := queries.GetPlayByID(ctx, playID)
	if err != nil {
		t.Fatalf("GetPlayByID: %v", err)
	}
	if got.GameType != nil {
		t.Fatalf("game_type = %v, want nil", got.GameType)
	}
	if got.LevelMin != nil || got.LevelMax != nil || got.LevelMinOrd != nil || got.LevelMaxOrd != nil {
		t.Fatalf("levels = %v/%v ord %v/%v, want nil", got.LevelMin, got.LevelMax, got.LevelMinOrd, got.LevelMaxOrd)
	}
	if got.Fee != nil {
		t.Fatalf("fee = %v, want nil", got.Fee)
	}
	if got.Courts != nil {
		t.Fatalf("courts = %v, want nil", got.Courts)
	}
}

func TestUpdatePlay_RejectsLoweringMaxPlayersBelowReservedCount(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "update-cap-host")
	addedID := createRouteTestUser(t, ctx, queries, "update-cap-player")
	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedAddedParticipant(t, ctx, queries, playID, addedID)

	ts := setupHostPlayManagementTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/plays/"+playID, strings.NewReader(`{"max_players":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 409, body=%s", resp.StatusCode, string(b))
	}

	got, err := queries.GetPlayByID(ctx, playID)
	if err != nil {
		t.Fatalf("GetPlayByID: %v", err)
	}
	if got.MaxPlayers == nil || *got.MaxPlayers != 3 {
		t.Fatalf("max_players = %v, want unchanged 3", got.MaxPlayers)
	}
}

func TestUpdatePlay_HostCanUpdateWithoutRosterSlot(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "update-host-not-player")
	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))

	ts := setupHostPlayManagementTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/plays/"+playID, strings.NewReader(`{"max_players":4}`))
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

	got, err := queries.GetPlayByID(ctx, playID)
	if err != nil {
		t.Fatalf("GetPlayByID: %v", err)
	}
	if got.MaxPlayers == nil || *got.MaxPlayers != 4 {
		t.Fatalf("max_players = %v, want 4", got.MaxPlayers)
	}
}

func TestUpdatePlay_RejectsCancelledPlay(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "update-cancelled-host")
	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))
	if _, err := queries.CancelUserCreatedPlay(ctx, db.CancelUserCreatedPlayParams{
		ID:          playID,
		CancelledBy: &creatorID,
	}); err != nil {
		t.Fatalf("CancelUserCreatedPlay: %v", err)
	}

	ts := setupHostPlayManagementTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/plays/"+playID, strings.NewReader(`{"max_players":4}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 409, body=%s", resp.StatusCode, string(b))
	}
}

func TestUpdatePlay_RejectsNonHost(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "update-host")
	otherID := createRouteTestUser(t, ctx, queries, "update-other")
	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))

	ts := setupHostPlayManagementTest(sessionWithProfile(otherID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/plays/"+playID, strings.NewReader(`{"max_players":4}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestCancelPlay_HostMarksPlayCancelledAndKeepsRoster(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "cancel-host")
	waitlistedID := createRouteTestUser(t, ctx, queries, "cancel-waitlisted")
	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)

	ts := setupHostPlayManagementTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/plays/"+playID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 204, body=%s", resp.StatusCode, string(b))
	}
	play, err := queries.GetPlayByID(ctx, playID)
	if err != nil {
		t.Fatalf("GetPlayByID: %v", err)
	}
	if play.CancelledAt == nil {
		t.Fatal("cancelled_at = nil, want timestamp")
	}
	if play.CancelledBy == nil || *play.CancelledBy != creatorID {
		t.Fatalf("cancelled_by = %v, want %s", play.CancelledBy, creatorID)
	}
	participants, err := queries.ListPlayParticipantsByPlay(ctx, playID)
	if err != nil {
		t.Fatalf("ListPlayParticipantsByPlay: %v", err)
	}
	if len(participants) != 2 {
		t.Fatalf("participants len = %d, want 2 preserved", len(participants))
	}
	if _, err := queries.GetPlayHost(ctx, db.GetPlayHostParams{PlayID: playID, UserID: creatorID}); err != nil {
		t.Fatalf("GetPlayHost: %v", err)
	}
}

func TestCancelPlay_HostCancelsWithoutRosterSlot(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "cancel-host-not-player")
	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))

	ts := setupHostPlayManagementTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/plays/"+playID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 204, body=%s", resp.StatusCode, string(b))
	}
	play, err := queries.GetPlayByID(ctx, playID)
	if err != nil {
		t.Fatalf("GetPlayByID: %v", err)
	}
	if play.CancelledAt == nil {
		t.Fatal("cancelled_at = nil, want timestamp")
	}
}

func TestCancelPlay_RejectsNonHost(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "delete-host-other")
	otherID := createRouteTestUser(t, ctx, queries, "delete-other")
	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))

	ts := setupHostPlayManagementTest(sessionWithProfile(otherID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/plays/"+playID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}
