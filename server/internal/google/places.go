package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"openplays/server/internal/geo"
)

const (
	placesURL       = "https://places.googleapis.com/v1/places:searchText"
	autocompleteURL = "https://places.googleapis.com/v1/places:autocomplete"
	placeDetailsURL = "https://places.googleapis.com/v1/places/"
)

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
	ID               string        `json:"id"`
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

type autocompleteRequest struct {
	Input               string   `json:"input"`
	IncludedRegionCodes []string `json:"includedRegionCodes,omitempty"`
	SessionToken        string   `json:"sessionToken,omitempty"`
}

type autocompleteResponse struct {
	Suggestions []autocompleteSuggestion `json:"suggestions"`
}

type autocompleteSuggestion struct {
	PlacePrediction placePrediction `json:"placePrediction"`
}

type placePrediction struct {
	PlaceID          string           `json:"placeId"`
	Text             predictionText   `json:"text"`
	StructuredFormat structuredFormat `json:"structuredFormat"`
}

type structuredFormat struct {
	MainText      predictionText `json:"mainText"`
	SecondaryText predictionText `json:"secondaryText"`
}

type predictionText struct {
	Text string `json:"text"`
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
	req.Header.Set("X-Goog-FieldMask", "places.id,places.displayName,places.formattedAddress,places.location,places.postalAddress")

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
	slog.Info("google places search", "query", query, "results", len(sr.Places), "raw", string(raw))

	if len(sr.Places) == 0 {
		return nil, nil
	}

	p := sr.Places[0]
	slog.Info("google places resolved", "query", query, "name", p.DisplayName.Text, "address", p.FormattedAddress)

	return &geo.Result{
		Name:      p.DisplayName.Text,
		Address:   p.FormattedAddress,
		Postal:    p.PostalAddress.PostalCode,
		Latitude:  p.Location.Latitude,
		Longitude: p.Location.Longitude,
		Source:    "google",
		PlaceID:   p.ID,
	}, nil
}

// Autocomplete returns lightweight Google Places predictions for a typed query.
func (c *Client) Autocomplete(ctx context.Context, query string, sessionToken string) ([]geo.Suggestion, error) {
	body, err := json.Marshal(autocompleteRequest{
		Input:               query,
		IncludedRegionCodes: []string{"sg"},
		SessionToken:        sessionToken,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", autocompleteURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", c.apiKey)
	req.Header.Set("X-Goog-FieldMask", "suggestions.placePrediction.placeId,suggestions.placePrediction.text,suggestions.placePrediction.structuredFormat")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google places autocomplete: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google places autocomplete: status %d: %s", resp.StatusCode, respBody)
	}

	var ar autocompleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("google places autocomplete: decode: %w", err)
	}

	items := make([]geo.Suggestion, 0, len(ar.Suggestions))
	for _, suggestion := range ar.Suggestions {
		prediction := suggestion.PlacePrediction
		if prediction.PlaceID == "" {
			continue
		}
		name := prediction.StructuredFormat.MainText.Text
		if name == "" {
			name = prediction.Text.Text
		}
		address := prediction.StructuredFormat.SecondaryText.Text
		if name == "" {
			name = address
			address = ""
		}
		items = append(items, geo.Suggestion{
			PlaceID: prediction.PlaceID,
			Name:    name,
			Address: address,
		})
	}
	return items, nil
}

// PlaceDetails resolves a Google place ID into canonical venue data.
func (c *Client) PlaceDetails(ctx context.Context, placeID string, sessionToken string) (*geo.Result, error) {
	placeID = strings.TrimPrefix(strings.TrimSpace(placeID), "places/")
	if placeID == "" {
		return nil, fmt.Errorf("google places details: empty place id")
	}

	u := placeDetailsURL + url.PathEscape(placeID)
	if sessionToken != "" {
		u += "?sessionToken=" + url.QueryEscape(sessionToken)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Goog-Api-Key", c.apiKey)
	req.Header.Set("X-Goog-FieldMask", "id,displayName,formattedAddress,location,postalAddress")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google places details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google places details: status %d: %s", resp.StatusCode, respBody)
	}

	var p place
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("google places details: decode: %w", err)
	}
	if p.ID == "" {
		p.ID = placeID
	}

	return &geo.Result{
		Name:      p.DisplayName.Text,
		Address:   p.FormattedAddress,
		Postal:    p.PostalAddress.PostalCode,
		Latitude:  p.Location.Latitude,
		Longitude: p.Location.Longitude,
		Source:    "google",
		PlaceID:   p.ID,
	}, nil
}
