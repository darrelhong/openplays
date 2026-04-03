package dedupe

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Threshold is the minimum Jaccard similarity (0.0-1.0) for two messages
// to be considered duplicates. Set to 0.85 based on testing with real
// Telegram badminton group messages — catches reposts with minor edits
// (slot count changes, session removals) while keeping different-host
// messages with similar structure as separate.
const Threshold = 0.85

// ContentHash returns a SHA256 hex hash of the normalized message text.
// Used for fast exact-match dedup and stored in the raw_messages table.
func ContentHash(text string) string {
	normalized := Normalize(text)
	h := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", h)
}

// IsSimilar checks if two message texts are similar enough to be considered
// duplicates, using trigram Jaccard similarity.
func IsSimilar(a, b string) bool {
	return Similarity(a, b) >= Threshold
}

// Similarity computes the Jaccard similarity between two texts using trigrams.
// Returns a value between 0.0 (completely different) and 1.0 (identical).
//
// Jaccard similarity = |intersection| / |union| of trigram sets.
func Similarity(a, b string) float64 {
	triA := Trigrams(Normalize(a))
	triB := Trigrams(Normalize(b))

	if len(triA) == 0 && len(triB) == 0 {
		return 1.0
	}
	if len(triA) == 0 || len(triB) == 0 {
		return 0.0
	}

	intersection := 0
	for tri := range triA {
		if triB[tri] {
			intersection++
		}
	}

	union := len(triA) + len(triB) - intersection
	if union == 0 {
		return 1.0
	}

	result := float64(intersection) / float64(union)

	return result
}

// emojiRe matches most emoji characters and variation selectors.
var emojiRe = regexp.MustCompile(`[\x{1F000}-\x{1FFFF}]|[\x{2600}-\x{27BF}]|[\x{FE00}-\x{FE0F}]|[\x{200D}]|[\x{20E3}]|[\x{E0020}-\x{E007F}]`)

// Normalize prepares text for comparison by lowercasing, stripping emoji,
// collapsing whitespace, and removing common noise characters.
func Normalize(text string) string {
	// Lowercase
	text = strings.ToLower(text)

	// Strip emoji
	text = emojiRe.ReplaceAllString(text, "")

	// Strip common noise: hashtags, markdown formatting
	text = strings.ReplaceAll(text, "#", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "_", "")

	// Collapse all whitespace (newlines, tabs, multiple spaces) into single space
	text = collapseWhitespace(text)

	return strings.TrimSpace(text)
}

// Trigrams extracts the set of 3-character sliding windows from text.
// Returns a set (map[string]bool) for efficient intersection/union ops.
func Trigrams(text string) map[string]bool {
	runes := []rune(text)
	if len(runes) < 3 {
		// For very short texts, use the whole string as a single "trigram"
		if len(runes) > 0 {
			return map[string]bool{string(runes): true}
		}
		return nil
	}

	trigrams := make(map[string]bool, len(runes)-2)
	for i := 0; i <= len(runes)-3; i++ {
		tri := string(runes[i : i+3])
		trigrams[tri] = true
	}
	return trigrams
}

func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !inSpace {
				b.WriteRune(' ')
				inSpace = true
			}
		} else {
			b.WriteRune(r)
			inSpace = false
		}
	}
	return b.String()
}
