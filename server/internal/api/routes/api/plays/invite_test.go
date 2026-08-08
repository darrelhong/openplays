package plays_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func setupInviteTest(authStore *fakeAuthStore, queries *db.Queries, notifiers ...notifications.Sender) *httptest.Server {
	svc := auth.NewService(authStore)
	var notifier notifications.Sender
	if len(notifiers) > 0 {
		notifier = notifiers[0]
	}

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/plays")
	plays.RegisterInviteParticipant(grp, queries, authmw.RequireAuth(api, svc), notifier)
	return httptest.NewServer(r)
}

func postInvite(t *testing.T, tsURL, playID, userID string) *http.Response {
	t.Helper()
	body := bytes.NewBufferString(fmt.Sprintf(`{"user_id":%q}`, userID))
	req, err := http.NewRequest(http.MethodPost, tsURL+"/api/plays/"+playID+"/invites", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestInvitePlayer_HappyPathSnapshotsRatingAndNotifies(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	hostID := createRouteTestUser(t, ctx, queries, "invite-happy-host")
	inviteeID := createRouteTestUser(t, ctx, queries, "invite-happy-player")
	profile := `{"badminton":{"level":"HB"}}`
	if _, err := sqlDB.ExecContext(ctx, "UPDATE users SET sports_profile = ? WHERE id = ?", profile, inviteeID); err != nil {
		t.Fatalf("set invitee profile: %v", err)
	}
	playID := createUserPlay(t, ctx, queries, hostID, 3, ptrString("MB"), ptrString("HI"))
	initialPlay, err := queries.GetPlayByID(ctx, playID)
	if err != nil {
		t.Fatalf("GetPlayByID: %v", err)
	}

	notifier := &fakeNotificationSender{}
	ts := setupInviteTest(sessionWithProfile(hostID, nil), queries, notifier)
	defer ts.Close()

	resp := postInvite(t, ts.URL, playID, inviteeID)
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
	if out.Status != string(model.ParticipantInvited) {
		t.Fatalf("status = %q, want invited", out.Status)
	}

	participant, err := queries.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{
		PlayID: playID,
		UserID: &inviteeID,
	})
	if err != nil {
		t.Fatalf("GetPlayParticipantByPlayAndUser: %v", err)
	}
	if participant.Status != model.ParticipantInvited {
		t.Fatalf("participant status = %q, want invited", participant.Status)
	}
	if participant.InvitedByUserID == nil || *participant.InvitedByUserID != hostID {
		t.Fatalf("invited_by_user_id = %v, want %q", participant.InvitedByUserID, hostID)
	}
	if participant.RatingCode == nil || *participant.RatingCode != "HB" || participant.RatingOrd == nil {
		t.Fatalf("rating snapshot = (%v, %v), want HB and ordinal", participant.RatingCode, participant.RatingOrd)
	}
	updatedPlay, err := queries.GetPlayByID(ctx, playID)
	if err != nil {
		t.Fatalf("GetPlayByID after invite: %v", err)
	}
	if updatedPlay.SlotsLeft == nil || initialPlay.SlotsLeft == nil || *updatedPlay.SlotsLeft != *initialPlay.SlotsLeft {
		t.Fatalf("slots_left changed from %v to %v", initialPlay.SlotsLeft, updatedPlay.SlotsLeft)
	}

	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.userID != inviteeID {
		t.Fatalf("notification user = %q, want %q", call.userID, inviteeID)
	}
	if call.payload.Kind != "play.invited" || call.payload.Body != "Test User invited you to the game" {
		t.Fatalf("notification = (%q, %q), want play.invited and host invite body", call.payload.Kind, call.payload.Body)
	}
	if call.payload.URL != "/play/"+playID || call.payload.Tag != "play:"+playID+":invited:"+inviteeID {
		t.Fatalf("notification link/tag = (%q, %q)", call.payload.URL, call.payload.Tag)
	}

	events, err := queries.ListHostVisiblePlayEvents(ctx, db.ListHostVisiblePlayEventsParams{PlayID: playID, Limit: 10})
	if err != nil {
		t.Fatalf("ListHostVisiblePlayEvents: %v", err)
	}
	if len(events) == 0 || events[0].EventType != model.PlayEventParticipantInvited {
		t.Fatalf("latest event = %v, want participant.invited", events)
	}
}

func TestInvitePlayer_RejectsNonHost(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	hostID := createRouteTestUser(t, ctx, queries, "invite-nonhost-host")
	viewerID := createRouteTestUser(t, ctx, queries, "invite-nonhost-viewer")
	inviteeID := createRouteTestUser(t, ctx, queries, "invite-nonhost-player")
	playID := createUserPlay(t, ctx, queries, hostID, 3, nil, nil)

	ts := setupInviteTest(sessionWithProfile(viewerID, nil), queries)
	defer ts.Close()
	resp := postInvite(t, ts.URL, playID, inviteeID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestInvitePlayer_RejectsSelf(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	hostID := createRouteTestUser(t, ctx, queries, "invite-self-host")
	playID := createUserPlay(t, ctx, queries, hostID, 3, nil, nil)

	ts := setupInviteTest(sessionWithProfile(hostID, nil), queries)
	defer ts.Close()
	resp := postInvite(t, ts.URL, playID, hostID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestInvitePlayer_RejectsDuplicateStatus(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	hostID := createRouteTestUser(t, ctx, queries, "invite-duplicate-host")
	inviteeID := createRouteTestUser(t, ctx, queries, "invite-duplicate-player")
	playID := createUserPlay(t, ctx, queries, hostID, 3, nil, nil)
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{PlayID: playID, UserID: &inviteeID, Status: model.ParticipantWaitlisted}); err != nil {
		t.Fatalf("seed participant: %v", err)
	}

	ts := setupInviteTest(sessionWithProfile(hostID, nil), queries)
	defer ts.Close()
	resp := postInvite(t, ts.URL, playID, inviteeID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}

func TestInvitePlayer_RejectsAtInvitationCap(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	hostID := createRouteTestUser(t, ctx, queries, "invite-cap-host")
	firstID := createRouteTestUser(t, ctx, queries, "invite-cap-first")
	secondID := createRouteTestUser(t, ctx, queries, "invite-cap-second")
	inviteeID := createRouteTestUser(t, ctx, queries, "invite-cap-player")
	playID := createUserPlay(t, ctx, queries, hostID, 2, nil, nil)
	for _, userID := range []string{firstID, secondID} {
		if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{PlayID: playID, UserID: &userID, Status: model.ParticipantInvited}); err != nil {
			t.Fatalf("seed invite: %v", err)
		}
	}

	ts := setupInviteTest(sessionWithProfile(hostID, nil), queries)
	defer ts.Close()
	resp := postInvite(t, ts.URL, playID, inviteeID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}

func TestInvitePlayer_RejectsBlockedUserInEitherDirection(t *testing.T) {
	for _, tc := range []struct {
		name      string
		blockerID string
		blockedID string
	}{
		{name: "host blocks invitee", blockerID: "invite-block-host", blockedID: "invite-block-player"},
		{name: "invitee blocks host", blockerID: "invite-block-player-reverse", blockedID: "invite-block-host-reverse"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB := testdb.New(t)
			queries := db.New(sqlDB)
			ctx := context.Background()
			hostID, inviteeID := tc.blockerID, tc.blockedID
			if tc.name == "invitee blocks host" {
				hostID, inviteeID = tc.blockedID, tc.blockerID
			}
			createRouteTestUser(t, ctx, queries, hostID)
			createRouteTestUser(t, ctx, queries, inviteeID)
			if err := queries.CreateBlock(ctx, db.CreateBlockParams{BlockerID: tc.blockerID, BlockedID: tc.blockedID}); err != nil {
				t.Fatalf("CreateBlock: %v", err)
			}
			playID := createUserPlay(t, ctx, queries, hostID, 3, nil, nil)

			ts := setupInviteTest(sessionWithProfile(hostID, nil), queries)
			defer ts.Close()
			resp := postInvite(t, ts.URL, playID, inviteeID)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", resp.StatusCode)
			}
		})
	}
}

func TestInvitePlayer_RejectsInactiveInvitee(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	hostID := createRouteTestUser(t, ctx, queries, "invite-inactive-host")
	inviteeID := createRouteTestUser(t, ctx, queries, "invite-inactive-player")
	if err := queries.UpdateUserStatus(ctx, db.UpdateUserStatusParams{ID: inviteeID, Status: "suspended"}); err != nil {
		t.Fatalf("UpdateUserStatus: %v", err)
	}
	playID := createUserPlay(t, ctx, queries, hostID, 3, nil, nil)

	ts := setupInviteTest(sessionWithProfile(hostID, nil), queries)
	defer ts.Close()
	resp := postInvite(t, ts.URL, playID, inviteeID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestInvitePlayer_RejectsImportedPlay(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	hostID := createRouteTestUser(t, ctx, queries, "invite-import-host")
	inviteeID := createRouteTestUser(t, ctx, queries, "invite-import-player")
	maxPlayers := int64(3)
	play, err := queries.CreatePlay(ctx, db.CreatePlayParams{
		ID: uuid.NewString(), ListingType: model.ListingPlay, Sport: model.SportBadminton,
		HostName: "Imported host", StartsAt: time.Now().UTC().Add(time.Hour), EndsAt: time.Now().UTC().Add(3 * time.Hour),
		Timezone: "Asia/Singapore", Venue: "Imported venue", Currency: "SGD", MaxPlayers: &maxPlayers, SlotsLeft: &maxPlayers,
	})
	if err != nil {
		t.Fatalf("CreatePlay: %v", err)
	}

	ts := setupInviteTest(sessionWithProfile(hostID, nil), queries)
	defer ts.Close()
	resp := postInvite(t, ts.URL, play.ID, inviteeID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestInvitePlayer_RejectsCancelledPlay(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	hostID := createRouteTestUser(t, ctx, queries, "invite-cancelled-host")
	inviteeID := createRouteTestUser(t, ctx, queries, "invite-cancelled-player")
	playID := createUserPlay(t, ctx, queries, hostID, 3, nil, nil)
	if _, err := queries.CancelUserCreatedPlay(ctx, db.CancelUserCreatedPlayParams{ID: playID, CancelledBy: &hostID}); err != nil {
		t.Fatalf("CancelUserCreatedPlay: %v", err)
	}

	ts := setupInviteTest(sessionWithProfile(hostID, nil), queries)
	defer ts.Close()
	resp := postInvite(t, ts.URL, playID, inviteeID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}

func TestInvitePlayer_RejectsEndedPlay(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	hostID := createRouteTestUser(t, ctx, queries, "invite-ended-host")
	inviteeID := createRouteTestUser(t, ctx, queries, "invite-ended-player")
	playID := createUserPlay(t, ctx, queries, hostID, 3, nil, nil)
	endedAt := time.Now().UTC().Add(-time.Hour)
	if _, err := sqlDB.ExecContext(ctx, "UPDATE plays SET starts_at = ?, ends_at = ? WHERE id = ?", endedAt.Add(-2*time.Hour), endedAt, playID); err != nil {
		t.Fatalf("end play: %v", err)
	}

	ts := setupInviteTest(sessionWithProfile(hostID, nil), queries)
	defer ts.Close()
	resp := postInvite(t, ts.URL, playID, inviteeID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}
