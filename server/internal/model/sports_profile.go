package model

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SportsProfile stores a user's self-declared sport-specific profile.
type SportsProfile struct {
	Badminton *SportLevelProfile `json:"badminton,omitempty"`
	Tennis    *SportLevelProfile `json:"tennis,omitempty"`
}

// SportLevelProfile stores the user's self-rating for one sport.
type SportLevelProfile struct {
	Level *string `json:"level,omitempty"`
}

// ParseSportsProfile decodes stored JSON profile data.
func ParseSportsProfile(raw *string) (*SportsProfile, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}

	var profile SportsProfile
	if err := json.Unmarshal([]byte(*raw), &profile); err != nil {
		return nil, fmt.Errorf("parse sports_profile: %w", err)
	}
	if err := profile.NormalizeAndValidate(); err != nil {
		return nil, err
	}
	if profile.IsEmpty() {
		return nil, nil
	}
	return &profile, nil
}

// SportsProfileString validates and encodes a profile for storage.
func SportsProfileString(profile *SportsProfile) (*string, error) {
	if profile == nil {
		return nil, nil
	}
	normalized := *profile
	if err := normalized.NormalizeAndValidate(); err != nil {
		return nil, err
	}
	if normalized.IsEmpty() {
		return nil, nil
	}

	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal sports_profile: %w", err)
	}
	raw := string(data)
	return &raw, nil
}

// NormalizeAndValidate trims empty levels and validates known sport scales.
func (p *SportsProfile) NormalizeAndValidate() error {
	if p == nil {
		return nil
	}
	if err := normalizeSportLevel(SportBadminton, &p.Badminton); err != nil {
		return err
	}
	if err := normalizeSportLevel(SportTennis, &p.Tennis); err != nil {
		return err
	}
	return nil
}

// LevelFor returns the profile level for a sport, when present.
func (p *SportsProfile) LevelFor(sport Sport) *string {
	if p == nil {
		return nil
	}
	switch sport {
	case SportBadminton:
		return levelValue(p.Badminton)
	case SportTennis:
		return levelValue(p.Tennis)
	default:
		return nil
	}
}

// IsEmpty reports whether the profile contains any usable sport values.
func (p *SportsProfile) IsEmpty() bool {
	return p == nil || (levelValue(p.Badminton) == nil && levelValue(p.Tennis) == nil)
}

func normalizeSportLevel(sport Sport, profile **SportLevelProfile) error {
	if profile == nil || *profile == nil {
		return nil
	}
	level := (*profile).Level
	if level == nil {
		*profile = nil
		return nil
	}
	trimmed := strings.TrimSpace(*level)
	if trimmed == "" {
		*profile = nil
		return nil
	}
	if LevelOrd(sport, trimmed) == nil {
		return fmt.Errorf("%s level %q is not supported", sport, trimmed)
	}
	(*profile).Level = &trimmed
	return nil
}

func levelValue(profile *SportLevelProfile) *string {
	if profile == nil || profile.Level == nil || *profile.Level == "" {
		return nil
	}
	return profile.Level
}
