package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"openplays/server/internal/geo"
)

const placesURL = "https://places.googleapis.com/v1/places:searchText"

type Config struct {
	APIKey string
}

// Client is a Google Places API (New) client.
type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{
		apiKey: cfg.APIKey,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

type textSearchRequest struct {
	TextQuery  string `json:"textQuery"`
	RegionCode string `json:"regionCode,omitempty"`
	PageSize   int    `json:"pageSize,omitempty"`
}

type textSearchResponse struct {
	Places []place `json:"places"`
}

type place struct {
	DisplayName      displayName   `json:"displayName"`
	FormattedAddress string        `json:"formattedAddress"`
	Location         location      `json:"location"`
	PostalAddress    postalAddress `json:"postalAddress"`
}

type displayName struct {
	Text string `json:"text"`
}

type location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type postalAddress struct {
	PostalCode string `json:"postalCode"`
}

// Geocode implements geo.Coder.
func (c *Client) Geocode(ctx context.Context, query string) (*geo.Result, error) {
	body, err := json.Marshal(textSearchRequest{
		TextQuery:  query,
		RegionCode: "SG",
		PageSize:   1,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", placesURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", c.apiKey)
	req.Header.Set("X-Goog-FieldMask", "places.displayName,places.formattedAddress,places.location,places.postalAddress")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google places: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google places: status %d: %s", resp.StatusCode, respBody)
	}

	var sr textSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("google places: decode: %w", err)
	}

	raw, _ := json.MarshalIndent(sr, "", "  ")
	log.Printf("google places search %q: %d result(s)\n%s", query, len(sr.Places), raw)

	if len(sr.Places) == 0 {
		return nil, nil
	}

	p := sr.Places[0]
	log.Printf("google places search %q → %s %s", query, p.DisplayName.Text, p.FormattedAddress)

	return &geo.Result{
		Name:      p.DisplayName.Text,
		Address:   p.FormattedAddress,
		Postal:    p.PostalAddress.PostalCode,
		Latitude:  p.Location.Latitude,
		Longitude: p.Location.Longitude,
		Source:    "google",
	}, nil
}
