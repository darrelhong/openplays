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
	"openplays/server/internal/notifications"
	"openplays/server/internal/testdb"
)

func setupJoinLeaveTest(authStore *fakeAuthStore, store *db.Queries, notifiers ...notifications.Sender) *httptest.Server {
	svc := auth.NewService(authStore)
	var notifier notifications.Sender
	if len(notifiers) > 0 {
		notifier = notifiers[0]
	}

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/plays")
	plays.RegisterJoin(grp, store, authmw.RequireAuth(api, svc), notifier)
	plays.RegisterLeave(grp, store, authmw.RequireAuth(api, svc), notifier)
	plays.RegisterConfirmParticipant(grp, store, authmw.RequireAuth(api, svc), notifier)

	return httptest.NewServer(r)
}

func TestJoinPlay_WaitlistNotifiesHost(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "creator-notify-waitlist")
	existingID := createRouteTestUser(t, ctx, queries, "existing-notify-waitlist")
	joinerID := createRouteTestUser(t, ctx, queries, "joiner-notify-waitlist")

	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedConfirmedParticipant(t, ctx, queries, playID, existingID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	notifier := &fakeNotificationSender{}
	authStore := sessionWithProfile(joinerID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupJoinLeaveTest(authStore, queries, notifier)
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
	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.userID != creatorID {
		t.Fatalf("notification user = %q, want host %q", call.userID, creatorID)
	}
	if call.payload.Kind != "play.waitlist_joined" {
		t.Fatalf("notification kind = %q, want play.waitlist_joined", call.payload.Kind)
	}
	if call.payload.URL != "/play/"+playID {
		t.Fatalf("notification url = %q, want /play/%s", call.payload.URL, playID)
	}
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

	notifier := &fakeNotificationSender{}
	authStore := sessionWithProfile(joinerID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupJoinLeaveTest(authStore, queries, notifier)
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
	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.userID != creatorID {
		t.Fatalf("notification user = %q, want host %q", call.userID, creatorID)
	}
	if call.payload.Kind != "play.player_joined" {
		t.Fatalf("notification kind = %q, want play.player_joined", call.payload.Kind)
	}
	if call.payload.Body != "Test User joined the game" {
		t.Fatalf("notification body = %q, want player joined copy", call.payload.Body)
	}
	if call.payload.URL != "/play/"+playID {
		t.Fatalf("notification url = %q, want /play/%s", call.payload.URL, playID)
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

func TestJoinPlay_AutoWaitlistWhenAddedSpotReserved(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "creator-added-reserved")
	addedID := createRouteTestUser(t, ctx, queries, "added-reserved-player")
	joinerID := createRouteTestUser(t, ctx, queries, "joiner-added-reserved")

	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedAddedParticipant(t, ctx, queries, playID, addedID)
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

func TestJoinPlay_RejectsCancelledPlay(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "creator-cancelled-join")
	joinerID := createRouteTestUser(t, ctx, queries, "joiner-cancelled")
	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	if _, err := queries.CancelUserCreatedPlay(ctx, db.CancelUserCreatedPlayParams{
		ID:          playID,
		CancelledBy: &creatorID,
	}); err != nil {
		t.Fatalf("CancelUserCreatedPlay: %v", err)
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

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
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

func TestLeavePlay_NotifiesHostWhenPlayerLeavesGame(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "creator-notify-left")
	joinerID := createRouteTestUser(t, ctx, queries, "joiner-notify-left")

	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedConfirmedParticipant(t, ctx, queries, playID, joinerID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	notifier := &fakeNotificationSender{}
	authStore := sessionWithProfile(joinerID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupJoinLeaveTest(authStore, queries, notifier)
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
	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.userID != creatorID {
		t.Fatalf("notification user = %q, want host %q", call.userID, creatorID)
	}
	if call.payload.Kind != "play.player_left" {
		t.Fatalf("notification kind = %q, want play.player_left", call.payload.Kind)
	}
	if call.payload.Body != "Test User left the game" {
		t.Fatalf("notification body = %q, want player left copy", call.payload.Body)
	}
	if call.payload.URL != "/play/"+playID {
		t.Fatalf("notification url = %q, want /play/%s", call.payload.URL, playID)
	}
}

func TestLeavePlay_NotifiesHostWhenAddedPlayerLeavesGame(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "creator-notify-added-left")
	addedID := createRouteTestUser(t, ctx, queries, "added-notify-left")

	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedAddedParticipant(t, ctx, queries, playID, addedID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	notifier := &fakeNotificationSender{}
	authStore := sessionWithProfile(addedID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupJoinLeaveTest(authStore, queries, notifier)
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
	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.userID != creatorID {
		t.Fatalf("notification user = %q, want host %q", call.userID, creatorID)
	}
	if call.payload.Kind != "play.player_left" {
		t.Fatalf("notification kind = %q, want play.player_left", call.payload.Kind)
	}
	if call.payload.URL != "/play/"+playID {
		t.Fatalf("notification url = %q, want /play/%s", call.payload.URL, playID)
	}
}

func TestLeavePlay_DoesNotNotifyHostWhenPlayerLeavesWaitlist(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "creator-no-notify-waitlist-left")
	waitlistedID := createRouteTestUser(t, ctx, queries, "waitlisted-no-notify-left")

	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	notifier := &fakeNotificationSender{}
	authStore := sessionWithProfile(waitlistedID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupJoinLeaveTest(authStore, queries, notifier)
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
	if len(notifier.calls) != 0 {
		t.Fatalf("notification calls = %d, want 0", len(notifier.calls))
	}
}

func TestLeavePlay_RejectsHostLeavingRoster(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "creator-leave-host")
	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)

	authStore := sessionWithProfile(creatorID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupJoinLeaveTest(authStore, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/plays/"+playID+"/participants/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}

	if _, err := queries.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{
		PlayID: playID,
		UserID: &creatorID,
	}); err != nil {
		t.Fatalf("expected host participant to remain: %v", err)
	}
}

func TestConfirmParticipant_ChangesAddedToConfirmed(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "confirm-added-creator")
	playerID := createRouteTestUser(t, ctx, queries, "confirm-added-player")

	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedAddedParticipant(t, ctx, queries, playID, playerID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	notifier := &fakeNotificationSender{}
	authStore := sessionWithProfile(playerID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupJoinLeaveTest(authStore, queries, notifier)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/plays/"+playID+"/participants/me/confirm", nil)
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

	participant, err := queries.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{
		PlayID: playID,
		UserID: &playerID,
	})
	if err != nil {
		t.Fatalf("GetPlayParticipantByPlayAndUser: %v", err)
	}
	if participant.Status != model.ParticipantConfirmed {
		t.Fatalf("participant status = %q, want confirmed", participant.Status)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.userID != creatorID {
		t.Fatalf("notification user = %q, want host %q", call.userID, creatorID)
	}
	if call.payload.Kind != "play.player_confirmed" {
		t.Fatalf("notification kind = %q, want play.player_confirmed", call.payload.Kind)
	}
	if call.payload.Body != "Test User confirmed their spot" {
		t.Fatalf("notification body = %q, want player confirmed copy", call.payload.Body)
	}
	if call.payload.URL != "/play/"+playID {
		t.Fatalf("notification url = %q, want /play/%s", call.payload.URL, playID)
	}
}

func TestConfirmParticipant_RejectsWaitlistedParticipant(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "confirm-waitlisted-creator")
	playerID := createRouteTestUser(t, ctx, queries, "confirm-waitlisted-player")

	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: playID,
		UserID: &playerID,
		Status: model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant waitlisted: %v", err)
	}

	authStore := sessionWithProfile(playerID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupJoinLeaveTest(authStore, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/plays/"+playID+"/participants/me/confirm", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
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
	if _, err := queries.CreatePlayHost(ctx, db.CreatePlayHostParams{
		PlayID: play.ID,
		UserID: creatorID,
	}); err != nil {
		t.Fatalf("CreatePlayHost host: %v", err)
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

func seedAddedParticipant(t *testing.T, ctx context.Context, queries *db.Queries, playID, userID string) {
	t.Helper()

	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: playID,
		UserID: &userID,
		Status: model.ParticipantAdded,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant added: %v", err)
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
