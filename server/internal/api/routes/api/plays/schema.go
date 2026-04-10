package plays

import (
	"fmt"

	"openplays/server/internal/model"
)

// PlayPublic is the API response schema for a play.
type PlayPublic struct {
	ID          int64             `json:"id"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	ListingType model.ListingType `json:"listing_type"`
	Sport       model.Sport       `json:"sport"`
	GameType    *model.GameType   `json:"game_type,omitempty"`
	HostName    string            `json:"host_name"`
	StartsAt    string            `json:"starts_at"`
	EndsAt      string            `json:"ends_at"`
	Timezone    string            `json:"timezone"`
	Venue       string            `json:"venue"`
	VenueName   *string           `json:"venue_name,omitempty"`
	VenueID     *int64            `json:"venue_id,omitempty"`

	VenuePostalCode *string  `json:"venue_postal_code,omitempty"`
	VenueLatitude   *float64 `json:"venue_latitude,omitempty"`
	VenueLongitude  *float64 `json:"venue_longitude,omitempty"`

	LevelMin *string `json:"level_min,omitempty"`
	LevelMax *string `json:"level_max,omitempty"`

	Fee        *int64 `json:"fee,omitempty"` // cents
	Currency   string `json:"currency"`
	MaxPlayers *int64 `json:"max_players,omitempty"`
	SlotsLeft  *int64 `json:"slots_left,omitempty"`
	Courts     *int64 `json:"courts,omitempty"`

	Contacts   model.Contacts    `json:"contacts"`
	GenderPref *model.GenderPref `json:"gender_pref,omitempty"`
	Meta       model.Meta        `json:"meta"`

	Source           *string `json:"source,omitempty"`
	SourceSenderLink *string `json:"source_sender_link,omitempty" doc:"Link to sender's Telegram profile, e.g. t.me/username"`
	SourceMessageID  *string `json:"source_message_id,omitempty"`
	SourceGroup      *string `json:"source_group,omitempty"`
	SourceLink       *string `json:"source_link,omitempty" doc:"Deep link to original message, e.g. t.me/group/123"`

	// distanceKm is an internal field used for cursor encoding when sorting
	// by distance. Not serialized to JSON (lowercase unexported in json tag).
	distanceKm float64 `json:"-"`
}

// buildSourceLink constructs a deep link to the original message.
// Returns nil if the source is not supported or fields are missing.
func buildSourceLink(source, group, messageID *string) *string {
	if source == nil || group == nil || messageID == nil {
		return nil
	}
	if *source == "telegram" {
		link := fmt.Sprintf("https://t.me/%s/%s", *group, *messageID)
		return &link
	}
	return nil
}

// buildSenderLink constructs a link to the sender's Telegram profile.
// Returns nil if the username is not available.
func buildSenderLink(source, username *string) *string {
	if source == nil || username == nil || *username == "" {
		return nil
	}
	if *source == "telegram" {
		link := fmt.Sprintf("https://t.me/%s", *username)
		return &link
	}
	return nil
}
