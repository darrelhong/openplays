package main

import (
	"context"
	"database/sql"
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
	"openplays/server/internal/onemap"
)

// Populate venues from geocoding providers and manage aliases.
//
// Usage:
//   # Search using Google Places (default)
//   go run ./tools/venuefill/ search "Hougang Community Club" "hougang cc"
//
//   # Search using OneMap
//   go run ./tools/venuefill/ search --onemap "Hougang Community Club" "hougang cc"
//
//   # Add aliases to an existing venue by postal code
//   go run ./tools/venuefill/ alias 538840 "hougang cc" "hougang community club"
//
//   # List all venues and aliases
//   go run ./tools/venuefill/ list

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "openplays_local.db"
	}

	sqlDb, err := sql.Open("sqlite", dbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer sqlDb.Close()

	queries := db.New(sqlDb)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch os.Args[1] {
	case "search":
		cmdSearch(ctx, queries, os.Args[2:])
	case "alias":
		cmdAlias(ctx, queries, os.Args[2:])
	case "list":
		cmdList(ctx, queries)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage:
  go run ./tools/venuefill/ search [--onemap] <search_term> [aliases...]
  go run ./tools/venuefill/ alias <postal_code> <alias> [aliases...]
  go run ./tools/venuefill/ list

Examples:
  search "Hougang Community Club"                          Search only, auto-alias the search term
  search "Hougang Community Club" "hougang cc" "hg cc"     Search + add extra aliases
  search --onemap "Hougang Community Club" "hougang cc"    Search via OneMap instead
  alias 538840 "hougang cc" "hg cc"                        Add aliases to existing venue`)
}

func newGeocoder(useOneMap bool) geo.Coder {
	if useOneMap {
		return onemap.NewClient(onemap.Config{
			Email:    os.Getenv("ONEMAP_EMAIL"),
			Password: os.Getenv("ONEMAP_PASSWORD"),
		})
	}
	apiKey := os.Getenv("GOOGLE_PLACES_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_PLACES_API_KEY is required (or use --onemap)")
	}
	return google.NewClient(google.Config{APIKey: apiKey})
}

func cmdSearch(ctx context.Context, queries *db.Queries, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: go run ./tools/venuefill/ search [--onemap] <query> [alias1] [alias2] ...")
		os.Exit(1)
	}

	useOneMap := false
	if args[0] == "--onemap" {
		useOneMap = true
		args = args[1:]
	}

	if len(args) < 1 {
		fmt.Println("Usage: go run ./tools/venuefill/ search [--onemap] <query> [alias1] [alias2] ...")
		os.Exit(1)
	}

	searchTerm := args[0]
	extraAliases := args[1:]
	geocoder := newGeocoder(useOneMap)

	provider := "Google Places"
	if useOneMap {
		provider = "OneMap"
	}
	fmt.Printf("Provider:    %s\n", provider)
	fmt.Printf("Search term: %q\n", searchTerm)
	if len(extraAliases) > 0 {
		fmt.Printf("Aliases:     %v\n", extraAliases)
	}
	fmt.Println()

	result, err := geocoder.Geocode(ctx, searchTerm)
	if err != nil {
		log.Fatalf("search failed: %v", err)
	}
	if result == nil {
		log.Fatalf("no results for %q", searchTerm)
	}

	fmt.Println()
	fmt.Printf("Found:    %s\n", result.Name)
	fmt.Printf("Address:  %s\n", result.Address)
	fmt.Printf("Postal:   %s\n", result.Postal)
	fmt.Printf("Location: %f, %f\n", result.Latitude, result.Longitude)
	fmt.Printf("Source:   %s\n", result.Source)

	if result.Postal == "" {
		log.Fatal("no postal code returned — cannot upsert venue without a postal code")
	}

	venue, err := queries.UpsertVenue(ctx, db.UpsertVenueParams{
		PostalCode: result.Postal,
		Name:       result.Name,
		Address:    result.Address,
		Latitude:   result.Latitude,
		Longitude:  result.Longitude,
		Source:     result.Source,
		SearchTerm: &searchTerm,
	})
	if err != nil {
		log.Fatalf("failed to upsert venue: %v", err)
	}

	fmt.Printf("\nVenue upserted: %s (%s)\n", venue.Name, venue.PostalCode)

	// Collect aliases: lowercased search term + any extra args
	aliases := []string{strings.ToLower(strings.TrimSpace(searchTerm))}
	for _, a := range extraAliases {
		a = strings.ToLower(strings.TrimSpace(a))
		if a != "" {
			aliases = append(aliases, a)
		}
	}

	fmt.Printf("\nAdding %d alias(es):\n", len(aliases))
	insertAliases(ctx, queries, venue.PostalCode, aliases)
}

func cmdAlias(ctx context.Context, queries *db.Queries, args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: go run ./tools/venuefill/ alias <postal_code> <alias1> [alias2] ...")
		os.Exit(1)
	}

	postalCode := args[0]
	var aliases []string
	for _, a := range args[1:] {
		a = strings.ToLower(strings.TrimSpace(a))
		if a != "" {
			aliases = append(aliases, a)
		}
	}

	insertAliases(ctx, queries, postalCode, aliases)
}

func cmdList(ctx context.Context, queries *db.Queries) {
	venues, err := queries.ListVenues(ctx)
	if err != nil {
		log.Fatalf("failed to list venues: %v", err)
	}

	fmt.Printf("Venues (%d):\n", len(venues))
	for _, v := range venues {
		searchTerm := ""
		if v.SearchTerm != nil {
			searchTerm = *v.SearchTerm
		}
		fmt.Printf("  %-8s %-40s %-8s %s\n", v.PostalCode, v.Name, v.Source, searchTerm)
	}

	aliases, err := queries.ListAliases(ctx)
	if err != nil {
		log.Fatalf("failed to list aliases: %v", err)
	}

	fmt.Printf("\nAliases (%d):\n", len(aliases))
	for _, a := range aliases {
		fmt.Printf("  %-40s → %s (%s)\n", a.Alias, a.VenuePostalCode, a.VenueName)
	}
}

func insertAliases(ctx context.Context, queries *db.Queries, postalCode string, aliases []string) {
	for _, alias := range aliases {
		err := queries.UpsertVenueAlias(ctx, db.UpsertVenueAliasParams{
			Alias:           alias,
			VenuePostalCode: postalCode,
		})
		if err != nil {
			log.Printf("  failed %q: %v", alias, err)
		} else {
			fmt.Printf("  %q → %s\n", alias, postalCode)
		}
	}
}
