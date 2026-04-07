// Package geo defines a common interface for geocoding providers.
package geo

import "context"

// Result holds the resolved location data for a venue.
type Result struct {
	Name      string // canonical venue name (e.g. "Hougang Community Club")
	Address   string // full street address
	Postal    string // postal code
	Latitude  float64
	Longitude float64
	Source    string // provider that resolved this (e.g. "onemap", "google")
}

// Coder geocodes a venue name into a structured location.
// Returns nil, nil if no results are found.
type Coder interface {
	Geocode(ctx context.Context, query string) (*Result, error)
}
