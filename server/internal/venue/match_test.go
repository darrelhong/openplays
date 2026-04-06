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
	{PostalCode: "538840", Name: "Hougang Community Club"},
	{PostalCode: "768928", Name: "Ahmad Ibrahim Secondary School"},
	{PostalCode: "388352", Name: "Singapore Badminton Hall"},
	{PostalCode: "397631", Name: "OCBC Arena"},
	{PostalCode: "757716", Name: "Bukit Canberra Sports Hall"},
	{PostalCode: "270036", Name: "Buona Vista Community Club"},
	{PostalCode: "169640", Name: "Kim Seng Community Centre"},
	{PostalCode: "768370", Name: "Yishun Sport Hall"},
	{PostalCode: "530540", Name: "Hougang Secondary School"},
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		input      string
		wantPostal string // empty means no match expected
	}{
		// Exact-ish matches
		{"Hougang Community Club", "538840"},
		{"hougang community club", "538840"},

		// Abbreviation expansion
		{"Hougang Sec", "530540"},
		{"Hougang Sec Sch", "530540"},
		{"Hougang CC", "538840"},
		{"Kim Seng CC", "169640"},

		// Missing first word
		{"Canberra Sports Hall", "757716"},
		{"Canberra Sport Hall", "757716"},

		// "sports hall" normalised to "sport hall"
		{"Yishun Sports Hall", "768370"},

		// Too vague -- should not match
		{"SBH", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			match := FuzzyMatch(tt.input, testVenues)
			if tt.wantPostal == "" {
				if match != nil {
					t.Errorf("FuzzyMatch(%q) = %+v, want nil", tt.input, match)
				}
				return
			}
			if match == nil {
				t.Fatalf("FuzzyMatch(%q) = nil, want postal %s", tt.input, tt.wantPostal)
			}
			if match.PostalCode != tt.wantPostal {
				t.Errorf("FuzzyMatch(%q) postal = %s, want %s (matched %q with score %.2f)",
					tt.input, match.PostalCode, tt.wantPostal, match.Name, match.Score)
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
