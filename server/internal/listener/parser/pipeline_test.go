package parser

import (
	"testing"
)

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
}
