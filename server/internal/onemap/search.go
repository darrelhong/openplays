package onemap

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// SearchResult is a single result from the OneMap elastic search endpoint.
type SearchResult struct {
	SearchVal string `json:"SEARCHVAL"`
	BlkNo     string `json:"BLK_NO"`
	RoadName  string `json:"ROAD_NAME"`
	Building  string `json:"BUILDING"`
	Address   string `json:"ADDRESS"`
	Postal    string `json:"POSTAL"`
	Latitude  string `json:"LATITUDE"`
	Longitude string `json:"LONGITUDE"`
}

// SearchResponse is the full response from the OneMap elastic search endpoint.
type SearchResponse struct {
	Found         int            `json:"found"`
	TotalNumPages int            `json:"totalNumPages"`
	PageNum       int            `json:"pageNum"`
	Results       []SearchResult `json:"results"`
}

// GeoResult is the resolved lat/lng for a venue.
type GeoResult struct {
	Latitude  float64
	Longitude float64
	Address   string
	Postal    string
}

var searchParams = url.Values{
	"searchVal":      {""},
	"returnGeom":     {"Y"},
	"getAddrDetails": {"Y"},
}

// SearchAll returns the full OneMap search response for a query.
func (c *Client) SearchAll(ctx context.Context, query string) (*SearchResponse, error) {
	params := url.Values{}
	for k, v := range searchParams {
		params[k] = v
	}
	params.Set("searchVal", query)

	var sr SearchResponse
	if err := c.get(ctx, "/common/elastic/search", params, &sr); err != nil {
		return nil, err
	}
	return &sr, nil
}

// Search geocodes a query string (venue name or address) via OneMap's
// elastic search endpoint. Returns nil if no results are found.
func (c *Client) Search(ctx context.Context, query string) (*GeoResult, error) {
	sr, err := c.SearchAll(ctx, query)
	if err != nil {
		return nil, err
	}

	if sr.Found == 0 || len(sr.Results) == 0 {
		return nil, nil
	}

	r := sr.Results[0]
	lat, err := strconv.ParseFloat(r.Latitude, 64)
	if err != nil {
		return nil, fmt.Errorf("onemap: bad latitude %q: %w", r.Latitude, err)
	}
	lng, err := strconv.ParseFloat(r.Longitude, 64)
	if err != nil {
		return nil, fmt.Errorf("onemap: bad longitude %q: %w", r.Longitude, err)
	}

	return &GeoResult{
		Latitude:  lat,
		Longitude: lng,
		Address:   r.Address,
		Postal:    r.Postal,
	}, nil
}
