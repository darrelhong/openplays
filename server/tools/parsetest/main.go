package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"openplays/server/internal/listener/parser"
)

// Quick test tool: pipe a message in, see parsed plays out.
//
// Usage:
//   echo "Looking for HB players..." | go run ./cmd/parsetest/
//   go run ./cmd/parsetest/ <<< "Date: 3 Apr..."
//
// Env vars:
//   LLM_BASE_URL  (default: http://localhost:1234/v1)
//   LLM_MODEL     (default: empty — uses whatever LM Studio has loaded)
//   LLM_API_KEY   (default: empty — for cloud providers)

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("failed to read stdin: %v", err)
	}

	text := string(input)
	if text == "" {
		fmt.Println("Usage: echo 'message text' | go run ./cmd/parsetest/")
		os.Exit(1)
	}

	baseURL := envOr("LLM_BASE_URL", "http://localhost:1234/v1")
	model := envOr("LLM_MODEL", "")
	apiKey := os.Getenv("LLM_API_KEY")

	cfg := parser.LLMConfig{
		BaseURL: baseURL,
		Model:   model,
		APIKey:  apiKey,
		Timeout: 500 * time.Second,
	}

	// Step 1: Split
	fmt.Println("=== SPLIT ===")
	splits := parser.SplitMessage(text)
	fmt.Printf("%d block(s) found\n\n", len(splits))

	refDate := time.Now().Format("2006-01-02")
	senderName := envOr("SENDER_NAME", "test_user")

	for i, s := range splits {
		fmt.Printf("--- Block %d/%d ---\n%s\n", i+1, len(splits), s.Block)
		if s.Shared != nil {
			fmt.Printf("  [shared context:")
			if s.Shared.Shuttle != nil {
				fmt.Printf(" shuttle=%q", *s.Shared.Shuttle)
			}
			if s.Shared.Fee != nil {
				fmt.Printf(" fee=%q", *s.Shared.Fee)
			}
			if s.Shared.LevelRaw != nil {
				fmt.Printf(" level=%q", *s.Shared.LevelRaw)
			}
			if s.Shared.MaxPax != nil {
				fmt.Printf(" max=%q", *s.Shared.MaxPax)
			}
			fmt.Println("]")
		}
		fmt.Println()
	}

	// Step 2: Print user prompts (copy-pasteable into LM Studio)
	for i, s := range splits {
		fmt.Printf("=== USER PROMPT (block %d/%d) ===\n", i+1, len(splits))
		fmt.Printf("Sender name: %s\nReference date (today): %s\n\nText block:\n%s\n\n", senderName, refDate, s.Block)
	}

	// Step 3: LLM extraction
	fmt.Println("=== LLM EXTRACT ===")
	fmt.Printf("endpoint: %s\n", cfg.BaseURL)
	if cfg.Model != "" {
		fmt.Printf("model:    %s\n", cfg.Model)
	} else {
		fmt.Println("model:    (default/loaded)")
	}
	fmt.Println()

	pipeline := parser.NewPipeline(cfg)
	tz := envOr("TIMEZONE", "Asia/Singapore")
	msgInput := parser.MessageInput{
		Text:       text,
		SenderName: senderName,
		Timestamp:  time.Now(),
		Timezone:   tz,
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	start := time.Now()
	candidates, err := pipeline.Parse(ctx, msgInput)
	elapsed := time.Since(start)

	if err != nil {
		log.Fatalf("parse failed: %v", err)
	}

	fmt.Printf("=== RESULT (%s) ===\n", elapsed.Round(time.Millisecond))

	if len(candidates) == 0 {
		fmt.Println("No plays extracted.")
		return
	}

	fmt.Printf("%d play(s) extracted:\n\n", len(candidates))

	for i, c := range candidates {
		play := parser.ToPlay(&c, msgInput)
		playJSON, _ := json.MarshalIndent(play, "", "  ")
		fmt.Printf("Play %d/%d:\n%s\n\n", i+1, len(candidates), string(playJSON))
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
