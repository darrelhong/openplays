package plays_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

func createRequireWaitlistPlay(t *testing.T, ctx context.Context, queries *db.Queries, creatorID string, maxPlayers int64) string {
	t.Helper()

	startsAt := time.Now().UTC().Add(24 * time.Hour)
	levelMin, levelMax := ptrString("MB"), ptrString("HI")
	play, err := queries.CreatePlay(ctx, db.CreatePlayParams{
		ID:              uuid.NewString(),
		ListingType:     model.ListingPlay,
		Sport:           model.SportBadminton,
		HostName:        "Host",
		StartsAt:        startsAt,
		EndsAt:          startsAt.Add(2 * time.Hour),
		Timezone:        "Asia/Singapore",
		Venue:           "SBH",
		LevelMin:        levelMin,
		LevelMax:        levelMax,
		LevelMinOrd:     levelOrdPtr(model.SportBadminton, levelMin),
		LevelMaxOrd:     levelOrdPtr(model.SportBadminton, levelMax),
		Currency:        "SGD",
		MaxPlayers:      &maxPlayers,
		SlotsLeft:       &maxPlayers,
		CreatedBy:       &creatorID,
		RequireWaitlist: true,
	})
	if err != nil {
		t.Fatalf("CreatePlay: %v", err)
	}
	if _, err := queries.CreatePlayHost(ctx, db.CreatePlayHostParams{
		PlayID: play.ID,
		UserID: creatorID,
	}); err != nil {
		t.Fatalf("CreatePlayHost: %v", err)
	}
	return play.ID
}

func seedRequestedParticipant(t *testing.T, ctx context.Context, queries *db.Queries, playID, userID string) db.PlayParticipant {
	t.Helper()

	participant, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: playID,
		UserID: &userID,
		Status: model.ParticipantRequested,
	})
	if err != nil {
		t.Fatalf("CreatePlayParticipant requested: %v", err)
	}
	return participant
}

func latestPlayEventType(t *testing.T, ctx context.Context, queries *db.Queries, playID string) model.PlayEventType {
	t.Helper()

	events, err := queries.ListHostVisiblePlayEvents(ctx, db.ListHostVisiblePlayEventsParams{
		PlayID: playID,
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("ListHostVisiblePlayEvents: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("no play events recorded")
	}
	return events[0].EventType
}

func TestParticipantVisibleEventsIncludeJoinAndRequestEvents(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "events-creator")
	playID := createRequireWaitlistPlay(t, ctx, queries, creatorID, 4)

	// Roster events are participant-visible; pending-queue events carry
	// identities that are host-only, like the queue itself
	visible := []model.PlayEventType{
		model.PlayEventParticipantJoined,
		model.PlayEventParticipantAdded,
		model.PlayEventParticipantConfirmed,
		model.PlayEventParticipantLeftConfirmed,
		model.PlayEventParticipantLeftAdded,
	}
	hostOnly := []model.PlayEventType{
		model.PlayEventParticipantJoinRequested,
		model.PlayEventParticipantMovedToWaitlist,
		model.PlayEventParticipantRequestWithdrawn,
		model.PlayEventParticipantLeftWaitlist,
	}
	for _, eventType := range append(append([]model.PlayEventType{}, visible...), hostOnly...) {
		if _, err := queries.CreatePlayEvent(ctx, db.CreatePlayEventParams{
			PlayID:    playID,
			EventType: eventType,
		}); err != nil {
			t.Fatalf("CreatePlayEvent %s: %v", eventType, err)
		}
	}

	events, err := queries.ListParticipantVisiblePlayEvents(ctx, db.ListParticipantVisiblePlayEventsParams{
		PlayID: playID,
		Limit:  50,
	})
	if err != nil {
		t.Fatalf("ListParticipantVisiblePlayEvents: %v", err)
	}
	got := make(map[model.PlayEventType]bool, len(events))
	for _, event := range events {
		got[event.EventType] = true
	}
	for _, eventType := range visible {
		if !got[eventType] {
			t.Errorf("event %s missing from participant-visible feed", eventType)
		}
	}
	for _, eventType := range hostOnly {
		if got[eventType] {
			t.Errorf("event %s leaked into the participant-visible feed", eventType)
		}
	}
}

func TestJoinRequireWaitlistPlay_AlwaysRequestsEvenWithSlotAndLevel(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "rw-creator")
	joinerID := createRouteTestUser(t, ctx, queries, "rw-joiner")

	playID := createRequireWaitlistPlay(t, ctx, queries, creatorID, 4)
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	if err := queries.UpdatePlaySlotsLeft(ctx, playID); err != nil {
		t.Fatalf("UpdatePlaySlotsLeft: %v", err)
	}

	notifier := &fakeNotificationSender{}
	// Matching level and plenty of slots: classic mode would auto-confirm
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
	if out.Status != string(model.ParticipantRequested) {
		t.Fatalf("status = %q, want requested", out.Status)
	}
	// Requests never reserve capacity
	if out.SlotsLeft == nil || *out.SlotsLeft != 3 {
		t.Fatalf("slots_left = %v, want 3", out.SlotsLeft)
	}

	participant, err := queries.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{
		PlayID: playID,
		UserID: &joinerID,
	})
	if err != nil {
		t.Fatalf("GetPlayParticipantByPlayAndUser: %v", err)
	}
	if participant.Status != model.ParticipantRequested {
		t.Fatalf("participant status = %q, want requested", participant.Status)
	}
	// Rating is still stored so hosts can see the requester's level
	if participant.RatingCode == nil || *participant.RatingCode != "HB" {
		t.Fatalf("rating_code = %v, want HB", participant.RatingCode)
	}

	if got := latestPlayEventType(t, ctx, queries, playID); got != model.PlayEventParticipantJoinRequested {
		t.Fatalf("event type = %q, want participant.join_requested", got)
	}

	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.userID != creatorID {
		t.Fatalf("notification user = %q, want host %q", call.userID, creatorID)
	}
	if call.payload.Kind != "play.join_requested" {
		t.Fatalf("notification kind = %q, want play.join_requested", call.payload.Kind)
	}
}

func TestLeaveRequireWaitlistPlay_WithdrawsRequest(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "rw-leave-creator")
	requesterID := createRouteTestUser(t, ctx, queries, "rw-leave-requester")

	playID := createRequireWaitlistPlay(t, ctx, queries, creatorID, 4)
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedRequestedParticipant(t, ctx, queries, playID, requesterID)

	notifier := &fakeNotificationSender{}
	authStore := sessionWithProfile(requesterID, nil)
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

	if got := latestPlayEventType(t, ctx, queries, playID); got != model.PlayEventParticipantRequestWithdrawn {
		t.Fatalf("event type = %q, want participant.request_withdrawn", got)
	}
	// Withdrawing a request is silent, like leaving the waitlist
	if len(notifier.calls) != 0 {
		t.Fatalf("notification calls = %d, want 0", len(notifier.calls))
	}
}

func TestHostAcceptRequestedParticipant_AddsWithSlot(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "rw-accept-creator")
	requesterID := createRouteTestUser(t, ctx, queries, "rw-accept-requester")

	playID := createRequireWaitlistPlay(t, ctx, queries, creatorID, 4)
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	participant := seedRequestedParticipant(t, ctx, queries, playID, requesterID)

	notifier := &fakeNotificationSender{}
	authStore := sessionWithProfile(creatorID, nil)
	ts := setupHostRosterTest(authStore, queries, notifier)
	defer ts.Close()

	url := fmt.Sprintf("%s/api/plays/%s/participants/%d/accept", ts.URL, playID, participant.ID)
	req, _ := http.NewRequest(http.MethodPost, url, nil)
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
	if out.Status != string(model.ParticipantAdded) {
		t.Fatalf("status = %q, want added", out.Status)
	}

	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	if notifier.calls[0].payload.Kind != "play.player_added" {
		t.Fatalf("notification kind = %q, want play.player_added", notifier.calls[0].payload.Kind)
	}
}

func TestHostAcceptRequestedParticipant_RejectsWhenFull(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "rw-full-creator")
	otherID := createRouteTestUser(t, ctx, queries, "rw-full-other")
	requesterID := createRouteTestUser(t, ctx, queries, "rw-full-requester")

	playID := createRequireWaitlistPlay(t, ctx, queries, creatorID, 2)
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedConfirmedParticipant(t, ctx, queries, playID, otherID)
	participant := seedRequestedParticipant(t, ctx, queries, playID, requesterID)

	authStore := sessionWithProfile(creatorID, nil)
	ts := setupHostRosterTest(authStore, queries)
	defer ts.Close()

	url := fmt.Sprintf("%s/api/plays/%s/participants/%d/accept", ts.URL, playID, participant.ID)
	req, _ := http.NewRequest(http.MethodPost, url, nil)
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

func TestHostWaitlistRequestedParticipant(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "rw-park-creator")
	requesterID := createRouteTestUser(t, ctx, queries, "rw-park-requester")

	playID := createRequireWaitlistPlay(t, ctx, queries, creatorID, 4)
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	participant := seedRequestedParticipant(t, ctx, queries, playID, requesterID)

	notifier := &fakeNotificationSender{}
	authStore := sessionWithProfile(creatorID, nil)
	ts := setupHostRosterTest(authStore, queries, notifier)
	defer ts.Close()

	url := fmt.Sprintf("%s/api/plays/%s/participants/%d/waitlist", ts.URL, playID, participant.ID)
	req, _ := http.NewRequest(http.MethodPost, url, nil)
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
	// Capacity unchanged: neither requested nor waitlisted reserve a slot
	if out.SlotsLeft == nil || *out.SlotsLeft != 3 {
		t.Fatalf("slots_left = %v, want 3", out.SlotsLeft)
	}

	if got := latestPlayEventType(t, ctx, queries, playID); got != model.PlayEventParticipantMovedToWaitlist {
		t.Fatalf("event type = %q, want participant.moved_to_waitlist", got)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.userID != requesterID {
		t.Fatalf("notification user = %q, want requester %q", call.userID, requesterID)
	}
	if call.payload.Kind != "play.moved_to_waitlist" {
		t.Fatalf("notification kind = %q, want play.moved_to_waitlist", call.payload.Kind)
	}

	// A non-requested participant cannot be waitlisted through this endpoint
	resp2, err := http.DefaultClient.Do(req.Clone(ctx))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("second waitlist status = %d, want 409", resp2.StatusCode)
	}
}

func TestHostAcceptWaitlistedParticipant_StillWorksOnRequireWaitlistPlay(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "rw-wl-creator")
	waitlistedID := createRouteTestUser(t, ctx, queries, "rw-wl-player")

	playID := createRequireWaitlistPlay(t, ctx, queries, creatorID, 4)
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	participant := seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)

	authStore := sessionWithProfile(creatorID, nil)
	ts := setupHostRosterTest(authStore, queries)
	defer ts.Close()

	url := fmt.Sprintf("%s/api/plays/%s/participants/%d/accept", ts.URL, playID, participant.ID)
	req, _ := http.NewRequest(http.MethodPost, url, nil)
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
