package usernames

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
)

const (
	MaxLength    = 32
	SuffixLength = 4
)

var ErrInvalid = errors.New("invalid username")

const suffixAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func BaseFromDisplayName(displayName string) string {
	base := slug(displayName)
	if base == "" {
		base = "player"
	}
	return trimToMax(base, MaxLength)
}

func Normalize(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return "", fmt.Errorf("%w: cannot be empty", ErrInvalid)
	}
	if len(normalized) > MaxLength {
		return "", fmt.Errorf("%w: must be at most %d characters", ErrInvalid, MaxLength)
	}
	for _, r := range normalized {
		if !isUsernameChar(r) {
			return "", fmt.Errorf("%w: use only lowercase letters, numbers, and underscores", ErrInvalid)
		}
	}
	return normalized, nil
}

func WithRandomSuffix(base string) (string, error) {
	suffix, err := RandomSuffix(SuffixLength)
	if err != nil {
		return "", err
	}
	prefix := trimToMax(base, MaxLength-1-SuffixLength)
	if prefix == "" {
		prefix = "player"
	}
	return prefix + "_" + suffix, nil
}

func RandomSuffix(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate username suffix: %w", err)
	}
	out := make([]byte, length)
	for i, b := range buf {
		out[i] = suffixAlphabet[int(b)%len(suffixAlphabet)]
	}
	return string(out), nil
}

func slug(value string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if b.Len() > 0 && !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

func trimToMax(value string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}
	if len(value) <= maxLength {
		return strings.Trim(value, "_")
	}
	return strings.Trim(value[:maxLength], "_")
}

func isUsernameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_'
}
