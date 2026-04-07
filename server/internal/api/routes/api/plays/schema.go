package plays

import "openplays/server/internal/model"

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
}
