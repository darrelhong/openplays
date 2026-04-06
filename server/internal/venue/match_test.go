package venue

import (
	"testing"
)

func TestExpandAndNormalise(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hougang Sec", "hougang secondary"},
		{"Hougang Sec Sch", "hougang secondary school"},
		{"Hougang CC", "hougang community club"},
		{"YCK SH", "yck sport hall"},
		{"Sports Hall", "sport hall"},
		{"Canberra Sports Hall", "canberra sport hall"},
		{"normal text", "normal text"},
		{"Pri Sch", "primary school"},
	}
	for _, tt := range tests {
		got := ExpandAndNormalise(tt.input)
		if got != tt.want {
			t.Errorf("ExpandAndNormalise(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

var testVenues = []Candidate{
	{ID: 1, Name: "Hougang Community Club"},
	{ID: 2, Name: "Ahmad Ibrahim Secondary School"},
	{ID: 3, Name: "Singapore Badminton Hall"},
	{ID: 4, Name: "OCBC Arena"},
	{ID: 5, Name: "Bukit Canberra Sports Hall"},
	{ID: 6, Name: "Buona Vista Community Club"},
	{ID: 7, Name: "Kim Seng Community Centre"},
	{ID: 8, Name: "Yishun Sport Hall"},
	{ID: 9, Name: "Hougang Secondary School"},
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		input  string
		wantID int64 // 0 means no match expected
	}{
		// Exact-ish matches
		{"Hougang Community Club", 1},
		{"hougang community club", 1},

		// Abbreviation expansion
		{"Hougang Sec", 9},
		{"Hougang Sec Sch", 9},
		{"Hougang CC", 1},
		{"Kim Seng CC", 7},

		// Missing first word
		{"Canberra Sports Hall", 5},
		{"Canberra Sport Hall", 5},

		// "sports hall" normalised to "sport hall"
		{"Yishun Sports Hall", 8},

		// Too vague -- should not match
		{"SBH", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			match := FuzzyMatch(tt.input, testVenues)
			if tt.wantID == 0 {
				if match != nil {
					t.Errorf("FuzzyMatch(%q) = %+v, want nil", tt.input, match)
				}
				return
			}
			if match == nil {
				t.Fatalf("FuzzyMatch(%q) = nil, want id %d", tt.input, tt.wantID)
			}
			if match.ID != tt.wantID {
				t.Errorf("FuzzyMatch(%q) id = %d, want %d (matched %q with score %.2f)",
					tt.input, match.ID, tt.wantID, match.Name, match.Score)
			}
		})
	}
}

func TestWordOverlap(t *testing.T) {
	tests := []struct {
		input     string
		candidate string
		want      float64
	}{
		{"canberra sport hall", "bukit canberra sport hall", 1.0},     // 3/3
		{"hougang secondary school", "hougang secondary school", 1.0}, // 3/3
		{"canberra hall", "bukit canberra sport hall", 1.0},           // 2/2
		{"foo bar baz", "bukit canberra sport hall", 0.0},             // 0/3
	}

	for _, tt := range tests {
		iw := wordSet(tt.input)
		cw := wordSet(tt.candidate)
		got := wordOverlap(iw, cw)
		if got != tt.want {
			t.Errorf("wordOverlap(%q, %q) = %.2f, want %.2f", tt.input, tt.candidate, got, tt.want)
		}
	}
}
