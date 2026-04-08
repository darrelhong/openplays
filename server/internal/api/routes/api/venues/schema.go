package venues

// VenuePublic is the API response schema for a venue.
type VenuePublic struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	PostalCode string  `json:"postal_code"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}
