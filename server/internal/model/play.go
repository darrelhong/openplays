package model

import (
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

// Play is the domain representation of a sports session or court listing.
// Times are stored as UTC for cross-timezone comparison and dedup.
// Timezone is stored alongside so local display time can be reconstructed.
//
// This is NOT a database row struct. When sqlc is added, it will generate
// its own struct from the SQL schema, and a thin mapper converts between them.
type Play struct {
	ListingType ListingType
	Sport       Sport
	GameType    *GameType // "doubles", "singles", "mixed_doubles"
	HostName    string

	// Times stored in UTC. Use Timezone to convert back to local.
	StartsAt *time.Time
	EndsAt   *time.Time
	Timezone string // IANA timezone, e.g. "Asia/Singapore"

	Venue     *string
	VenueNorm *string

	LevelMin *string // sport-specific: "HB" (badminton), "3.5" (tennis)
	LevelMax *string

	Fee      *int   // smallest currency unit (cents)
	Currency string // ISO 4217: "SGD", "MYR", "USD"

	MaxPlayers *int
	SlotsLeft  *int
	Courts     *int

	Contacts   []ContactMethod
	GenderPref *GenderPref

	// Sport-specific extras ONLY. Keep narrow:
	//   Badminton: "shuttle", "air_con"
	//   Football:  "pitch_type", "ball_provided"
	//   Gendered pricing: "fee_male", "fee_female"
	Meta map[string]any

	// Source tracking
	Source               string
	SourceSenderUsername string
	SourceRawMessage     string
	SourceMessageTime    time.Time
}

// LocalDate returns the play date as "YYYY-MM-DD" in the play's timezone.
func (p *Play) LocalDate() string {
	if p.StartsAt == nil {
		return ""
	}
	loc, err := time.LoadLocation(p.Timezone)
	if err != nil {
		return p.StartsAt.Format("2006-01-02")
	}
	return p.StartsAt.In(loc).Format("2006-01-02")
}

// LocalStartTime returns "HH:MM" in the play's timezone.
func (p *Play) LocalStartTime() string {
	if p.StartsAt == nil {
		return ""
	}
	loc, err := time.LoadLocation(p.Timezone)
	if err != nil {
		return p.StartsAt.Format("15:04")
	}
	return p.StartsAt.In(loc).Format("15:04")
}

// LocalEndTime returns "HH:MM" in the play's timezone.
func (p *Play) LocalEndTime() string {
	if p.EndsAt == nil {
		return ""
	}
	loc, err := time.LoadLocation(p.Timezone)
	if err != nil {
		return p.EndsAt.Format("15:04")
	}
	return p.EndsAt.In(loc).Format("15:04")
}

// ParsedPlayCandidate is the parser output for a single message block.
// Ephemeral — the pipeline converts these into Play records.
type ParsedPlayCandidate struct {
	ListingType    *string `json:"listing_type,omitempty"`
	HostName       *string `json:"host_name,omitempty"`
	Sport          *string `json:"sport,omitempty"`
	GameType       *string `json:"game_type,omitempty"` // "doubles", "singles", "mixed_doubles"
	Date           *string `json:"date,omitempty"`
	StartTime      *string `json:"start_time,omitempty"`
	EndTime        *string `json:"end_time,omitempty"`
	Venue          *string `json:"venue,omitempty"`
	LevelMin       *string `json:"level_min,omitempty"`
	LevelMax       *string `json:"level_max,omitempty"`
	LevelRaw       *string `json:"level_raw,omitempty"`
	LevelMaleMin   *string `json:"level_male_min,omitempty"`
	LevelMaleMax   *string `json:"level_male_max,omitempty"`
	LevelFemaleMin *string `json:"level_female_min,omitempty"`
	LevelFemaleMax *string `json:"level_female_max,omitempty"`
	FeeCents       *int    `json:"fee_cents,omitempty"`
	Currency       *string `json:"currency,omitempty"`
	MaxPlayers     *int    `json:"max_players,omitempty"`
	SlotsLeft      *int    `json:"slots_left,omitempty"`
	Courts         *int    `json:"courts,omitempty"`
	GenderPref     *string `json:"gender_pref,omitempty"`
	Shuttle        *string `json:"shuttle,omitempty"`
	AirCon         *bool   `json:"air_con,omitempty"`
	Details        *string `json:"details,omitempty"` // "Rubber flooring", "Free parking", etc.
	FeeMaleCents   *int    `json:"fee_male_cents,omitempty"`
	FeeFemaleCents *int    `json:"fee_female_cents,omitempty"`

	Contacts []ContactMethod `json:"contacts,omitempty"`

	// Parser metadata — not persisted
	RawBlock   string             `json:"-"`
	Confidence float64            `json:"-"`
	FieldConf  map[string]float64 `json:"-"`
}

// VenueEntry represents a canonical venue with known aliases for fuzzy matching.
type VenueEntry struct {
	Canonical string
	Aliases   []string
	Tokens    []string
	Area      string
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

// TimeOverlaps checks if two UTC time ranges overlap.
func TimeOverlaps(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && start2.Before(end1)
}
