package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Sport represents a supported sport type.
type Sport string

const (
	SportBadminton  Sport = "badminton"
	SportTennis     Sport = "tennis"
	SportFootball   Sport = "football"
	SportPickleball Sport = "pickleball"
)

// GenderPref represents gender preference for a play session.
type GenderPref string

const (
	GenderAll        GenderPref = "all"
	GenderMaleOnly   GenderPref = "male_only"
	GenderFemaleOnly GenderPref = "female_only"
)

// ContactMethod represents a structured way to reach a host.
type ContactMethod struct {
	Type  string `json:"type"`  // "whatsapp", "telegram", "phone", "pm"
	Value string `json:"value"` // "91065080", "@username", etc.
}

// ListingType distinguishes between different kinds of listings.
type ListingType string

const (
	ListingPlay        ListingType = "play"         // organising a game, looking for players
	ListingSellBooking ListingType = "sell_booking" // reselling/letting go a booked facility
)

// GameType represents the format of play.
type GameType string

const (
	GameDoubles      GameType = "doubles"
	GameSingles      GameType = "singles"
	GameMixedDoubles GameType = "mixed_doubles"
)

// Contacts is a slice of ContactMethod that implements sql Scanner/Valuer
// for transparent JSON storage in SQLite TEXT columns.
type Contacts []ContactMethod

func (c Contacts) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

func (c *Contacts) Scan(src any) error {
	if src == nil {
		*c = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("contacts: unsupported scan source type %T", src)
	}
	return json.Unmarshal(data, c)
}

// Meta holds sport-specific attributes as a JSON object.
// Implements sql Scanner/Valuer for transparent JSON storage in SQLite TEXT columns.
//
// Badminton: {"shuttle": "RSL Supreme", "air_con": true}
// Football:  {"pitch_type": "turf", "ball_provided": true}
// Gendered pricing: {"fee_male": 1200, "fee_female": 1100}
// Gendered levels: {"level_male_min": "HB", "level_male_max": "LI", ...}
// Misc details: {"details": "Rubber flooring, Free parking"}
type Meta map[string]any

func (m Meta) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

func (m *Meta) Scan(src any) error {
	if src == nil {
		*m = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("meta: unsupported scan source type %T", src)
	}
	return json.Unmarshal(data, m)
}

// ToUTC converts a local date ("YYYY-MM-DD") and time ("HH:MM") in the given
// IANA timezone to a UTC time.Time. Returns nil if date or time is nil/empty.
func ToUTC(date, timeStr *string, tz string) *time.Time {
	if date == nil || timeStr == nil || *date == "" || *timeStr == "" {
		return nil
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	raw := fmt.Sprintf("%s %s", *date, *timeStr)
	t, err := time.ParseInLocation("2006-01-02 15:04", raw, loc)
	if err != nil {
		return nil
	}

	utc := t.UTC()
	return &utc
}
