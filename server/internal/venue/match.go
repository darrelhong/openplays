// Package venue handles venue name normalisation, abbreviation expansion,
// and fuzzy matching against known venues.
//
// # Resolution flow
//
//  1. Exact alias lookup — lowercased raw venue string is looked up in
//     venue_aliases. If found, the cached venue is returned immediately.
//
//  2. Expanded alias lookup — abbreviations in the raw string are expanded
//     (e.g. "hougang sec" → "hougang secondary school") and the expanded
//     form is looked up in venue_aliases. On hit, the original raw string
//     is also saved as an alias so future lookups skip expansion.
//
//  3. Fuzzy word overlap — the expanded string is compared against all
//     venue names in the database using word-level overlap scoring.
//     Each input word is checked against candidate venue words; a word
//     matches if it equals or is a substring of a candidate word.
//     The score is (matched input words / total input words). The best
//     match above 60% threshold is used. On hit, the original raw string
//     is saved as an alias.
//
//  4. Geocoder fallback — if no fuzzy match, the raw string is sent to
//     the configured geocoding provider (Google Places or OneMap). On hit,
//     the venue is upserted and the raw string is saved as an alias.
//
//  5. Unresolved — if all steps fail, venue_postal_code is left NULL on
//     the play for manual resolution later.
//
// # Normalisation
//
// Both input strings and venue names are normalised before comparison:
//   - Lowercased
//   - "sports hall" → "sport hall" (canonical form)
//
// # Abbreviation expansion
//
// Common Singapore venue abbreviations are expanded before matching:
//
//	sec → secondary        cc  → community club
//	sch → school           sh  → sport hall
//	pri → primary          jc  → junior college
//
// Expansions are single-word replacements so they compose correctly:
// "sec sch" → "secondary school", "pri sch" → "primary school".
//
// # What this doesn't handle
//
// Initialisms (SBH, TPCC, BV CC) and nicknames cannot be resolved by
// expansion or fuzzy matching — these need manual aliases added via
// the venuefill tool.
package venue

import (
	"strings"
)

// minMatchRatio is the minimum fraction of input words that must appear
// in a candidate venue name for a fuzzy match.
const minMatchRatio = 0.60

// expansions maps common single-word abbreviations to their full forms.
// Each abbreviation expands to exactly one word so they compose:
// "sec sch" → "secondary school", not "secondary school school".
var expansions = map[string]string{
	"sec": "secondary",
	"sch": "school",
	"cc":  "community club",
	"sh":  "sport hall",
	"pri": "primary",
	"jc":  "junior college",
	"sp":  "sport",
	"ctr": "centre",
	"csc": "community sport centre",
	"ssc": "sport centre",
	"pa":  "people's association",
}

// Candidate is a venue to match against.
type Candidate struct {
	PostalCode string
	Name       string
}

// Match holds the result of a fuzzy match.
type Match struct {
	PostalCode string
	Name       string
	Score      float64
}

// normalise lowercases and replaces "sports hall" with "sport hall".
func normalise(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "sports hall", "sport hall")
	return s
}

// expand replaces known abbreviations in a string with their full forms.
func expand(s string) string {
	words := strings.Fields(s)
	var out []string
	for _, w := range words {
		if full, ok := expansions[w]; ok {
			out = append(out, full)
		} else {
			out = append(out, w)
		}
	}
	return strings.Join(out, " ")
}

// ExpandAndNormalise applies abbreviation expansion then normalisation.
func ExpandAndNormalise(s string) string {
	return normalise(expand(normalise(s)))
}

// wordSet returns the unique words in a string.
func wordSet(s string) map[string]bool {
	set := make(map[string]bool)
	for _, w := range strings.Fields(s) {
		set[w] = true
	}
	return set
}

// wordOverlap scores how many input words appear in the candidate.
// A word matches if it equals or is a substring of any candidate word.
// Returns (matches / total input words).
func wordOverlap(inputWords map[string]bool, candidateWords map[string]bool) float64 {
	if len(inputWords) == 0 {
		return 0
	}
	matches := 0
	for iw := range inputWords {
		for cw := range candidateWords {
			if iw == cw || strings.Contains(cw, iw) || strings.Contains(iw, cw) {
				matches++
				break
			}
		}
	}
	return float64(matches) / float64(len(inputWords))
}

// FuzzyMatch finds the best matching venue from candidates for a raw venue string.
// Returns nil if no match meets the threshold.
func FuzzyMatch(raw string, candidates []Candidate) *Match {
	expanded := ExpandAndNormalise(raw)
	inputWords := wordSet(expanded)

	if len(inputWords) == 0 {
		return nil
	}

	var best *Match
	for _, c := range candidates {
		normName := normalise(c.Name)
		candidateWords := wordSet(normName)
		score := wordOverlap(inputWords, candidateWords)

		if score >= minMatchRatio && (best == nil || score > best.Score) {
			best = &Match{
				PostalCode: c.PostalCode,
				Name:       c.Name,
				Score:      score,
			}
		}
	}

	return best
}
