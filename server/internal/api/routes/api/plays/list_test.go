package plays

import (
	"testing"
)

func TestEncodeCursor(t *testing.T) {
	tests := []struct {
		name     string
		startsAt string
		id       int64
		want     string
	}{
		{
			name:     "standard RFC3339 stays RFC3339",
			startsAt: "2026-04-10T12:00:00Z",
			id:       42,
			want:     "2026-04-10T12:00:00Z,42",
		},
		{
			name:     "with timezone offset",
			startsAt: "2026-04-10T20:00:00+08:00",
			id:       99,
			want:     "2026-04-10T12:00:00Z,99", // normalized to UTC RFC3339
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeCursor(tt.startsAt, tt.id)
			if got != tt.want {
				t.Errorf("encodeCursor(%q, %d) = %q, want %q", tt.startsAt, tt.id, got, tt.want)
			}
		})
	}
}

func TestDecodeCursor(t *testing.T) {
	tests := []struct {
		name       string
		cursor     string
		wantTime   string
		wantID     int64
		wantOK     bool
	}{
		{
			name:     "valid cursor",
			cursor:   "2026-04-10T12:00:00Z,42",
			wantTime: "2026-04-10T12:00:00Z",
			wantID:   42,
			wantOK:   true,
		},
		{
			name:   "empty cursor",
			cursor: "",
			wantOK: false,
		},
		{
			name:   "no comma",
			cursor: "invalid",
			wantOK: false,
		},
		{
			name:   "bad id",
			cursor: "2026-04-10T12:00:00Z,notanumber",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, gotID, gotOK := decodeCursor(tt.cursor)
			if gotOK != tt.wantOK {
				t.Fatalf("decodeCursor(%q) ok = %v, want %v", tt.cursor, gotOK, tt.wantOK)
			}
			if !gotOK {
				return
			}
			if gotTime != tt.wantTime {
				t.Errorf("decodeCursor(%q) time = %q, want %q", tt.cursor, gotTime, tt.wantTime)
			}
			if gotID != tt.wantID {
				t.Errorf("decodeCursor(%q) id = %d, want %d", tt.cursor, gotID, tt.wantID)
			}
		})
	}
}

func TestCursorRoundTrip(t *testing.T) {
	// Encode from API format (RFC3339), decode, and verify the cursor stays
	// in RFC3339 externally.
	cursor := encodeCursor("2026-04-10T12:00:00Z", 123)

	startsAt, id, ok := decodeCursor(cursor)
	if !ok {
		t.Fatalf("decodeCursor(%q) failed", cursor)
	}
	if id != 123 {
		t.Errorf("round-trip id = %d, want 123", id)
	}
	if startsAt != "2026-04-10T12:00:00Z" {
		t.Errorf("round-trip time = %q, want RFC3339", startsAt)
	}
}

func TestCursorStartsAtForDB(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		wantOK   bool
	}{
		{
			name:   "convert RFC3339 UTC to sqlite format",
			input:  "2026-04-10T12:00:00Z",
			want:   "2026-04-10 12:00:00+00:00",
			wantOK: true,
		},
		{
			name:   "convert offset time to sqlite format",
			input:  "2026-04-10T20:00:00+08:00",
			want:   "2026-04-10 12:00:00+00:00",
			wantOK: true,
		},
		{
			name:   "invalid time",
			input:  "not-a-time",
			want:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := cursorStartsAtForDB(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("cursorStartsAtForDB(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("cursorStartsAtForDB(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
