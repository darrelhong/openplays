package logging

import (
	"log/slog"
	"os"
)

// Init sets up the global slog logger based on the LOG_FORMAT env var.
// - "json": JSON output (for Datadog, Betterstack, etc.)
// - anything else (default): human-readable text
func Init() {
	format := os.Getenv("LOG_FORMAT")
	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	slog.SetDefault(slog.New(handler))
}
