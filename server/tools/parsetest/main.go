package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"openplays/server/internal/listener/pipeline"
)

// Quick test tool: pipe a message in, see parsed plays out.
//
// Usage:
//
//	echo "Looking for HB players..." | go run ./tools/parsetest/
//	SENDER_NAME="Daniel" go run ./tools/parsetest/ < example_messages.txt
//
// Env vars:
//
//	LLM_BASE_URL  (default: http://localhost:1234/v1)
//	LLM_MODEL     (default: empty — uses whatever LM Studio has loaded)
//	LLM_API_KEY   (default: empty — for cloud providers)
//	SENDER_NAME   (default: test_user)
//	TIMEZONE      (default: Asia/Singapore)
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("failed to read stdin: %v", err)
	}

	text := string(input)
	if text == "" {
		fmt.Println("Usage: echo 'message text' | go run ./tools/parsetest/")
		os.Exit(1)
	}

	baseURL := envOr("LLM_BASE_URL", "http://localhost:1234/v1")
	model := envOr("LLM_MODEL", "")
	apiKey := os.Getenv("LLM_API_KEY")

	cfg := pipeline.LLMConfig{
		BaseURL: baseURL,
		Model:   model,
		APIKey:  apiKey,
		Timeout: 500 * time.Second,
	}

	senderName := envOr("SENDER_NAME", "test_user")
	tz := envOr("TIMEZONE", "Asia/Singapore")

	fmt.Println("=== LLM EXTRACT ===")
	fmt.Printf("endpoint: %s\n", cfg.BaseURL)
	if cfg.Model != "" {
		fmt.Printf("model:    %s\n", cfg.Model)
	} else {
		fmt.Println("model:    (default/loaded)")
	}
	fmt.Printf("sender:   %s\n", senderName)
	fmt.Println()

	extractor := pipeline.NewLLMExtractor(cfg)
	msgInput := pipeline.MessageInput{
		Text:       text,
		SenderName: senderName,
		Timestamp:  time.Now(),
		Timezone:   tz,
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	refDate := msgInput.Timestamp.Format("2006-01-02")
	start := time.Now()
	candidates, err := extractor.Extract(ctx, msgInput.Text, refDate, msgInput.SenderName)
	elapsed := time.Since(start)

	if err != nil {
		log.Fatalf("extract failed: %v", err)
	}

	fmt.Printf("=== RESULT (%s) ===\n", elapsed.Round(time.Millisecond))

	if len(candidates) == 0 {
		fmt.Println("No plays extracted.")
		return
	}

	fmt.Printf("%d play(s) extracted:\n\n", len(candidates))

	for i, c := range candidates {
		cJSON, _ := json.MarshalIndent(c, "", "  ")
		fmt.Printf("Candidate %d/%d:\n%s\n\n", i+1, len(candidates), string(cJSON))
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
