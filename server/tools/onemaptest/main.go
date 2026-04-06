package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"openplays/server/internal/onemap"
)

// Quick test tool: search OneMap for a venue.
//
// Usage:
//   go run ./tools/onemaptest/ "Hougang CC"
//   go run ./tools/onemaptest/ "Peirce Secondary School"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	query := strings.Join(os.Args[1:], " ")
	if query == "" {
		fmt.Println("Usage: go run ./tools/onemaptest/ <query>")
		os.Exit(1)
	}

	client := onemap.NewClient(onemap.Config{
		Email:    os.Getenv("ONEMAP_EMAIL"),
		Password: os.Getenv("ONEMAP_PASSWORD"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Searching: %q\n\n", query)

	resp, err := client.SearchAll(ctx, query)
	if err != nil {
		log.Fatalf("search failed: %v", err)
	}

	out, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(out))
}
