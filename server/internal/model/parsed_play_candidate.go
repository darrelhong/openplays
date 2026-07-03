package model

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// FlexFloat is a float64 that also unmarshals from JSON strings like "2",
// "3.5", or "3.5 courts" — LLMs sometimes quote numbers or echo surrounding
// text despite the schema saying "number". Non-numeric strings still fail,
// so genuinely malformed output (e.g. "$10") is rejected rather than
// silently misparsed.
type FlexFloat float64

func (f *FlexFloat) UnmarshalJSON(b []byte) error {
	v, err := parseFlexNumber(b)
	if err != nil {
		return fmt.Errorf("parse FlexFloat: %w", err)
	}
	*f = FlexFloat(v)
	return nil
}

// FlexInt is an int with the same string tolerance as FlexFloat.
type FlexInt int

func (f *FlexInt) UnmarshalJSON(b []byte) error {
	v, err := parseFlexNumber(b)
	if err != nil {
		return fmt.Errorf("parse FlexInt: %w", err)
	}
	*f = FlexInt(math.Round(v))
	return nil
}

func parseFlexNumber(b []byte) (float64, error) {
	s := strings.TrimSpace(string(b))
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0, fmt.Errorf("empty value")
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("%q: %w", s, err)
	}
	return v, nil
}

// ParsedPlayCandidate is the LLM parser output for a single play session.
// Ephemeral — the pipeline converts these into db.Play records for storage.
type ParsedPlayCandidate struct {
	ListingType    *string    `json:"listing_type,omitempty"`
	HostName       *string    `json:"host_name,omitempty"`
	Sport          *string    `json:"sport,omitempty"`
	GameType       *string    `json:"game_type,omitempty"`
	Date           *string    `json:"date,omitempty"`
	StartTime      *string    `json:"start_time,omitempty"`
	EndTime        *string    `json:"end_time,omitempty"`
	Venue          *string    `json:"venue,omitempty"`
	LevelMin       *string    `json:"level_min,omitempty"`
	LevelMax       *string    `json:"level_max,omitempty"`
	LevelRaw       *string    `json:"level_raw,omitempty"`
	LevelMaleMin   *string    `json:"level_male_min,omitempty"`
	LevelMaleMax   *string    `json:"level_male_max,omitempty"`
	LevelFemaleMin *string    `json:"level_female_min,omitempty"`
	LevelFemaleMax *string    `json:"level_female_max,omitempty"`
	FeeCents       *FlexInt   `json:"fee_cents,omitempty"`
	Currency       *string    `json:"currency,omitempty"`
	MaxPlayers     *FlexInt   `json:"max_players,omitempty"`
	SlotsLeft      *FlexInt   `json:"slots_left,omitempty"`
	Courts         *FlexFloat `json:"courts,omitempty"`
	GenderPref     *string    `json:"gender_pref,omitempty"`
	Shuttle        *string    `json:"shuttle,omitempty"`
	AirCon         *bool      `json:"air_con,omitempty"`
	Details        *string    `json:"details,omitempty"` // "Rubber flooring", "Free parking", etc.
	FeeMaleCents   *FlexInt   `json:"fee_male_cents,omitempty"`
	FeeFemaleCents *FlexInt   `json:"fee_female_cents,omitempty"`

	Contacts []ContactMethod `json:"contacts,omitempty"`

	// Parser metadata — not persisted
	RawBlock   string             `json:"-"`
	Confidence float64            `json:"-"`
	FieldConf  map[string]float64 `json:"-"`
}
