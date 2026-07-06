package reviews_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/notifications"
	"openplays/server/internal/reviews"
	"openplays/server/internal/testdb"
)

type sentPrompt struct {
	userID  string
	payload notifications.Payload
}

type fakeSender struct {
	calls   []sentPrompt
	failFor map[string]bool
}

func (f *fakeSender) Notify(_ context.Context, userID string, payload notifications.Payload) error {
	if f.failFor[userID] {
		return errors.New("push exploded")
	}
	f.calls = append(f.calls, sentPrompt{userID: userID, payload: payload})
	return nil
}

func (f *fakeSender) userIDs() []string {
	out := make([]string, 0, len(f.calls))
	for _, call := range f.calls {
		out = append(out, call.userID)
	}
	return out
}

func createPromptTestUser(t *testing.T, ctx context.Context, queries *db.Queries, id string) string {
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

// createEndedPlay inserts a user-created play whose window ended endedAgo ago
// (negative values put the end in the future), with the host confirmed.
func createEndedPlay(t *testing.T, ctx context.Context, sqlDB *sql.DB, queries *db.Queries, id, hostID string, endedAgo time.Duration) {
	t.Helper()

	const timeFormat = "2006-01-02 15:04:05+00:00"
	endsAt := time.Now().UTC().Add(-endedAgo)
	startsAt := endsAt.Add(-2 * time.Hour)
	// host_name reuses the play id to dodge the plays dedupe unique index
	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO plays (id, listing_type, sport, host_name, starts_at, ends_at, timezone, venue, currency, max_players, slots_left, created_by, source)
		VALUES (?, 'play', 'badminton', ?, ?, ?, 'Asia/Singapore', 'Test Hall', 'SGD', 4, 2, ?, 'user')`,
		id, id, startsAt.Format(timeFormat), endsAt.Format(timeFormat), hostID,
	); err != nil {
		t.Fatalf("insert play %q: %v", id, err)
	}
	if _, err := queries.CreatePlayHost(ctx, db.CreatePlayHostParams{PlayID: id, UserID: hostID}); err != nil {
		t.Fatalf("create play host: %v", err)
	}
	seedPromptParticipant(t, ctx, queries, id, hostID)
}

func seedPromptParticipant(t *testing.T, ctx context.Context, queries *db.Queries, playID, userID string) {
	t.Helper()

	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: playID,
		UserID: &userID,
		Status: model.ParticipantConfirmed,
	}); err != nil {
		t.Fatalf("create participant: %v", err)
	}
}

func runTick(t *testing.T, prompter *reviews.Prompter) {
	t.Helper()
	prompter.Tick(context.Background())
}

func TestPrompter_NotifiesEligibleParticipantsOnce(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	hostID := createPromptTestUser(t, ctx, queries, "prompt-host")
	playerID := createPromptTestUser(t, ctx, queries, "prompt-player")
	waitlistedID := createPromptTestUser(t, ctx, queries, "prompt-waitlisted")

	createEndedPlay(t, ctx, sqlDB, queries, "prompt-play", hostID, 2*time.Hour)
	seedPromptParticipant(t, ctx, queries, "prompt-play", playerID)
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: "prompt-play",
		UserID: &waitlistedID,
		Status: model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("create waitlisted participant: %v", err)
	}

	sender := &fakeSender{}
	prompter := reviews.NewPrompter(queries, sender)

	runTick(t, prompter)

	if len(sender.calls) != 2 {
		t.Fatalf("notified %v, want the host and the confirmed player", sender.userIDs())
	}
	notified := map[string]bool{}
	for _, call := range sender.calls {
		notified[call.userID] = true
		if call.payload.Kind != "play.review_prompt" {
			t.Fatalf("kind = %q, want play.review_prompt", call.payload.Kind)
		}
		if call.payload.URL != "/play/prompt-play" {
			t.Fatalf("url = %q, want the play page", call.payload.URL)
		}
	}
	if !notified[hostID] || !notified[playerID] || notified[waitlistedID] {
		t.Fatalf("notified = %v", sender.userIDs())
	}

	// A rescan sends nothing new: the marker rows make prompts at-most-once
	runTick(t, prompter)
	if len(sender.calls) != 2 {
		t.Fatalf("second tick re-notified: %v", sender.userIDs())
	}
}

func TestPrompter_SkipsIneligiblePlays(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	hostID := createPromptTestUser(t, ctx, queries, "prompt-skip-host")
	playerID := createPromptTestUser(t, ctx, queries, "prompt-skip-player")

	// Still in progress: not ended, nothing to review yet
	createEndedPlay(t, ctx, sqlDB, queries, "prompt-live", hostID, -time.Hour)
	seedPromptParticipant(t, ctx, queries, "prompt-live", playerID)

	// Ended too long ago: outside the 72h backstop (pre-feature plays)
	createEndedPlay(t, ctx, sqlDB, queries, "prompt-old", hostID, 4*24*time.Hour)
	seedPromptParticipant(t, ctx, queries, "prompt-old", playerID)

	// Cancelled plays are never reviewable
	createEndedPlay(t, ctx, sqlDB, queries, "prompt-cancelled", hostID, 2*time.Hour)
	seedPromptParticipant(t, ctx, queries, "prompt-cancelled", playerID)
	if _, err := queries.CancelUserCreatedPlay(ctx, db.CancelUserCreatedPlayParams{
		ID:          "prompt-cancelled",
		CancelledBy: &hostID,
	}); err != nil {
		t.Fatalf("cancel play: %v", err)
	}

	// A lone player has no co-players to review
	createEndedPlay(t, ctx, sqlDB, queries, "prompt-solo", hostID, 2*time.Hour)

	sender := &fakeSender{}
	runTick(t, reviews.NewPrompter(queries, sender))

	if len(sender.calls) != 0 {
		t.Fatalf("notified %v, want nobody", sender.userIDs())
	}
}

func TestNextTickAfter(t *testing.T) {
	base := time.Date(2026, 7, 6, 17, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{"on the hour", base, base.Add(time.Minute)},
		{"just before the mark", base.Add(59 * time.Second), base.Add(time.Minute)},
		{"exactly on the mark", base.Add(time.Minute), base.Add(6 * time.Minute)},
		{"between marks", base.Add(3 * time.Minute), base.Add(6 * time.Minute)},
		{"end of hour", base.Add(59 * time.Minute), base.Add(61 * time.Minute)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := reviews.NextTickAfter(tc.now)
			if !got.Equal(tc.want) {
				t.Fatalf("NextTickAfter(%v) = %v, want %v", tc.now, got, tc.want)
			}
		})
	}
}

func TestPrompter_NotifyFailureIsAtMostOnce(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	hostID := createPromptTestUser(t, ctx, queries, "prompt-fail-host")
	playerID := createPromptTestUser(t, ctx, queries, "prompt-fail-player")
	createEndedPlay(t, ctx, sqlDB, queries, "prompt-fail", hostID, 2*time.Hour)
	seedPromptParticipant(t, ctx, queries, "prompt-fail", playerID)

	sender := &fakeSender{failFor: map[string]bool{hostID: true}}
	prompter := reviews.NewPrompter(queries, sender)

	runTick(t, prompter)

	// The failed nudge doesn't block the other participant
	if len(sender.calls) != 1 || sender.calls[0].userID != playerID {
		t.Fatalf("notified %v, want just the player", sender.userIDs())
	}

	// Marked before notifying: the failed user is not retried
	sender.failFor = nil
	runTick(t, prompter)
	if len(sender.calls) != 1 {
		t.Fatalf("failed prompt was retried: %v", sender.userIDs())
	}
}
