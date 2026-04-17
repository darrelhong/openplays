package onemap

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"openplays/server/internal/geo"
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
	Building  string
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

	slog.Info("onemap search", "query", query, "results", sr.Found)
	if raw, err := json.MarshalIndent(sr, "", "  "); err == nil {
		slog.Info("onemap response", "raw", string(raw))
	}
	if sr.Found == 0 || len(sr.Results) == 0 {
		return nil, nil
	}

	r := sr.Results[0]
	slog.Info("onemap resolved", "query", query, "building", r.Building, "postal", r.Postal, "address", r.Address)
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
		Address:   titleCase(r.Address),
		Postal:    r.Postal,
		Building:  titleCase(r.Building),
	}, nil
}

// Geocode implements geo.Coder.
func (c *Client) Geocode(ctx context.Context, query string) (*geo.Result, error) {
	gr, err := c.Search(ctx, query)
	if err != nil || gr == nil {
		return nil, err
	}
	return &geo.Result{
		Name:      gr.Building,
		Address:   gr.Address,
		Postal:    gr.Postal,
		Latitude:  gr.Latitude,
		Longitude: gr.Longitude,
		Source:    "onemap",
	}, nil
}

// titleCase converts "HOUGANG COMMUNITY CLUB" to "Hougang Community Club".
func titleCase(s string) string {
	prev := ' '
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(prev) || prev == '(' || prev == '-' {
			prev = r
			return unicode.ToUpper(r)
		}
		prev = r
		return unicode.ToLower(r)
	}, s)
}
