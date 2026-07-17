package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const MaxBioLength = 500

var (
	genericProfileHandlePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	telegramHandlePattern       = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	instagramHandlePattern      = regexp.MustCompile(`^[A-Za-z0-9._]+$`)
	xHandlePattern              = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	stravaAthleteIDPattern      = regexp.MustCompile(`^[0-9]+$`)
)

// ProfileLinks stores identifiers used to build links to a user's public
// profiles. Values are handles rather than URLs so link construction remains
// under application control.
type ProfileLinks struct {
	Rovo            *string `json:"rovo,omitempty"`
	Reclub          *string `json:"reclub,omitempty"`
	Telegram        *string `json:"telegram,omitempty"`
	Instagram       *string `json:"instagram,omitempty"`
	Facebook        *string `json:"facebook,omitempty"`
	X               *string `json:"x,omitempty"`
	StravaAthleteID *string `json:"strava_athlete_id,omitempty"`
}

// ParseProfileLinks decodes validated profile identifiers from storage.
func ParseProfileLinks(raw *string) (*ProfileLinks, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}

	var links ProfileLinks
	if err := json.Unmarshal([]byte(*raw), &links); err != nil {
		return nil, fmt.Errorf("parse profile_links: %w", err)
	}
	if err := links.NormalizeAndValidate(); err != nil {
		return nil, err
	}
	if links.IsEmpty() {
		return nil, nil
	}
	return &links, nil
}

// ProfileLinksString validates and encodes profile identifiers for storage.
func ProfileLinksString(links *ProfileLinks) (*string, error) {
	if links == nil {
		return nil, nil
	}

	normalized := *links
	if err := normalized.NormalizeAndValidate(); err != nil {
		return nil, err
	}
	if normalized.IsEmpty() {
		return nil, nil
	}

	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal profile_links: %w", err)
	}
	value := string(data)
	return &value, nil
}

// NormalizeAndValidate trims handles, removes an optional leading @, and
// rejects characters that cannot safely form the expected profile URL path.
func (p *ProfileLinks) NormalizeAndValidate() error {
	checks := []struct {
		name    string
		value   **string
		pattern *regexp.Regexp
		max     int
	}{
		{"rovo", &p.Rovo, genericProfileHandlePattern, 64},
		{"reclub", &p.Reclub, genericProfileHandlePattern, 64},
		{"telegram", &p.Telegram, telegramHandlePattern, 32},
		{"instagram", &p.Instagram, instagramHandlePattern, 30},
		{"facebook", &p.Facebook, genericProfileHandlePattern, 64},
		{"x", &p.X, xHandlePattern, 15},
		{"strava_athlete_id", &p.StravaAthleteID, stravaAthleteIDPattern, 20},
	}

	for _, check := range checks {
		if err := normalizeProfileIdentifier(check.name, check.value, check.pattern, check.max); err != nil {
			return err
		}
	}
	return nil
}

func (p ProfileLinks) IsEmpty() bool {
	return p.Rovo == nil && p.Reclub == nil && p.Telegram == nil &&
		p.Instagram == nil && p.Facebook == nil && p.X == nil &&
		p.StravaAthleteID == nil
}

func normalizeProfileIdentifier(name string, value **string, pattern *regexp.Regexp, max int) error {
	if *value == nil {
		return nil
	}

	normalized := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(**value), "@"))
	if normalized == "" {
		*value = nil
		return nil
	}
	if len(normalized) > max || !pattern.MatchString(normalized) {
		return fmt.Errorf("invalid %s profile identifier", name)
	}
	*value = &normalized
	return nil
}

// NormalizeBio trims a bio, converts an empty value to nil, and enforces the
// public profile length limit in Unicode characters.
func NormalizeBio(bio *string) (*string, error) {
	if bio == nil {
		return nil, nil
	}
	normalized := strings.TrimSpace(*bio)
	if normalized == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(normalized) > MaxBioLength {
		return nil, fmt.Errorf("bio must be %d characters or fewer", MaxBioLength)
	}
	return &normalized, nil
}
