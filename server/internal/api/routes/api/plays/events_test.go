package plays

import (
	"testing"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

func TestMapPlayHistoryEventsBuildsMessagesAndRelativeTime(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	hostName := "Host User"
	playerName := "Player One"

	events, err := mapPlayHistoryEvents([]db.PlayEvent{
		{
			ID:                 1,
			EventType:          model.PlayEventParticipantAdded,
			ActorDisplayName:   &hostName,
			SubjectDisplayName: &playerName,
			CreatedAt:          now.Add(-2 * time.Hour),
		},
		{
			ID:                 2,
			EventType:          model.PlayEventParticipantRemoved,
			ActorDisplayName:   &hostName,
			SubjectDisplayName: &playerName,
			CreatedAt:          now.Add(-5 * time.Minute),
		},
		{
			ID:               3,
			EventType:        model.PlayEventUpdated,
			ActorDisplayName: &hostName,
			CreatedAt:        now.Add(-30 * time.Second),
		},
	}, now)
	if err != nil {
		t.Fatalf("mapPlayHistoryEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("events len = %d, want 3", len(events))
	}

	if events[0].ActorDisplayName != nil {
		t.Fatalf("added actor_display_name = %v, want nil", *events[0].ActorDisplayName)
	}
	if events[0].Message != "Player One was added to the game" {
		t.Fatalf("added message = %q, want redacted actor copy", events[0].Message)
	}
	if events[0].RelativeTime != "about 2 hours ago" {
		t.Fatalf("added relative_time = %q, want about 2 hours ago", events[0].RelativeTime)
	}

	if events[1].ActorDisplayName != nil {
		t.Fatalf("removed actor_display_name = %v, want nil", *events[1].ActorDisplayName)
	}
	if events[1].Message != "Player One was removed from the game" {
		t.Fatalf("removed message = %q, want redacted actor copy", events[1].Message)
	}
	if events[1].RelativeTime != "5 minutes ago" {
		t.Fatalf("removed relative_time = %q, want 5 minutes ago", events[1].RelativeTime)
	}

	if events[2].ActorDisplayName == nil || *events[2].ActorDisplayName != hostName {
		t.Fatalf("updated actor_display_name = %v, want host", events[2].ActorDisplayName)
	}
	if events[2].Message != "Game updated" {
		t.Fatalf("updated message = %q, want Game updated", events[2].Message)
	}
	if events[2].RelativeTime != "1 minute ago" {
		t.Fatalf("updated relative_time = %q, want 1 minute ago", events[2].RelativeTime)
	}
}

func TestFormatPlayHistoryRelativeTimeMatchesDateFnsNonStrictThresholds(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		age  time.Duration
		want string
	}{
		{name: "under thirty seconds", age: 29*time.Second + 999*time.Millisecond, want: "less than a minute"},
		{name: "thirty seconds", age: 30 * time.Second, want: "1 minute"},
		{name: "under ninety seconds", age: 89*time.Second + 999*time.Millisecond, want: "1 minute"},
		{name: "ninety seconds", age: 90 * time.Second, want: "2 minutes"},
		{name: "under forty four and a half minutes", age: 44*time.Minute + 29*time.Second, want: "44 minutes"},
		{name: "forty four and a half minutes", age: 44*time.Minute + 30*time.Second, want: "about 1 hour"},
		{name: "under eighty nine and a half minutes", age: 89*time.Minute + 29*time.Second, want: "about 1 hour"},
		{name: "eighty nine and a half minutes", age: 89*time.Minute + 30*time.Second, want: "about 2 hours"},
		{name: "under one day", age: 23*time.Hour + 59*time.Minute + 29*time.Second, want: "about 24 hours"},
		{name: "one day", age: 23*time.Hour + 59*time.Minute + 30*time.Second, want: "1 day"},
		{name: "under two days", age: 41*time.Hour + 59*time.Minute + 29*time.Second, want: "1 day"},
		{name: "two days", age: 41*time.Hour + 59*time.Minute + 30*time.Second, want: "2 days"},
		{name: "thirty days", age: 29*24*time.Hour + 23*time.Hour + 59*time.Minute + 29*time.Second, want: "30 days"},
		{name: "about one month", age: 29*24*time.Hour + 23*time.Hour + 59*time.Minute + 30*time.Second, want: "about 1 month"},
		{name: "under about two months", age: 44*24*time.Hour + 23*time.Hour + 59*time.Minute + 29*time.Second, want: "about 1 month"},
		{name: "about two months", age: 44*24*time.Hour + 23*time.Hour + 59*time.Minute + 30*time.Second, want: "about 2 months"},
		{name: "two months", age: 59*24*time.Hour + 23*time.Hour + 59*time.Minute + 30*time.Second, want: "2 months"},
		{name: "twelve months", age: 364 * 24 * time.Hour, want: "12 months"},
		{name: "about one year", age: 365 * 24 * time.Hour, want: "about 1 year"},
		{name: "over one year", age: 365*24*time.Hour + 3*30*24*time.Hour, want: "over 1 year"},
		{name: "almost two years", age: 365*24*time.Hour + 9*30*24*time.Hour, want: "almost 2 years"},
		{name: "about two years", age: 2 * 365 * 24 * time.Hour, want: "about 2 years"},
		{name: "future", age: -90 * time.Second, want: "2 minutes"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPlayHistoryRelativeTime(now.Add(-tt.age), now, playHistoryRelativeTimeOptions{})
			if got != tt.want {
				t.Fatalf("formatPlayHistoryRelativeTime(%s) = %q, want %q", tt.age, got, tt.want)
			}
		})
	}
}

func TestFormatPlayHistoryRelativeTimeAddSuffix(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	options := playHistoryRelativeTimeOptions{AddSuffix: true}

	past := formatPlayHistoryRelativeTime(now.Add(-90*time.Second), now, options)
	if past != "2 minutes ago" {
		t.Fatalf("past relative time = %q, want 2 minutes ago", past)
	}

	future := formatPlayHistoryRelativeTime(now.Add(90*time.Second), now, options)
	if future != "in 2 minutes" {
		t.Fatalf("future relative time = %q, want in 2 minutes", future)
	}
}
