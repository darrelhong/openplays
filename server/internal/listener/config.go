package listener

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds Telegram API credentials
type Config struct {
	APIID                       int
	APIHash                     string
	Phone                       string
	TargetTelegramGroupUsername string
}

// Load reads environment variables
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

	return &Config{
		APIID:                       apiID,
		APIHash:                     apiHash,
		Phone:                       phone,
		TargetTelegramGroupUsername: targetTelegramGroupUsername,
	}, nil
}
