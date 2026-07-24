package pipeline

import (
	"testing"
	"time"

	"openplays/server/internal/model"
)

func TestToUpsertPlayParams_NormalizesHostNameForDeduplication(t *testing.T) {
	start := time.Date(2026, 7, 24, 19, 0, 0, 0, time.FixedZone("SGT", 8*60*60))
	date := "2026-07-24"
	startTime := "19:00"
	endTime := "21:00"
	input := MessageInput{
		SenderName: "Jasmine",
		Timestamp:  start,
		Timezone:   "Asia/Singapore",
		Source:     "telegram",
	}

	tests := []struct {
		name     string
		hostName string
	}{
		{name: "unchanged", hostName: "Jasmine"},
		{name: "emoji and whitespace", hostName: "🏸  Jasmine"},
		{name: "case variation", hostName: "JASMINE"},
		{name: "invisible formatting", hostName: "\u200bJasmine\u200f"},
		{name: "edge punctuation", hostName: "** Jasmine! **"},
		{name: "compatibility characters", hostName: "Ｊａｓｍｉｎｅ"},
		{name: "empty after normalization falls back to sender", hostName: "🏸"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := model.ParsedPlayCandidate{
				HostName:  &tt.hostName,
				Date:      &date,
				StartTime: &startTime,
				EndTime:   &endTime,
			}

			params := ToUpsertPlayParams(&candidate, input)
			if params.HostName != "Jasmine" {
				t.Errorf("HostName for %q = %q, want %q", tt.hostName, params.HostName, "Jasmine")
			}
		})
	}
}

func TestToUpsertPlayParams_PreservesDifferentOrganizer(t *testing.T) {
	hostName := "🏸 Alice"
	candidate := model.ParsedPlayCandidate{HostName: &hostName}

	params := ToUpsertPlayParams(&candidate, MessageInput{
		SenderName: "Jasmine",
		Timezone:   "Asia/Singapore",
		Source:     "telegram",
	})

	if params.HostName != "Alice" {
		t.Fatalf("HostName = %q, want %q", params.HostName, "Alice")
	}
}

func TestParseResponse_FloatCourts(t *testing.T) {
	t.Run("array with float courts parses", func(t *testing.T) {
		json := `[
			{
				"listing_type": "play",
				"host_name": "Alice",
				"date": "2026-04-11",
				"start_time": "17:00",
				"end_time": "19:00",
				"venue": "OCBC Arena",
				"courts": 3.5,
				"fee_cents": 1200,
				"currency": "SGD",
				"contacts": []
			},
			{
				"listing_type": "play",
				"host_name": "Alice",
				"date": "2026-04-11",
				"start_time": "20:00",
				"end_time": "22:00",
				"venue": "OCBC Arena",
				"courts": 2,
				"fee_cents": 1200,
				"currency": "SGD",
				"contacts": []
			}
		]`

		candidates, err := parseResponse(json, "raw block")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(candidates) != 2 {
			t.Fatalf("got %d candidates, want 2", len(candidates))
		}
		if candidates[0].Courts == nil || *candidates[0].Courts != 3.5 {
			t.Errorf("candidate 0 courts: got %v, want 3.5", candidates[0].Courts)
		}
		if candidates[1].Courts == nil || *candidates[1].Courts != 2.0 {
			t.Errorf("candidate 1 courts: got %v, want 2", candidates[1].Courts)
		}
	})

	t.Run("single object with float courts parses", func(t *testing.T) {
		json := `{
			"listing_type": "play",
			"host_name": "Bob",
			"courts": 1.5,
			"currency": "SGD",
			"contacts": []
		}`

		candidates, err := parseResponse(json, "raw block")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(candidates) != 1 {
			t.Fatalf("got %d candidates, want 1", len(candidates))
		}
		if candidates[0].Courts == nil || *candidates[0].Courts != 1.5 {
			t.Errorf("courts: got %v, want 1.5", candidates[0].Courts)
		}
	})

	// LLMs sometimes quote the number or echo surrounding text despite the
	// schema saying "number" — this killed real messages in prod.
	t.Run("string numbers parse", func(t *testing.T) {
		json := `[
			{"listing_type": "play", "host_name": "Cara", "courts": "2", "fee_cents": "1200", "max_players": "6", "slots_left": "2", "currency": "SGD", "contacts": []},
			{"listing_type": "play", "host_name": "Cara", "courts": "3.5 courts", "currency": "SGD", "contacts": []}
		]`

		candidates, err := parseResponse(json, "raw block")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(candidates) != 2 {
			t.Fatalf("got %d candidates, want 2", len(candidates))
		}
		if candidates[0].Courts == nil || *candidates[0].Courts != 2.0 {
			t.Errorf("candidate 0 courts: got %v, want 2", candidates[0].Courts)
		}
		if candidates[0].FeeCents == nil || *candidates[0].FeeCents != 1200 {
			t.Errorf("candidate 0 fee_cents: got %v, want 1200", candidates[0].FeeCents)
		}
		if candidates[0].MaxPlayers == nil || *candidates[0].MaxPlayers != 6 {
			t.Errorf("candidate 0 max_players: got %v, want 6", candidates[0].MaxPlayers)
		}
		if candidates[0].SlotsLeft == nil || *candidates[0].SlotsLeft != 2 {
			t.Errorf("candidate 0 slots_left: got %v, want 2", candidates[0].SlotsLeft)
		}
		if candidates[1].Courts == nil || *candidates[1].Courts != 3.5 {
			t.Errorf("candidate 1 courts: got %v, want 3.5", candidates[1].Courts)
		}
	})

	// Non-numeric strings must still fail loudly — silently misparsing "$10"
	// would store wrong data.
	t.Run("non-numeric string is rejected", func(t *testing.T) {
		json := `[{"listing_type": "play", "host_name": "Cara", "fee_cents": "$10", "currency": "SGD", "contacts": []}]`

		if _, err := parseResponse(json, "raw block"); err == nil {
			t.Fatal("expected error for non-numeric fee_cents, got nil")
		}
	})
}
