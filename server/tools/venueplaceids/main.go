package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/glebarez/sqlite"
	"github.com/joho/godotenv"

	"openplays/server/internal/db"
	"openplays/server/internal/geo"
	"openplays/server/internal/google"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	apply := flag.Bool("apply", false, "write google_place_id values; default is dry-run")
	limit := flag.Int("limit", 0, "maximum number of missing venues to process; 0 means all")
	allowPostalMismatch := flag.Bool("allow-postal-mismatch", false, "allow writing a place ID even when Google returns a different postal code")
	sleep := flag.Duration("sleep", 200*time.Millisecond, "delay between Google Places requests")
	flag.Parse()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "openplays_local.db"
	}
	apiKey := os.Getenv("GOOGLE_PLACES_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_PLACES_API_KEY is required")
	}

	sqlDB, err := sql.Open("sqlite", dbURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer sqlDB.Close()

	queries := db.New(sqlDB)
	places := google.NewClient(google.Config{APIKey: apiKey})

	venues, err := queries.ListVenuesMissingGooglePlaceIDWithName(context.Background())
	if err != nil {
		log.Fatalf("list venues missing place IDs: %v", err)
	}

	mode := "dry-run"
	if *apply {
		mode = "apply"
	}
	fmt.Printf("Mode: %s\n", mode)
	fmt.Printf("Eligible venues missing Google place IDs: %d\n\n", len(venues))

	var processed, matched, updated, skipped, failed int
	for _, venue := range venues {
		if *limit > 0 && processed >= *limit {
			break
		}
		processed++

		query := searchQuery(venue)
		result, err := geocode(context.Background(), places, query)
		if err != nil {
			failed++
			log.Printf("[%d] %s: search failed: %v", venue.ID, venue.Name, err)
			sleepBetweenRequests(*sleep)
			continue
		}
		if result == nil || strings.TrimSpace(result.PlaceID) == "" {
			skipped++
			log.Printf("[%d] %s: no Google place ID found", venue.ID, venue.Name)
			sleepBetweenRequests(*sleep)
			continue
		}
		if !*allowPostalMismatch && !postalMatches(venue.PostalCode, result.Postal) {
			skipped++
			log.Printf("[%d] %s: postal mismatch db=%s google=%s place_id=%s",
				venue.ID, venue.Name, postalLabel(venue.PostalCode), result.Postal, result.PlaceID)
			sleepBetweenRequests(*sleep)
			continue
		}

		matched++
		fmt.Printf("[%d] %-40s -> %-28s %s\n", venue.ID, venue.Name, result.PlaceID, result.Name)
		if *apply {
			placeID := strings.TrimSpace(result.PlaceID)
			if _, err := queries.UpdateVenueGooglePlaceID(context.Background(), db.UpdateVenueGooglePlaceIDParams{
				GooglePlaceID: &placeID,
				ID:            venue.ID,
			}); err != nil {
				failed++
				log.Printf("[%d] %s: update failed: %v", venue.ID, venue.Name, err)
			} else {
				updated++
			}
		}
		sleepBetweenRequests(*sleep)
	}

	fmt.Printf("\nProcessed: %d\nMatched:   %d\nUpdated:   %d\nSkipped:   %d\nFailed:    %d\n", processed, matched, updated, skipped, failed)
	if !*apply {
		fmt.Println("\nDry run only. Re-run with --apply to write google_place_id values.")
	}
}

func geocode(ctx context.Context, places geo.Coder, query string) (*geo.Result, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	return places.Geocode(reqCtx, query)
}

func searchQuery(venue db.Venue) string {
	parts := []string{strings.TrimSpace(venue.Name)}
	if venue.PostalCode != nil && strings.TrimSpace(*venue.PostalCode) != "" {
		parts = append(parts, strings.TrimSpace(*venue.PostalCode))
	}
	parts = append(parts, "Singapore")
	return strings.Join(parts, " ")
}

func postalMatches(dbPostal *string, googlePostal string) bool {
	dbPostalValue := strings.TrimSpace(postalLabel(dbPostal))
	googlePostal = strings.TrimSpace(googlePostal)
	return dbPostalValue == "" || googlePostal == "" || dbPostalValue == googlePostal
}

func postalLabel(postal *string) string {
	if postal == nil {
		return ""
	}
	return *postal
}

func sleepBetweenRequests(duration time.Duration) {
	if duration > 0 {
		time.Sleep(duration)
	}
}
