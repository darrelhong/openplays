package notifications

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/testdb"
)

func TestNotifyFollowersWritesOnceAndPushesEachFollower(t *testing.T) {
	store := &recordingFollowerStore{
		followerIDs: []string{"follower-1", "follower-2"},
		fanoutDone:  make(chan struct{}),
		pushDone:    make(chan struct{}),
	}
	service := &WebPushService{
		publicKey:  "public-key",
		privateKey: "private-key",
		subscriber: "mailto:dev@openplays.app",
		store:      store,
	}
	payload := Payload{
		Title:  "Alice is hosting a game",
		Body:   "Join the game",
		URL:    "/play/play-1",
		Tag:    "follow:hosted:play-1",
		Kind:   "follow.play_hosted",
		PlayID: "play-1",
	}

	if err := service.NotifyFollowers(context.Background(), "actor-1", payload); err != nil {
		t.Fatalf("NotifyFollowers: %v", err)
	}

	select {
	case <-store.fanoutDone:
	case <-time.After(time.Second):
		t.Fatal("fan-out did not run")
	}
	select {
	case <-store.pushDone:
	case <-time.After(time.Second):
		t.Fatal("push fan-out did not complete")
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.fanoutCalls != 1 {
		t.Fatalf("fan-out calls = %d, want 1", store.fanoutCalls)
	}
	if store.fanout.ActorUserID != "actor-1" || store.fanout.Title != payload.Title || store.fanout.Kind == nil || *store.fanout.Kind != payload.Kind {
		t.Fatalf("fan-out params = %+v, want actor and payload", store.fanout)
	}
	got := append([]string(nil), store.pushedIDs...)
	sort.Strings(got)
	if want := []string{"follower-1", "follower-2"}; !equalStrings(got, want) {
		t.Fatalf("push recipients = %v, want %v", got, want)
	}
}

func TestFanOutFollowerNotificationsFiltersRecipients(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	for _, user := range []struct {
		id     string
		status string
	}{
		{id: "actor", status: "active"},
		{id: "accepted", status: "active"},
		{id: "blocked", status: "active"},
		{id: "inactive", status: "suspended"},
		{id: "pending", status: "active"},
	} {
		if _, err := sqlDB.ExecContext(ctx, `INSERT INTO users (id, email, display_name, status) VALUES (?, ?, ?, ?)`, user.id, user.id+"@example.com", user.id, user.status); err != nil {
			t.Fatalf("insert user %s: %v", user.id, err)
		}
	}
	for _, follow := range []struct {
		follower string
		status   string
	}{
		{follower: "accepted", status: "accepted"},
		{follower: "blocked", status: "accepted"},
		{follower: "inactive", status: "accepted"},
		{follower: "pending", status: "pending"},
	} {
		if _, err := queries.CreateUserFollow(ctx, db.CreateUserFollowParams{
			FollowerID: follow.follower, FolloweeID: "actor", Status: follow.status,
		}); err != nil {
			t.Fatalf("insert follow %s: %v", follow.follower, err)
		}
	}
	if _, err := sqlDB.ExecContext(ctx, `INSERT INTO user_blocks (blocker_id, blocked_id) VALUES (?, ?)`, "actor", "blocked"); err != nil {
		t.Fatalf("insert block: %v", err)
	}

	if err := queries.FanOutFollowerNotifications(ctx, db.FanOutFollowerNotificationsParams{
		ActorUserID: "actor",
		Title:       "Actor is hosting a game",
		Body:        stringPtr("Join the game"),
		Kind:        stringPtr("follow.play_hosted"),
		PlayID:      stringPtr("play-1"),
	}); err != nil {
		t.Fatalf("fan out notifications: %v", err)
	}

	var recipients []string
	rows, err := sqlDB.QueryContext(ctx, `SELECT user_id FROM user_notifications ORDER BY user_id`)
	if err != nil {
		t.Fatalf("list notification recipients: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			t.Fatalf("scan notification recipient: %v", err)
		}
		recipients = append(recipients, userID)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate notification recipients: %v", err)
	}
	if want := []string{"accepted"}; !equalStrings(recipients, want) {
		t.Fatalf("notification recipients = %v, want %v", recipients, want)
	}
}

type recordingFollowerStore struct {
	blockingWebPushStore
	mu          sync.Mutex
	followerIDs []string
	fanout      db.FanOutFollowerNotificationsParams
	fanoutCalls int
	pushedIDs   []string
	fanoutDone  chan struct{}
	pushDone    chan struct{}
}

func (s *recordingFollowerStore) FanOutFollowerNotifications(_ context.Context, arg db.FanOutFollowerNotificationsParams) error {
	s.mu.Lock()
	s.fanout = arg
	s.fanoutCalls++
	s.mu.Unlock()
	close(s.fanoutDone)
	return nil
}

func (s *recordingFollowerStore) ListAcceptedFollowerIDs(context.Context, string) ([]string, error) {
	return s.followerIDs, nil
}

func (s *recordingFollowerStore) ListWebPushSubscriptionsByUser(_ context.Context, userID string) ([]db.WebPushSubscription, error) {
	s.mu.Lock()
	s.pushedIDs = append(s.pushedIDs, userID)
	if len(s.pushedIDs) == len(s.followerIDs) {
		close(s.pushDone)
	}
	s.mu.Unlock()
	return nil, nil
}

func stringPtr(value string) *string {
	return &value
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
