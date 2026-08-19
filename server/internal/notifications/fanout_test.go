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

func TestFanOutToFollowersWritesOnceAndPushesEachFollower(t *testing.T) {
	sender := &recordingFollowerFanOutSender{
		followerIDs: []string{"follower-1", "follower-2"},
		fanoutDone:  make(chan struct{}),
		pushDone:    make(chan struct{}),
	}
	payload := Payload{
		Title:  "Alice is hosting a game",
		Body:   "Join the game",
		URL:    "/play/play-1",
		Tag:    "follow:hosted:play-1",
		Kind:   "follow.play_hosted",
		PlayID: "play-1",
	}

	FanOutToFollowers(sender, "actor-1", payload)

	select {
	case <-sender.fanoutDone:
	case <-time.After(time.Second):
		t.Fatal("fan-out did not run")
	}
	select {
	case <-sender.pushDone:
	case <-time.After(time.Second):
		t.Fatal("push fan-out did not complete")
	}

	sender.mu.Lock()
	defer sender.mu.Unlock()
	if sender.fanoutCalls != 1 {
		t.Fatalf("fan-out calls = %d, want 1", sender.fanoutCalls)
	}
	if sender.fanout.ActorUserID != "actor-1" || sender.fanout.Title != payload.Title || sender.fanout.Kind == nil || *sender.fanout.Kind != payload.Kind {
		t.Fatalf("fan-out params = %+v, want actor and payload", sender.fanout)
	}
	got := append([]string(nil), sender.pushedIDs...)
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

type recordingFollowerFanOutSender struct {
	mu          sync.Mutex
	followerIDs []string
	fanout      db.FanOutFollowerNotificationsParams
	fanoutCalls int
	pushedIDs   []string
	fanoutDone  chan struct{}
	pushDone    chan struct{}
}

func (s *recordingFollowerFanOutSender) FanOutFollowerNotifications(_ context.Context, arg db.FanOutFollowerNotificationsParams) error {
	s.mu.Lock()
	s.fanout = arg
	s.fanoutCalls++
	s.mu.Unlock()
	close(s.fanoutDone)
	return nil
}

func (s *recordingFollowerFanOutSender) ListAcceptedFollowerIDs(context.Context, string) ([]string, error) {
	return s.followerIDs, nil
}

func (s *recordingFollowerFanOutSender) DispatchFollowerPush(userID string, _ Payload) {
	s.mu.Lock()
	s.pushedIDs = append(s.pushedIDs, userID)
	if len(s.pushedIDs) == len(s.followerIDs) {
		close(s.pushDone)
	}
	s.mu.Unlock()
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
