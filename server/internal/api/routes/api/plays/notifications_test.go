package plays_test

import (
	"context"
	"net/http"
	"testing"

	"openplays/server/internal/db"
	"openplays/server/internal/notifications"
	"openplays/server/internal/testdb"
)

type notificationCall struct {
	userID  string
	payload notifications.Payload
}

type fakeNotificationSender struct {
	calls []notificationCall
}

func (f *fakeNotificationSender) Notify(_ context.Context, userID string, payload notifications.Payload) error {
	f.calls = append(f.calls, notificationCall{
		userID:  userID,
		payload: payload,
	})
	return nil
}

func TestCancelPlay_NotifiesReservedRosterOnly(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	hostID := createRouteTestUser(t, ctx, queries, "cancel-notify-host")
	confirmedID := createRouteTestUser(t, ctx, queries, "cancel-notify-confirmed")
	addedID := createRouteTestUser(t, ctx, queries, "cancel-notify-added")
	waitlistedID := createRouteTestUser(t, ctx, queries, "cancel-notify-waitlisted")
	requestedID := createRouteTestUser(t, ctx, queries, "cancel-notify-requested")
	invitedID := createRouteTestUser(t, ctx, queries, "cancel-notify-invited")
	playID := createUserPlay(t, ctx, queries, hostID, 6, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, confirmedID)
	seedAddedParticipant(t, ctx, queries, playID, addedID)
	seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)
	seedRequestedParticipant(t, ctx, queries, playID, requestedID)
	seedInvitedParticipant(t, ctx, queries, playID, invitedID)

	notifier := &fakeNotificationSender{}
	ts := setupHostPlayManagementTest(sessionWithProfile(hostID, nil), queries, notifier)
	defer ts.Close()

	deletePlay := func() *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/plays/"+playID, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	resp := deletePlay()
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("first cancel status = %d, want 204", resp.StatusCode)
	}

	if len(notifier.calls) != 2 {
		t.Fatalf("notification calls = %d, want 2", len(notifier.calls))
	}
	wantRecipients := map[string]bool{confirmedID: true, addedID: true}
	for _, call := range notifier.calls {
		if !wantRecipients[call.userID] {
			t.Fatalf("unexpected notification recipient = %q", call.userID)
		}
		if call.payload.Kind != "play.cancelled" {
			t.Fatalf("notification kind = %q, want play.cancelled", call.payload.Kind)
		}
		if call.payload.Body != "This game was cancelled" {
			t.Fatalf("notification body = %q, want cancellation copy", call.payload.Body)
		}
		if call.payload.URL != "/play/"+playID {
			t.Fatalf("notification url = %q, want /play/%s", call.payload.URL, playID)
		}
		if call.payload.Tag != "play:"+playID+":cancelled" {
			t.Fatalf("notification tag = %q, want cancellation tag", call.payload.Tag)
		}
	}

	resp = deletePlay()
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("second cancel status = %d, want 204", resp.StatusCode)
	}
	if len(notifier.calls) != 2 {
		t.Fatalf("notification calls after repeated cancel = %d, want 2", len(notifier.calls))
	}
}
