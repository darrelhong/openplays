package dedupe

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase and trim",
			input:    "  Looking for HB Players  ",
			expected: "looking for hb players",
		},
		{
			name:     "strip emoji",
			input:    "🏸 Looking for players 🔥",
			expected: "looking for players",
		},
		{
			name:     "collapse whitespace and newlines",
			input:    "Date: 3 Apr\n\nVenue: SBH Expo\n\nTime: 8pm",
			expected: "date: 3 apr venue: sbh expo time: 8pm",
		},
		{
			name:     "strip hashtags and markdown",
			input:    "#HB #LI *bold* _italic_",
			expected: "hb li bold italic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTrigrams(t *testing.T) {
	tri := Trigrams("abcde")
	expected := map[string]bool{"abc": true, "bcd": true, "cde": true}

	if len(tri) != len(expected) {
		t.Fatalf("got %d trigrams, want %d", len(tri), len(expected))
	}
	for k := range expected {
		if !tri[k] {
			t.Errorf("missing trigram %q", k)
		}
	}
}

func TestTrigrams_Short(t *testing.T) {
	// Text shorter than 3 chars uses whole string as single trigram
	tri := Trigrams("ab")
	if len(tri) != 1 || !tri["ab"] {
		t.Errorf("short text trigrams = %v, want {ab: true}", tri)
	}

	// Empty text returns nil
	tri = Trigrams("")
	if tri != nil {
		t.Errorf("empty text trigrams = %v, want nil", tri)
	}
}

func TestSimilarity_Identical(t *testing.T) {
	sim := Similarity("Looking for HB players at Bedok", "Looking for HB players at Bedok")
	if sim != 1.0 {
		t.Errorf("identical texts similarity = %f, want 1.0", sim)
	}
}

func TestSimilarity_NearDuplicate(t *testing.T) {
	// Same message, minor slot count change
	a := "Looking for HB players. Date: 3 Apr. Venue: SBH Expo. 3 slots left."
	b := "Looking for HB players. Date: 3 Apr. Venue: SBH Expo. 2 slots left."

	sim := Similarity(a, b)
	if sim < 0.90 {
		t.Errorf("near-duplicate similarity = %f, want >= 0.90", sim)
	}
}

func TestSimilarity_DifferentMessages(t *testing.T) {
	a := "Looking for HB players at Heartbeat Bedok, 3 Apr 8pm"
	b := "Selling Astrox 88s Pro 3rd Gen, $190, 4UG5, Strung with Exbolt"

	sim := Similarity(a, b)
	if sim > 0.3 {
		t.Errorf("different messages similarity = %f, want < 0.3", sim)
	}
}

func TestSimilarity_SameHostRepost(t *testing.T) {
	// Host reposts same message with emoji/formatting changes
	a := `🌟Looking for friendly HB-LI players for wkend games in Bishan/Northeast

🗓️：4 Apr, Sat
📍：Kuo Chuan Presbyterian Primary (near Bishan MRT)
⌚️：4pm - 6pm
🏸：HB-LI
3 slot left

💰：$10 per pax
New RSL Supreme Shuttlecocks`

	b := `🌟Looking for friendly HB-LI players for wkend games in Bishan/Northeast

🗓️：4 Apr, Sat
📍：Kuo Chuan Presbyterian Primary (near Bishan MRT)
⌚️：4pm - 6pm
🏸：HB-LI
2 slot left

💰：$10 per pax
New RSL Supreme Shuttlecocks`

	sim := Similarity(a, b)
	if sim < Threshold {
		t.Errorf("repost similarity = %f, want >= %f", sim, Threshold)
	}
}

func TestIsSimilar(t *testing.T) {
	a := "Looking for HB players. Date: 3 Apr. Venue: SBH Expo."
	b := "Looking for HB players. Date: 3 Apr. Venue: SBH Expo."
	if !IsSimilar(a, b) {
		t.Error("identical messages should be similar")
	}

	c := "Selling racket, $190, brand new condition"
	if IsSimilar(a, c) {
		t.Error("completely different messages should not be similar")
	}
}

func TestContentHash_Deterministic(t *testing.T) {
	h1 := ContentHash("Looking for HB players 🏸")
	h2 := ContentHash("Looking for HB players 🏸")
	if h1 != h2 {
		t.Error("same input should produce same hash")
	}

	// Emoji/whitespace differences should still produce same hash after normalization
	h3 := ContentHash("Looking for HB players")
	if h1 != h3 {
		t.Error("normalized equivalents should produce same hash")
	}
}

func TestContentHash_Different(t *testing.T) {
	h1 := ContentHash("Looking for HB players at Bedok")
	h2 := ContentHash("Selling racket $190")
	if h1 == h2 {
		t.Error("different messages should produce different hashes")
	}
}
