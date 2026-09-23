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

		// Too vague -- should not match (initialism expansion is deliberately not
		// attempted here; it belongs in the alias table, not in fuzzy scoring)
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

// TestTokensMatch pins down the token comparison. The previous implementation
// used strings.Contains in both directions, which made "sengkang" match "kang"
// and scored "SENGKANG S/HALL" at 1.00 against Yio Chu Kang Sport Hall.
func TestTokensMatch(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		// exact
		{"sengkang", "sengkang", true},
		{"hougang", "hougang", true},

		// legitimate prefixes over enough of the word
		{"canberra", "canberra", true},
		{"badminton", "badminton", true},
		{"yio", "yio", true},

		// the regression: "kang" is a substring of "sengkang" but not a prefix
		{"sengkang", "kang", false},
		{"seng", "sengkang", false}, // prefix, but 4/8 = 0.5 < 0.6 ratio
		{"sengk", "sengkang", true}, // prefix, 5/8 = 0.625 >= 0.6 ratio

		// "jurong" must not absorb "jurongville" (different venue)
		{"jurong", "jurongville", false},

		// short tokens never prefix-match
		{"cc", "community", false},
		{"sh", "sport", false},

		// unrelated
		{"sengkang", "clementi", false},
	}
	for _, tt := range tests {
		if got := tokensMatch(tt.a, tt.b); got != tt.want {
			t.Errorf("tokensMatch(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

// TestFuzzyMatch_Ambiguity guards the fix that stopped "SENGKANG S/HALL"
// resolving to Yio Chu Kang: ties must be reported, not silently won by
// whichever candidate happens to come first.
func TestFuzzyMatch_Ambiguity(t *testing.T) {
	candidates := []Candidate{
		{ID: 30, Name: "Sengkang Sport Hall"},
		{ID: 46, Name: "Yio Chu Kang Sport Hall"},
		{ID: 25, Name: "Choa Chu Kang Sport Hall"},
	}

	m, ties := fuzzyMatchDetailed("SENGKANG S/HALL", candidates)
	if m == nil {
		t.Fatal("expected a match for SENGKANG S/HALL")
	}
	if m.ID != 30 {
		t.Errorf("matched %d (%s), want 30 (Sengkang Sport Hall)", m.ID, m.Name)
	}
	if ties != 1 {
		t.Errorf("ties = %d, want 1 (a unique winner)", ties)
	}

	// Candidate order must not change the answer.
	reversed := []Candidate{candidates[2], candidates[1], candidates[0]}
	m2, _ := fuzzyMatchDetailed("SENGKANG S/HALL", reversed)
	if m2 == nil || m2.ID != 30 {
		t.Errorf("reversed order matched %+v, want id 30", m2)
	}
}

// TestFuzzyMatch_GenericOnlyInputMakesNoClaim covers inputs that carry no
// identifying information: they must not resolve to a venue just because the
// generic words line up.
func TestFuzzyMatch_GenericOnlyInputMakesNoClaim(t *testing.T) {
	candidates := []Candidate{
		{ID: 1, Name: "Hougang Sport Hall"},
		{ID: 2, Name: "Yishun Sport Hall"},
	}
	// "s/hall" expands to "sport hall"; with no real name there is nothing to
	// distinguish venue 1 from venue 2.
	m, ties := fuzzyMatchDetailed("S/Hall", candidates)
	if m == nil {
		t.Fatal("expected a match (threshold is met) so the caller can see the tie")
	}
	if ties < 2 {
		t.Errorf("ties = %d, want >= 2 so the caller treats it as ambiguous", ties)
	}
}
