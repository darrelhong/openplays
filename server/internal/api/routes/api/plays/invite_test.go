package plays_test

import (
	"bytes"
	"context"
	"database/sql"
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

func setupInviteTest(authStore *fakeAuthStore, queries *db.Queries, notifiers ...notifications.Notifier) *httptest.Server {
	svc := auth.NewService(authStore)
	var notifier notifications.Notifier
	if len(notifiers) > 0 {
		notifier = notifiers[0]
	}

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/plays")
	plays.RegisterInviteParticipant(grp, queries, authmw.RequireAuth(api, svc), notifier)
	plays.RegisterAcceptInvite(grp, queries, authmw.RequireAuth(api, svc), notifier)
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

func postAcceptInvite(t *testing.T, tsURL, playID string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, tsURL+"/api/plays/"+playID+"/invites/me/accept", nil)
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

func seedInvitedParticipant(t *testing.T, ctx context.Context, queries *db.Queries, playID, userID string) db.PlayParticipant {
	t.Helper()
	participant, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: playID,
		UserID: &userID,
		Status: model.ParticipantInvited,
	})
	if err != nil {
		t.Fatalf("CreatePlayParticipant invited: %v", err)
	}
	return participant
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

func TestAcceptInvite_PlacementAndEvents(t *testing.T) {
	tests := []struct {
		name          string
		requireWait   bool
		full          bool
		levelMismatch bool
		wantStatus    model.PlayParticipantStatus
		wantEvent     model.PlayEventType
		wantSlots     int64
	}{
		{
			name:       "classic with an open slot",
			wantStatus: model.ParticipantConfirmed,
			wantEvent:  model.PlayEventParticipantInviteAccepted,
			wantSlots:  1,
		},
		{
			name:       "classic full",
			full:       true,
			wantStatus: model.ParticipantWaitlisted,
			wantEvent:  model.PlayEventParticipantMovedToWaitlist,
			wantSlots:  0,
		},
		{
			name:        "require waitlist",
			requireWait: true,
			wantStatus:  model.ParticipantRequested,
			wantEvent:   model.PlayEventParticipantJoinRequested,
			wantSlots:   3,
		},
		{
			name:          "level mismatch is ignored",
			levelMismatch: true,
			wantStatus:    model.ParticipantConfirmed,
			wantEvent:     model.PlayEventParticipantInviteAccepted,
			wantSlots:     1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB := testdb.New(t)
			queries := db.New(sqlDB)
			ctx := context.Background()
			hostID := createRouteTestUser(t, ctx, queries, "accept-invite-host-"+tc.name)
			inviteeID := createRouteTestUser(t, ctx, queries, "accept-invite-player-"+tc.name)

			var playID string
			switch {
			case tc.requireWait:
				playID = createRequireWaitlistPlay(t, ctx, queries, hostID, 4)
			case tc.levelMismatch:
				playID = createUserPlay(t, ctx, queries, hostID, 3, ptrString("HI"), ptrString("HI"))
			default:
				maxPlayers := int64(3)
				if tc.full {
					maxPlayers = 2
				}
				playID = createUserPlay(t, ctx, queries, hostID, maxPlayers, ptrString("MB"), ptrString("HI"))
			}
			seedConfirmedParticipant(t, ctx, queries, playID, hostID)
			if tc.full {
				existingID := createRouteTestUser(t, ctx, queries, "accept-invite-existing")
				seedConfirmedParticipant(t, ctx, queries, playID, existingID)
			}
			invited := seedInvitedParticipant(t, ctx, queries, playID, inviteeID)
			if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
				t.Fatalf("UpdatePlaySlotsLeft: %v", err)
			}

			notifier := &fakeNotificationSender{}
			authStore := sessionWithProfile(inviteeID, ptrString(`{"badminton":{"level":"HB"}}`))
			ts := setupInviteTest(authStore, queries, notifier)
			defer ts.Close()

			resp := postAcceptInvite(t, ts.URL, playID)
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
			if out.Status != string(tc.wantStatus) {
				t.Fatalf("status = %q, want %q", out.Status, tc.wantStatus)
			}
			if out.SlotsLeft == nil || *out.SlotsLeft != tc.wantSlots {
				t.Fatalf("slots_left = %v, want %d", out.SlotsLeft, tc.wantSlots)
			}

			got, err := queries.GetPlayParticipantByID(ctx, invited.ID)
			if err != nil {
				t.Fatalf("GetPlayParticipantByID: %v", err)
			}
			if got.Status != tc.wantStatus {
				t.Fatalf("participant status = %q, want %q", got.Status, tc.wantStatus)
			}
			if gotEvent := latestPlayEventType(t, ctx, queries, playID); gotEvent != tc.wantEvent {
				t.Fatalf("event type = %q, want %q", gotEvent, tc.wantEvent)
			}
			wantFollowerCall := tc.wantStatus == model.ParticipantConfirmed
			if got := len(notifier.followerCalls); (got == 1) != wantFollowerCall {
				t.Fatalf("follower notification calls = %d, want %t", got, wantFollowerCall)
			}
			if wantFollowerCall {
				call := notifier.followerCalls[0]
				if call.actorUserID != inviteeID || call.payload.Kind != "follow.play_joined" {
					t.Fatalf("follower notification = %+v, want invitee joined", call)
				}
				if call.payload.Tag != "follow:joined:"+inviteeID+":"+playID {
					t.Fatalf("follower notification tag = %q, want joined tag", call.payload.Tag)
				}
			}
			if len(notifier.calls) != 1 || notifier.calls[0].userID != hostID {
				t.Fatalf("notification calls = %#v, want one host notification", notifier.calls)
			}
			if notifier.calls[0].payload.Kind != "play.invite_accepted" {
				t.Fatalf("notification kind = %q, want play.invite_accepted", notifier.calls[0].payload.Kind)
			}
		})
	}
}

func TestAcceptInvite_RejectsNonInvitedParticipant(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	hostID := createRouteTestUser(t, ctx, queries, "accept-noninvited-host")
	inviteeID := createRouteTestUser(t, ctx, queries, "accept-noninvited-player")
	playID := createUserPlay(t, ctx, queries, hostID, 3, nil, nil)
	seedConfirmedParticipant(t, ctx, queries, playID, hostID)
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{PlayID: playID, UserID: &inviteeID, Status: model.ParticipantWaitlisted}); err != nil {
		t.Fatalf("seed waitlisted participant: %v", err)
	}

	ts := setupInviteTest(sessionWithProfile(inviteeID, nil), queries)
	defer ts.Close()
	resp := postAcceptInvite(t, ts.URL, playID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}

func TestDeclineInviteDeletesRowAndAllowsJoin(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	hostID := createRouteTestUser(t, ctx, queries, "decline-invite-host")
	inviteeID := createRouteTestUser(t, ctx, queries, "decline-invite-player")
	playID := createUserPlay(t, ctx, queries, hostID, 3, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, hostID)
	seedInvitedParticipant(t, ctx, queries, playID, inviteeID)

	notifier := &fakeNotificationSender{}
	ts := setupJoinLeaveTest(sessionWithProfile(inviteeID, ptrString(`{"badminton":{"level":"HB"}}`)), queries, notifier)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/plays/"+playID+"/participants/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("decline status = %d, want 204", resp.StatusCode)
	}
	if _, err := queries.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{PlayID: playID, UserID: &inviteeID}); err != sql.ErrNoRows {
		t.Fatalf("participant lookup err = %v, want sql.ErrNoRows", err)
	}
	if got := latestPlayEventType(t, ctx, queries, playID); got != model.PlayEventParticipantInviteDeclined {
		t.Fatalf("event type = %q, want participant.invite_declined", got)
	}
	if len(notifier.calls) != 1 || notifier.calls[0].userID != hostID || notifier.calls[0].payload.Kind != "play.invite_declined" {
		t.Fatalf("decline notifications = %#v, want one host invite_declined", notifier.calls)
	}

	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/plays/"+playID+"/join", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("join-after-decline status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode join response: %v", err)
	}
	if out.Status != string(model.ParticipantAdded) {
		t.Fatalf("join-after-decline status = %q, want added", out.Status)
	}
}

func TestHostCannotCancelInvite(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	hostID := createRouteTestUser(t, ctx, queries, "cancel-invite-host")
	inviteeID := createRouteTestUser(t, ctx, queries, "cancel-invite-player")
	playID := createUserPlay(t, ctx, queries, hostID, 3, nil, nil)
	invited := seedInvitedParticipant(t, ctx, queries, playID, inviteeID)
	notifier := &fakeNotificationSender{}
	ts := setupHostRosterTest(sessionWithProfile(hostID, nil), queries, notifier)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/plays/%s/participants/%d", ts.URL, playID, invited.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
	if got, err := queries.GetPlayParticipantByID(ctx, invited.ID); err != nil {
		t.Fatalf("GetPlayParticipantByID: %v", err)
	} else if got.Status != model.ParticipantInvited {
		t.Fatalf("participant status = %q, want invited", got.Status)
	}
	if len(notifier.calls) != 0 {
		t.Fatalf("cancel notifications = %d, want 0", len(notifier.calls))
	}
}

func TestCancelInviteByNonHostIsForbidden(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	hostID := createRouteTestUser(t, ctx, queries, "cancel-nonhost-host")
	viewerID := createRouteTestUser(t, ctx, queries, "cancel-nonhost-viewer")
	inviteeID := createRouteTestUser(t, ctx, queries, "cancel-nonhost-player")
	playID := createUserPlay(t, ctx, queries, hostID, 3, nil, nil)
	invited := seedInvitedParticipant(t, ctx, queries, playID, inviteeID)

	ts := setupHostRosterTest(sessionWithProfile(viewerID, nil), queries)
	defer ts.Close()
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/plays/%s/participants/%d", ts.URL, playID, invited.ID), nil)
	if err != nil {
		t.Fatal(err)
	}
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
			var problem struct {
				Detail string `json:"detail"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&problem); err != nil {
				t.Fatalf("decode problem response: %v", err)
			}
			if problem.Detail != "cannot invite this player" {
				t.Fatalf("detail = %q, want generic invite failure", problem.Detail)
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
