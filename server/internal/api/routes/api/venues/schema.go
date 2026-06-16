package venues

// VenuePublic is the API response schema for a venue.
type VenuePublic struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Address       string  `json:"address"`
	PostalCode    string  `json:"postal_code"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	GooglePlaceID *string `json:"google_place_id,omitempty"`
}

// VenueSearchItem is either a saved venue or an unresolved venue suggestion.
// Saved venues include id. Unresolved suggestions include google_place_id.
type VenueSearchItem struct {
	ID            *int64   `json:"id,omitempty"`
	Name          string   `json:"name"`
	Address       string   `json:"address,omitempty"`
	PostalCode    *string  `json:"postal_code,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	GooglePlaceID *string  `json:"google_place_id,omitempty"`
}
