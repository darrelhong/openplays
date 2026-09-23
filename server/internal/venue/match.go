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
//     Each input word is checked against candidate words; a word matches
//     if equal, or if one is a prefix of the other covering at least 60%
//     of the longer word. The score is (matched input words / total input
//     words); the best match above the 60% threshold is used.
//
//     Ties are broken by Jaccard similarity, and a match is only cached as
//     an alias when it wins outright. If the input contains a real name
//     word, candidates matching only generic words are rejected. See
//     fuzzyMatchDetailed for why each of these guards exists.
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
	"regexp"
	"sort"
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
	ID   int64
	Name string
}

// Match holds the result of a fuzzy match.
type Match struct {
	ID    int64
	Name  string
	Score float64
}

// nonAlnum collapses punctuation to spaces. Venue strings are written by hand
// in Telegram messages, so separators vary: "Sengkang S/Hall", "Blk 123, #01",
// "@Yishun". Normalising punctuation early means "s/hall" and "s hall" tokenise
// the same way instead of becoming one unmatchable token.
var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// normalise lowercases, canonicalises the "sports hall" family, and strips
// punctuation to single spaces.
func normalise(s string) string {
	s = strings.ToLower(s)
	// "S/Hall" is extremely common in these messages. Expand it here, before
	// punctuation stripping, so it becomes the two words "sport hall" and can
	// match venues named "... Sport Hall".
	s = strings.ReplaceAll(s, "s/hall", " sport hall ")
	s = strings.ReplaceAll(s, "sports hall", "sport hall")
	s = strings.ReplaceAll(s, "sporthall", "sport hall")
	s = nonAlnum.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

// expand replaces known abbreviations in a string with their full forms.
// Multi-word expansions ("cc" → "community club") are split into separate
// words so they tokenise correctly.
func expand(s string) string {
	words := strings.Fields(s)
	var out []string
	for _, w := range words {
		if full, ok := expansions[w]; ok {
			out = append(out, strings.Fields(full)...)
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

// genericWords appear in many venue names, so on their own they identify
// nothing. "Hougang S/Hall" and "Kim Seng Sports Hall" both reduce to generic
// words plus one real name; matching only on the generic words is meaningless.
var genericWords = map[string]bool{
	"sport": true, "hall": true, "centre": true, "center": true,
	"community": true, "club": true, "school": true, "secondary": true,
	"primary": true, "badminton": true, "stadium": true, "court": true,
	"courts": true, "hub": true, "the": true, "at": true, "and": true,
}

const (
	// minPrefixLen is the shortest token that may participate in a prefix match.
	minPrefixLen = 3
	// prefixRatio is how much of the longer word a prefix must cover. Without
	// it, "jurong" matches "jurongville", which is a different school.
	prefixRatio = 0.6
)

// tokensMatch reports whether two tokens refer to the same thing: either an
// exact match, or a prefix covering enough of the longer token.
//
// This is deliberately not a substring test. A substring test makes
// "sengkang" match "kang", so "SENGKANG S/HALL" scored 1.00 against Yio Chu
// Kang Sport Hall — the bug that mis-filed 150+ plays.
func tokensMatch(a, b string) bool {
	if a == b {
		return true
	}
	short, long := a, b
	if len(short) > len(long) {
		short, long = long, short
	}
	if len(short) < minPrefixLen || !strings.HasPrefix(long, short) {
		return false
	}
	return float64(len(short))/float64(len(long)) >= prefixRatio
}

func isGeneric(w string) bool { return genericWords[w] }

// adjacentJoins returns each pair of adjacent words joined, so a candidate
// written "Seng Kang Primary School Hall" also produces the token "sengkang".
// Singapore venue names are inconsistently spaced in these messages
// ("Sengkang" vs "Seng Kang"); without this the two spellings look unrelated
// and the wrong venue wins.
func adjacentJoins(words []string) map[string]bool {
	joins := make(map[string]bool)
	for i := 0; i+1 < len(words); i++ {
		joins[words[i]+words[i+1]] = true
	}
	return joins
}


func hasDistinctive(words map[string]bool) bool {
	for w := range words {
		if !isGeneric(w) {
			return true
		}
	}
	return false
}

// scoredCandidate is a candidate venue with its two scores.
//
// ratio is the fraction of input tokens matched (recall, which drives the
// threshold). jaccard is intersection-over-union, used to break ties between
// venues that all match the input equally well: it prefers the venue whose
// name is closest in size, so "canberra sport hall" beats "bukit canberra
// sport hall" only when it genuinely covers more of the name.
//
type scoredCandidate struct {
	cand  Candidate
	ratio float64
	jac   float64
}

// fuzzyMatchDetailed finds the best matching venue and reports how many
// candidates tied on the winning score.
//
// A non-zero tie count means the result is arbitrary — several venues fit the
// input equally well — and the caller must not treat the winner as fact.
// Returning that count instead of silently picking the first is the fix for
// the mis-filing: the old code compared scores with `>` only, so the winner
// was decided by database row order.
func fuzzyMatchDetailed(raw string, candidates []Candidate) (*Match, int) {
	inputExpanded := ExpandAndNormalise(raw)
	inputTokens := strings.Fields(inputExpanded)
	if len(inputTokens) == 0 {
		return nil, 0
	}
	inputWords := wordSet(inputExpanded)
	inputDistinctive := hasDistinctive(inputWords)

	var hits []scoredCandidate
	for _, c := range candidates {
		candExpanded := ExpandAndNormalise(c.Name)
		candWords := wordSet(candExpanded)
		// Candidate tokens include joined adjacent words, so a venue spelled
		// "Seng Kang" offers the token "sengkang" to match against.
		candTokens := map[string]bool{}
		for w := range candWords {
			candTokens[w] = true
		}
		for j := range adjacentJoins(strings.Fields(candExpanded)) {
			candTokens[j] = true
		}

		matchedWords := make([]bool, len(inputTokens))
		distinctive := false

		// Direct token matches.
		for i, iw := range inputTokens {
			for cw := range candTokens {
				if tokensMatch(iw, cw) {
					matchedWords[i] = true
					if !isGeneric(iw) || !isGeneric(cw) {
						distinctive = true
					}
					break
				}
			}
		}

		// Joined matches: a spelling split across words ("feng shan") is one
		// name, so if the join matches a candidate token, all the words it
		// spans count as matched. This is what lets "feng shan primary" reach
		// the venue named "Fengshan Primary School Hall".
		for i := 0; i+1 < len(inputTokens); i++ {
			join := inputTokens[i] + inputTokens[i+1]
			if isGeneric(join) {
				continue
			}
			for cw := range candTokens {
				if tokensMatch(join, cw) {
					matchedWords[i] = true
					matchedWords[i+1] = true
					distinctive = true
					break
				}
			}
		}

		matched := 0
		for _, m := range matchedWords {
			if m {
				matched++
			}
		}

		// If the input contains a real name ("sengkang"), a candidate that only
		// matched generic words ("sport hall") is not a credible match.
		if inputDistinctive && !distinctive {
			continue
		}
		ratio := float64(matched) / float64(len(inputTokens))
		if ratio < minMatchRatio {
			continue
		}
		union := len(inputWords) + len(candWords) - matched
		jac := 0.0
		if union > 0 {
			jac = float64(matched) / float64(union)
		}
		hits = append(hits, scoredCandidate{cand: c, ratio: ratio, jac: jac})
	}
	if len(hits) == 0 {
		return nil, 0
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].ratio != hits[j].ratio {
			return hits[i].ratio > hits[j].ratio
		}
		return hits[i].jac > hits[j].jac
	})
	best := hits[0]
	ties := 0
	for _, h := range hits {
		if h.ratio == best.ratio && h.jac == best.jac {
			ties++
		}
	}
	return &Match{ID: best.cand.ID, Name: best.cand.Name, Score: best.ratio}, ties
}

// FuzzyMatch finds the best matching venue from candidates for a raw venue
// string, returning nil if nothing clears the threshold.
func FuzzyMatch(raw string, candidates []Candidate) *Match {
	m, _ := fuzzyMatchDetailed(raw, candidates)
	return m
}
