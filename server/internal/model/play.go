// Package model defines domain types (sports, levels, listing types, etc).
// Frontend mirror: web/src/lib/consts/
// Keep sport values, game types, and levels in sync with the frontend constants.
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// Sport represents a supported sport type.
type Sport string

const (
	SportBadminton  Sport = "badminton"
	SportTennis     Sport = "tennis"
	SportFootball   Sport = "football"
	SportPickleball Sport = "pickleball"
)

// SportValues is the list of valid sport strings, for use in API enum validation.
var SportValues = []string{
	string(SportBadminton),
	string(SportTennis),
	string(SportFootball),
	string(SportPickleball),
}

// Schema implements huma.SchemaProvider so huma generates enum values in the OpenAPI spec.
func (s Sport) Schema(r huma.Registry) *huma.Schema {
	schema := r.Schema(reflect.TypeOf(""), false, "")
	for _, v := range SportValues {
		schema.Enum = append(schema.Enum, v)
	}
	return schema
}

// GenderPref represents gender preference for a play session.
type GenderPref string

const (
	GenderAll        GenderPref = "all"
	GenderMaleOnly   GenderPref = "male_only"
	GenderFemaleOnly GenderPref = "female_only"
)

// Schema implements huma.SchemaProvider.
func (g GenderPref) Schema(r huma.Registry) *huma.Schema {
	schema := r.Schema(reflect.TypeOf(""), false, "")
	schema.Enum = []any{string(GenderAll), string(GenderMaleOnly), string(GenderFemaleOnly)}
	return schema
}

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

// Schema implements huma.SchemaProvider.
func (l ListingType) Schema(r huma.Registry) *huma.Schema {
	schema := r.Schema(reflect.TypeOf(""), false, "")
	schema.Enum = []any{string(ListingPlay), string(ListingSellBooking)}
	return schema
}

// PlayVisibility controls whether a play appears in public discovery.
type PlayVisibility string

const (
	// Public plays appear in public discovery.
	PlayVisibilityPublic PlayVisibility = "public"
	// Unlisted plays are hidden from public discovery but viewable by direct link.
	PlayVisibilityUnlisted PlayVisibility = "unlisted"
)

func (v PlayVisibility) Schema(r huma.Registry) *huma.Schema {
	schema := r.Schema(reflect.TypeOf(""), false, "")
	schema.Enum = []any{string(PlayVisibilityPublic), string(PlayVisibilityUnlisted)}
	return schema
}

// PlayParticipantStatus is a user's or guest's roster state for a play.
type PlayParticipantStatus string

const (
	ParticipantConfirmed  PlayParticipantStatus = "confirmed"
	ParticipantWaitlisted PlayParticipantStatus = "waitlisted"
	ParticipantAdded      PlayParticipantStatus = "added"
	// ParticipantRequested is the join state on require-waitlist plays: the
	// player asked to join and a host decides whether to add them to the game
	// or park them on the waitlist. Does not reserve a slot.
	ParticipantRequested PlayParticipantStatus = "requested"
)

// Schema implements huma.SchemaProvider.
func (s PlayParticipantStatus) Schema(r huma.Registry) *huma.Schema {
	schema := r.Schema(reflect.TypeOf(""), false, "")
	schema.Enum = []any{
		string(ParticipantConfirmed),
		string(ParticipantWaitlisted),
		string(ParticipantAdded),
		string(ParticipantRequested),
	}
	return schema
}

// PlayEventType identifies an immutable play activity event.
type PlayEventType string

const (
	PlayEventCreated PlayEventType = "play.created"
	PlayEventUpdated PlayEventType = "play.updated"

	// PlayEventParticipantJoined is a direct join: the player reserved a spot
	// as "added" and still confirms their participation.
	PlayEventParticipantJoined PlayEventType = "participant.joined"
	// PlayEventParticipantJoinRequested is any join into the pending queue,
	// on both classic and require-waitlist plays.
	PlayEventParticipantJoinRequested    PlayEventType = "participant.join_requested"
	PlayEventParticipantAdded            PlayEventType = "participant.added"
	PlayEventParticipantConfirmed        PlayEventType = "participant.confirmed"
	PlayEventParticipantMovedToWaitlist  PlayEventType = "participant.moved_to_waitlist"
	PlayEventParticipantLeftConfirmed    PlayEventType = "participant.left_confirmed"
	PlayEventParticipantLeftAdded        PlayEventType = "participant.left_added"
	PlayEventParticipantLeftWaitlist     PlayEventType = "participant.left_waitlist"
	PlayEventParticipantRequestWithdrawn PlayEventType = "participant.request_withdrawn"
	PlayEventParticipantRemoved          PlayEventType = "participant.removed"

	PlayEventCancelled PlayEventType = "play.cancelled"
)

// Schema implements huma.SchemaProvider.
func (e PlayEventType) Schema(r huma.Registry) *huma.Schema {
	schema := r.Schema(reflect.TypeOf(""), false, "")
	schema.Enum = []any{
		string(PlayEventCreated),
		string(PlayEventUpdated),
		string(PlayEventParticipantJoined),
		string(PlayEventParticipantJoinRequested),
		string(PlayEventParticipantAdded),
		string(PlayEventParticipantConfirmed),
		string(PlayEventParticipantMovedToWaitlist),
		string(PlayEventParticipantLeftConfirmed),
		string(PlayEventParticipantLeftAdded),
		string(PlayEventParticipantLeftWaitlist),
		string(PlayEventParticipantRequestWithdrawn),
		string(PlayEventParticipantRemoved),
		string(PlayEventCancelled),
	}
	return schema
}

// GameType represents the format of play.
type GameType string

const (
	GameDoubles      GameType = "doubles"
	GameSingles      GameType = "singles"
	GameMixedDoubles GameType = "mixed_doubles"
)

// Schema implements huma.SchemaProvider.
func (g GameType) Schema(r huma.Registry) *huma.Schema {
	schema := r.Schema(reflect.TypeOf(""), false, "")
	schema.Enum = []any{string(GameDoubles), string(GameSingles), string(GameMixedDoubles)}
	return schema
}

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
