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

// Courts is optional: a listing whose court count the model could not read must
// still be inserted, with courts NULL rather than 0.
//
// 0 and null mean different things here — 0 would render as "0 courts" in the
// UI, which is wrong and worse than saying nothing. This is a real case: a live
// eval run produced courts=null for one tier while its source line did state a
// court count for a sibling tier, and the model dropped it.
func TestParseResponse_NullCourtsStaysNull(t *testing.T) {
	t.Run("explicit null persists as nil", func(t *testing.T) {
		json := `[{
			"listing_type": "play",
			"host_name": "Alice",
			"date": "2026-04-03",
			"start_time": "13:00",
			"end_time": "15:00",
			"venue": "Sengkang Sports Hall",
			"courts": null,
			"fee_cents": 1300,
			"currency": "SGD",
			"contacts": []
		}]`

		candidates, err := parseResponse(json, "raw block")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if candidates[0].Courts != nil {
			t.Errorf("candidate courts = %v, want nil", *candidates[0].Courts)
		}

		params := ToUpsertPlayParams(&candidates[0], MessageInput{
			SenderName: "Alice",
			Timezone:   "Asia/Singapore",
		})
		if params.Courts != nil {
			t.Errorf("params.Courts = %v, want nil (NULL in DB, not 0)", *params.Courts)
		}
	})

	t.Run("absent courts field persists as nil", func(t *testing.T) {
		json := `[{
			"listing_type": "play",
			"host_name": "Alice",
			"date": "2026-04-03",
			"start_time": "13:00",
			"end_time": "15:00",
			"venue": "Sengkang Sports Hall",
			"fee_cents": 1300,
			"currency": "SGD",
			"contacts": []
		}]`

		candidates, err := parseResponse(json, "raw block")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if candidates[0].Courts != nil {
			t.Errorf("candidate courts = %v, want nil", *candidates[0].Courts)
		}
		params := ToUpsertPlayParams(&candidates[0], MessageInput{SenderName: "Alice"})
		if params.Courts != nil {
			t.Errorf("params.Courts = %v, want nil", *params.Courts)
		}
	})

	t.Run("zero is preserved as zero, not treated as absent", func(t *testing.T) {
		json := `[{"venue":"A","courts":0}]`
		candidates, err := parseResponse(json, "raw block")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if candidates[0].Courts == nil || *candidates[0].Courts != 0 {
			t.Errorf("courts = %v, want 0", candidates[0].Courts)
		}
	})
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

// TestParseResponse_TrailingJunk covers the model emitting a complete array
// followed by a stray character. json.Unmarshal rejects the whole response
// ("invalid character '}' after top-level value"), which discarded every
// listing and burned a retry; these responses contain valid data and are kept.
func TestParseResponse_TrailingJunk(t *testing.T) {
	valid := `[{"listing_type":"play","host_name":"Apple21","date":"2026-09-24","venue":"Cheng San CC","fee_cents":1000}]`

	for _, tc := range []struct {
		name    string
		content string
		wantN   int
	}{
		{"clean", valid, 1},
		{"stray closing brace", valid + "}", 1},
		{"stray brace with space", valid + " }", 1},
		{"stray brace on new line", valid + "\n}", 1},
		{"trailing prose", valid + "\nHope that helps!", 1},
		{"trailing semicolon", valid + ";", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseResponse(tc.content, "raw block")
			if err != nil {
				t.Fatalf("parseResponse returned error: %v", err)
			}
			if len(got) != tc.wantN {
				t.Fatalf("got %d listings, want %d", len(got), tc.wantN)
			}
			if got[0].HostName == nil || *got[0].HostName != "Apple21" {
				t.Errorf("host_name = %v, want Apple21", got[0].HostName)
			}
			if got[0].RawBlock != "raw block" {
				t.Errorf("RawBlock = %q, want %q", got[0].RawBlock, "raw block")
			}
		})
	}
}

// TestParseResponse_TruncationStillRejected is the guard on the above: the
// loosening must not turn truncated output into a partial success. A half
// written array has no complete leading value, so it must still error.
func TestParseResponse_TruncationStillRejected(t *testing.T) {
	for _, content := range []string{
		`[{"listing_type":"play","host_name":"Apple21","cur`,
		`[{"listing_type":"play"},`,
		`[{"listing_type":"play"}`,
		``,
		`not json at all`,
	} {
		if _, err := parseResponse(content, "raw block"); err == nil {
			t.Errorf("parseResponse(%.40q) succeeded, want error", content)
		}
	}
}

// TestParseResponse_RejectsNonArrayNonObject keeps the top-level type guard
// intact.
//
// `null` is deliberately excluded: json.Unmarshal into a slice accepts null
// and yields an empty slice, so it parsed to zero listings before this change
// and still does. That is pre-existing behaviour, not something the trailing
// junk fix altered, and it is recorded here so the next reader does not
// mistake it for a regression.
func TestParseResponse_RejectsNonArrayNonObject(t *testing.T) {
	for _, content := range []string{`"a string"`, `42`, `true`} {
		if _, err := parseResponse(content, "raw block"); err == nil {
			t.Errorf("parseResponse(%s) succeeded, want error", content)
		}
	}

	// Documented behaviour: null is an empty result, not an error.
	got, err := parseResponse(`null`, "raw block")
	if err != nil {
		t.Errorf("parseResponse(null) error = %v, want nil (pre-existing behaviour)", err)
	}
	if len(got) != 0 {
		t.Errorf("parseResponse(null) returned %d listings, want 0", len(got))
	}
}

// TestParseResponse_RejectsNonObjectElements covers elements inside an array.
//
// Each element is decoded into model.ParsedPlayCandidate, so an element that
// is not an object (or null) is a hard error. This matters because the eval
// transform must fail on exactly the same inputs: if it accepted `[[]]` and
// scored a listing, a run would look green while the listener rejected the
// same response and burned a retry. The transform previously did accept these,
// which is how the divergence was found.
//
// Note these all stay errors after the trailing-junk fix: reading the first
// JSON value does not make malformed elements valid.
func TestParseResponse_RejectsNonObjectElements(t *testing.T) {
	for _, content := range []string{
		`[[]]`,                // element is an array
		`["a"]`,               // element is a string
		`[123]`,               // element is a number
		`[true]`,              // element is a bool
		`[{"venue":"X"},[]]`,  // valid element followed by an array
		`[{"venue":"X"},123]`, // valid element followed by a number
	} {
		if _, err := parseResponse(content, "raw block"); err == nil {
			t.Errorf("parseResponse(%s) succeeded, want error", content)
		}
	}

	// null elements are NOT rejected: Go decodes null into a zero-value struct.
	// Verified explicitly so the rule above is not read too broadly.
	got, err := parseResponse(`[null]`, "raw block")
	if err != nil {
		t.Errorf("parseResponse([null]) error = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Errorf("parseResponse([null]) returned %d listings, want 1", len(got))
	}

	// A valid element followed by null keeps both, matching Go's decode.
	got, err = parseResponse(`[{"venue":"X"},null]`, "raw block")
	if err != nil {
		t.Fatalf("parseResponse([obj,null]) error = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Errorf("parseResponse([obj,null]) returned %d listings, want 2", len(got))
	}
}
