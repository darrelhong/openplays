package model

// ParsedPlayCandidate is the LLM parser output for a single play session.
// Ephemeral — the pipeline converts these into db.Play records for storage.
type ParsedPlayCandidate struct {
	ListingType    *string  `json:"listing_type,omitempty"`
	HostName       *string  `json:"host_name,omitempty"`
	Sport          *string  `json:"sport,omitempty"`
	GameType       *string  `json:"game_type,omitempty"`
	Date           *string  `json:"date,omitempty"`
	StartTime      *string  `json:"start_time,omitempty"`
	EndTime        *string  `json:"end_time,omitempty"`
	Venue          *string  `json:"venue,omitempty"`
	LevelMin       *string  `json:"level_min,omitempty"`
	LevelMax       *string  `json:"level_max,omitempty"`
	LevelRaw       *string  `json:"level_raw,omitempty"`
	LevelMaleMin   *string  `json:"level_male_min,omitempty"`
	LevelMaleMax   *string  `json:"level_male_max,omitempty"`
	LevelFemaleMin *string  `json:"level_female_min,omitempty"`
	LevelFemaleMax *string  `json:"level_female_max,omitempty"`
	FeeCents       *int     `json:"fee_cents,omitempty"`
	Currency       *string  `json:"currency,omitempty"`
	MaxPlayers     *int     `json:"max_players,omitempty"`
	SlotsLeft      *int     `json:"slots_left,omitempty"`
	Courts         *float64 `json:"courts,omitempty"`
	GenderPref     *string  `json:"gender_pref,omitempty"`
	Shuttle        *string  `json:"shuttle,omitempty"`
	AirCon         *bool    `json:"air_con,omitempty"`
	Details        *string  `json:"details,omitempty"` // "Rubber flooring", "Free parking", etc.
	FeeMaleCents   *int     `json:"fee_male_cents,omitempty"`
	FeeFemaleCents *int     `json:"fee_female_cents,omitempty"`

	Contacts []ContactMethod `json:"contacts,omitempty"`

	// Parser metadata — not persisted
	RawBlock   string             `json:"-"`
	Confidence float64            `json:"-"`
	FieldConf  map[string]float64 `json:"-"`
}
