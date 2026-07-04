package plays_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/testdb"
)

// markPlayEnded shifts a play into the past so the roster freeze applies.
func markPlayEnded(t *testing.T, ctx context.Context, sqlDB *sql.DB, playID string) {
	t.Helper()

	endsAt := time.Now().UTC().Add(-2 * time.Hour)
	startsAt := endsAt.Add(-2 * time.Hour)
	if _, err := sqlDB.ExecContext(ctx, "UPDATE plays SET starts_at = ?, ends_at = ? WHERE id = ?", startsAt, endsAt, playID); err != nil {
		t.Fatalf("mark play ended: %v", err)
	}
}

func TestEndedPlay_JoinRejected(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "freeze-join-creator")
	joinerID := createRouteTestUser(t, ctx, queries, "freeze-join-joiner")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, nil, nil)
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	markPlayEnded(t, ctx, sqlDB, playID)

	ts := setupJoinLeaveTest(sessionWithProfile(joinerID, ptrString(`{"badminton":{"level":"HB"}}`)), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/plays/"+playID+"/join", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("join ended play status = %d, want 409", resp.StatusCode)
	}
}

func TestEndedPlay_LeaveRejected(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "freeze-leave-creator")
	playerID := createRouteTestUser(t, ctx, queries, "freeze-leave-player")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, nil, nil)
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedConfirmedParticipant(t, ctx, queries, playID, playerID)
	markPlayEnded(t, ctx, sqlDB, playID)

	ts := setupJoinLeaveTest(sessionWithProfile(playerID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/plays/"+playID+"/participants/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("leave ended play status = %d, want 409", resp.StatusCode)
	}
}

func TestEndedPlay_ConfirmRejected(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "freeze-confirm-creator")
	playerID := createRouteTestUser(t, ctx, queries, "freeze-confirm-player")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, nil, nil)
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedAddedParticipant(t, ctx, queries, playID, playerID)
	markPlayEnded(t, ctx, sqlDB, playID)

	ts := setupJoinLeaveTest(sessionWithProfile(playerID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/plays/"+playID+"/participants/me/confirm", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("confirm on ended play status = %d, want 409", resp.StatusCode)
	}
}

func TestEndedPlay_HostRosterActionsRejected(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "freeze-host-creator")
	pendingID := createRouteTestUser(t, ctx, queries, "freeze-host-pending")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, nil, nil)
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	pending := seedWaitlistedParticipant(t, ctx, queries, playID, pendingID)
	markPlayEnded(t, ctx, sqlDB, playID)

	ts := setupHostRosterTest(sessionWithProfile(creatorID, nil), queries)
	defer ts.Close()

	// Accept, waitlist, and remove all pass through the shared ended-play guard
	requests := []struct {
		method string
		url    string
	}{
		{http.MethodPost, fmt.Sprintf("%s/api/plays/%s/participants/%d/accept", ts.URL, playID, pending.ID)},
		{http.MethodPost, fmt.Sprintf("%s/api/plays/%s/participants/%d/waitlist", ts.URL, playID, pending.ID)},
		{http.MethodDelete, fmt.Sprintf("%s/api/plays/%s/participants/%d", ts.URL, playID, pending.ID)},
	}
	for _, r := range requests {
		req, _ := http.NewRequest(r.method, r.url, nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("%s %s on ended play status = %d, want 409", r.method, r.url, resp.StatusCode)
		}
	}
}
