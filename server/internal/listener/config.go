package listener

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

	"openplays/server/internal/google"
	"openplays/server/internal/listener/pipeline"
	"openplays/server/internal/onemap"
)

// Config holds Telegram API credentials, DB, LLM, and geocoding settings.
type Config struct {
	TelegramAPIID         int
	TelegramAPIHash       string
	TelegramUserPhone     string
	TelegramSessionDB     string
	TelegramGroupUsername string
	TelegramGroupTimezone string
	DBURL                 string
	LLM                   pipeline.LLMConfig
	OneMap                onemap.Config
	Google                google.Config
}

// LoadConfig reads environment variables.
func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using system environment variables")
	}

	apiID, err := strconv.Atoi(os.Getenv("TELEGRAM_API_ID"))
	if err != nil {
		return nil, fmt.Errorf("invalid TELEGRAM_API_ID: %w", err)
	}

	apiHash := os.Getenv("TELEGRAM_API_HASH")
	if apiHash == "" {
		return nil, fmt.Errorf("TELEGRAM_API_HASH is required")
	}

	phone := os.Getenv("TELEGRAM_USER_PHONE")
	if phone == "" {
		return nil, fmt.Errorf("TELEGRAM_USER_PHONE is required")
	}

	sessionDB := os.Getenv("TELEGRAM_SESSION_DB")
	if sessionDB == "" {
		sessionDB = "tele_session.db"
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
	llmCfg := pipeline.DefaultLLMConfig()
	if baseURL := os.Getenv("LLM_BASE_URL"); baseURL != "" {
		llmCfg.BaseURL = baseURL
	}
	if model := os.Getenv("LLM_MODEL"); model != "" {
		llmCfg.Model = model
	}
	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		llmCfg.APIKey = apiKey
	}
	if rl := os.Getenv("LLM_MAX_REQ_PER_MIN"); rl != "" {
		if v, err := strconv.Atoi(rl); err == nil && v > 0 {
			llmCfg.RateLimit = v
		}
	}
	// LLM_REASONING_EFFORT lets a deployment target a model that rejects the
	// parameter (older/local builds). Set it to an empty string deliberately to
	// omit reasoning_effort altogether.
	if effort, ok := os.LookupEnv("LLM_REASONING_EFFORT"); ok {
		llmCfg.ReasoningEffort = strings.TrimSpace(effort)
	}
	if mct := os.Getenv("LLM_MAX_COMPLETION_TOKENS"); mct != "" {
		if v, err := strconv.Atoi(mct); err == nil && v > 0 {
			llmCfg.MaxCompletionTokens = v
		}
	}

	oneMapEmail := os.Getenv("ONEMAP_EMAIL")
	oneMapPassword := os.Getenv("ONEMAP_PASSWORD")
	oneMapCfg := onemap.Config{
		Email:    oneMapEmail,
		Password: oneMapPassword,
	}
	googleCfg := google.Config{
		APIKey: os.Getenv("GOOGLE_PLACES_API_KEY"),
	}

	return &Config{
		TelegramAPIID:         apiID,
		TelegramAPIHash:       apiHash,
		TelegramUserPhone:     phone,
		TelegramSessionDB:     sessionDB,
		TelegramGroupUsername: targetTelegramGroupUsername,
		TelegramGroupTimezone: targetTelegramGroupTimezone,
		DBURL:                 dbURL,
		LLM:                   llmCfg,
		OneMap:                oneMapCfg,
		Google:                googleCfg,
	}, nil
}
