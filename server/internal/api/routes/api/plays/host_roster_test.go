package plays_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/plays"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/notifications"
	"openplays/server/internal/testdb"
)

func setupHostRosterTest(authStore *fakeAuthStore, store *db.Queries, notifiers ...notifications.Sender) *httptest.Server {
	svc := auth.NewService(authStore)
	var notifier notifications.Sender
	if len(notifiers) > 0 {
		notifier = notifiers[0]
	}

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/plays")
	plays.RegisterHostRosterManagement(grp, store, authmw.RequireAuth(api, svc), notifier)

	return httptest.NewServer(r)
}

func TestHostAddWaitlistedParticipant_NotifiesAddedPlayer(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "host-notify-added")
	waitlistedID := createRouteTestUser(t, ctx, queries, "player-notify-added")
	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	waitlisted := seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	notifier := &fakeNotificationSender{}
	ts := setupHostRosterTest(sessionWithProfile(creatorID, nil), queries, notifier)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/plays/%s/participants/%d/accept", ts.URL, playID, waitlisted.ID), nil)
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
	if call.userID != waitlistedID {
		t.Fatalf("notification user = %q, want added player %q", call.userID, waitlistedID)
	}
	if call.payload.Kind != "play.player_added" {
		t.Fatalf("notification kind = %q, want play.player_added", call.payload.Kind)
	}
	if call.payload.URL != "/play/"+playID {
		t.Fatalf("notification url = %q, want /play/%s", call.payload.URL, playID)
	}
}

func TestHostAddWaitlistedParticipant_WhenSlotOpen(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "host-accept-creator")
	waitlistedID := createRouteTestUser(t, ctx, queries, "host-accept-waitlisted")
	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	waitlisted := seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	ts := setupHostRosterTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/plays/%s/participants/%d/accept", ts.URL, playID, waitlisted.ID), nil)
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
	if out.Status != string(model.ParticipantAdded) {
		t.Fatalf("status = %q, want added", out.Status)
	}
	if out.SlotsLeft == nil || *out.SlotsLeft != 1 {
		t.Fatalf("slots_left = %v, want 1", out.SlotsLeft)
	}

	got, err := queries.GetPlayParticipantByID(ctx, waitlisted.ID)
	if err != nil {
		t.Fatalf("GetPlayParticipantByID: %v", err)
	}
	if got.Status != model.ParticipantAdded {
		t.Fatalf("participant status = %q, want added", got.Status)
	}
}

func TestHostAddWaitlistedParticipant_HostCanAddWithoutRosterSlot(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "host-not-player")
	waitlistedID := createRouteTestUser(t, ctx, queries, "host-not-player-waitlisted")
	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	waitlisted := seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	ts := setupHostRosterTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/plays/%s/participants/%d/accept", ts.URL, playID, waitlisted.ID), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	got, err := queries.GetPlayParticipantByID(ctx, waitlisted.ID)
	if err != nil {
		t.Fatalf("GetPlayParticipantByID: %v", err)
	}
	if got.Status != model.ParticipantAdded {
		t.Fatalf("participant status = %q, want added", got.Status)
	}
}

func TestHostAcceptWaitlistedParticipant_RejectsCancelledPlay(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "host-cancelled-creator")
	waitlistedID := createRouteTestUser(t, ctx, queries, "host-cancelled-waitlisted")
	playID := createUserPlay(t, ctx, queries, creatorID, 3, ptrString("MB"), ptrString("HI"))
	waitlisted := seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)
	if _, err := queries.CancelUserCreatedPlay(ctx, db.CancelUserCreatedPlayParams{
		ID:          playID,
		CancelledBy: &creatorID,
	}); err != nil {
		t.Fatalf("CancelUserCreatedPlay: %v", err)
	}

	ts := setupHostRosterTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/plays/%s/participants/%d/accept", ts.URL, playID, waitlisted.ID), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
	got, err := queries.GetPlayParticipantByID(ctx, waitlisted.ID)
	if err != nil {
		t.Fatalf("GetPlayParticipantByID: %v", err)
	}
	if got.Status != model.ParticipantWaitlisted {
		t.Fatalf("participant status = %q, want waitlisted", got.Status)
	}
}

func TestHostAcceptWaitlistedParticipant_RejectsWhenFull(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "host-full-creator")
	existingID := createRouteTestUser(t, ctx, queries, "host-full-existing")
	waitlistedID := createRouteTestUser(t, ctx, queries, "host-full-waitlisted")
	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedAddedParticipant(t, ctx, queries, playID, existingID)
	waitlisted := seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	ts := setupHostRosterTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/plays/%s/participants/%d/accept", ts.URL, playID, waitlisted.ID), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}

	got, err := queries.GetPlayParticipantByID(ctx, waitlisted.ID)
	if err != nil {
		t.Fatalf("GetPlayParticipantByID: %v", err)
	}
	if got.Status != model.ParticipantWaitlisted {
		t.Fatalf("participant status = %q, want waitlisted", got.Status)
	}
}

func TestHostRemoveParticipant_FreesSlotWithoutAutoPromotingWaitlist(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "host-remove-creator")
	confirmedID := createRouteTestUser(t, ctx, queries, "host-remove-confirmed")
	waitlistedID := createRouteTestUser(t, ctx, queries, "host-remove-waitlisted")
	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	confirmed := seedConfirmedParticipantRow(t, ctx, queries, playID, confirmedID)
	waitlisted := seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	ts := setupHostRosterTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/plays/%s/participants/%d", ts.URL, playID, confirmed.ID), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if _, err := queries.GetPlayParticipantByID(ctx, confirmed.ID); err == nil {
		t.Fatal("expected confirmed participant to be removed")
	}

	stillWaitlisted, err := queries.GetPlayParticipantByID(ctx, waitlisted.ID)
	if err != nil {
		t.Fatalf("GetPlayParticipantByID waitlisted: %v", err)
	}
	if stillWaitlisted.Status != model.ParticipantWaitlisted {
		t.Fatalf("waitlisted status = %q, want waitlisted", stillWaitlisted.Status)
	}

	play, err := queries.GetPlayByID(ctx, playID)
	if err != nil {
		t.Fatalf("GetPlayByID: %v", err)
	}
	if play.SlotsLeft == nil || *play.SlotsLeft != 1 {
		t.Fatalf("slots_left = %v, want 1", play.SlotsLeft)
	}
}

func TestHostRemoveParticipant_RemovesWaitlistedParticipant(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "host-remove-wl-creator")
	waitlistedID := createRouteTestUser(t, ctx, queries, "host-remove-wl-player")
	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	waitlisted := seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)

	ts := setupHostRosterTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/plays/%s/participants/%d", ts.URL, playID, waitlisted.ID), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if _, err := queries.GetPlayParticipantByID(ctx, waitlisted.ID); err != sql.ErrNoRows {
		t.Fatalf("GetPlayParticipantByID err = %v, want sql.ErrNoRows", err)
	}
}

func TestHostRemoveParticipant_RejectsRemovingHost(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "host-remove-self")
	playID := createUserPlay(t, ctx, queries, creatorID, 2, ptrString("MB"), ptrString("HI"))
	hostParticipant := seedConfirmedParticipantRow(t, ctx, queries, playID, creatorID)

	ts := setupHostRosterTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/plays/%s/participants/%d", ts.URL, playID, hostParticipant.ID), nil)
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

func seedConfirmedParticipantRow(t *testing.T, ctx context.Context, queries *db.Queries, playID, userID string) db.PlayParticipant {
	t.Helper()

	participant, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: playID,
		UserID: &userID,
		Status: model.ParticipantConfirmed,
	})
	if err != nil {
		t.Fatalf("CreatePlayParticipant confirmed: %v", err)
	}
	return participant
}

func seedWaitlistedParticipant(t *testing.T, ctx context.Context, queries *db.Queries, playID, userID string) db.PlayParticipant {
	t.Helper()

	participant, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: playID,
		UserID: &userID,
		Status: model.ParticipantWaitlisted,
	})
	if err != nil {
		t.Fatalf("CreatePlayParticipant waitlisted: %v", err)
	}
	return participant
}
