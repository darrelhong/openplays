package listener

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"

	"openplays/server/internal/listener/parser"
)

// Config holds Telegram API credentials, DB, and LLM settings.
type Config struct {
	APIID                       int
	APIHash                     string
	Phone                       string
	TargetTelegramGroupUsername string
	TargetTelegramGroupTimezone string
	DBURL                       string
	LLM                         parser.LLMConfig
}

// LoadConfig reads environment variables.
func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	apiID, err := strconv.Atoi(os.Getenv("TELEGRAM_API_ID"))
	if err != nil {
		return nil, fmt.Errorf("invalid TELEGRAM_API_ID: %w", err)
	}

	apiHash := os.Getenv("TELEGRAM_API_HASH")
	if apiHash == "" {
		return nil, fmt.Errorf("TELEGRAM_API_HASH is required")
	}

	phone := os.Getenv("TELEGRAM_PHONE")
	if phone == "" {
		return nil, fmt.Errorf("TELEGRAM_PHONE is required")
	}

	targetTelegramGroupUsername := os.Getenv("TELEGRAM_GROUP_USERNAME")
	if targetTelegramGroupUsername == "" {
		return nil, fmt.Errorf("TELEGRAM_GROUP_USERNAME is required")
	}

	targetTelegramGroupTimezone := os.Getenv("TELEGRAM_GROUP_TIMEZONE")
	if targetTelegramGroupTimezone == "" {
		targetTelegramGroupTimezone = "Asia/Singapore"
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "openplays.db"
	}

	// LLM config with defaults
	llmCfg := parser.DefaultLLMConfig()
	if baseURL := os.Getenv("LLM_BASE_URL"); baseURL != "" {
		llmCfg.BaseURL = baseURL
	}
	if model := os.Getenv("LLM_MODEL"); model != "" {
		llmCfg.Model = model
	}
	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		llmCfg.APIKey = apiKey
	}

	return &Config{
		APIID:                       apiID,
		APIHash:                     apiHash,
		Phone:                       phone,
		TargetTelegramGroupUsername: targetTelegramGroupUsername,
		TargetTelegramGroupTimezone: targetTelegramGroupTimezone,
		DBURL:                       dbURL,
		LLM:                         llmCfg,
	}, nil
}
