package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"

	apiRouter "openplays/server/internal/api/routes/api"
	"openplays/server/internal/auth"
	"openplays/server/internal/avatar"
	"openplays/server/internal/db"
	"openplays/server/internal/geo"
	"openplays/server/internal/google"
	"openplays/server/internal/logging"
	"openplays/server/internal/notifications"
	"openplays/server/internal/objectstore"
	"openplays/server/internal/reviews"
)

func main() {
	logging.Init()

	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using system environment variables")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "openplays_local.db"
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	if googleClientID == "" {
		slog.Warn("GOOGLE_CLIENT_ID not set, Google auth will reject all tokens")
	}
	googleVerifier := auth.NewGoogleVerifier(googleClientID)

	var places geo.PlaceProvider
	if googlePlacesAPIKey := os.Getenv("GOOGLE_PLACES_API_KEY"); googlePlacesAPIKey != "" {
		places = google.NewClient(google.Config{APIKey: googlePlacesAPIKey})
		slog.Info("venue search enabled", "provider", "google_places")
	}

	facebookVerifier := auth.NewFacebookVerifier(auth.FacebookConfig{
		AppID:     os.Getenv("FACEBOOK_APP_ID"),
		AppSecret: os.Getenv("FACEBOOK_APP_SECRET"),
	})

	cookieSecure := os.Getenv("COOKIE_SECURE") != "false" // default true, set COOKIE_SECURE=false for local dev
	devAuthEnabled := os.Getenv("DEV_AUTH_ENABLED") == "true"

	sqlDb, err := sql.Open("sqlite", dbURL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer sqlDb.Close()

	queries := db.New(sqlDb)
	svc := auth.NewService(queries)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var avatarService *avatar.Service
	if os.Getenv("OBJECT_STORE_BUCKET") != "" {
		objectConfig, err := objectstore.ConfigFromEnv()
		if err != nil {
			slog.Error("invalid object store configuration", "error", err)
			os.Exit(1)
		}
		objects, err := objectstore.New(ctx, objectConfig)
		if err != nil {
			slog.Error("failed to initialize object store", "error", err)
			os.Exit(1)
		}
		defer objects.Close()
		avatarService = avatar.NewService(objects, queries, avatar.Processor{})
	} else {
		slog.Info("avatar uploads disabled; OBJECT_STORE_BUCKET is not set")
	}

	// Shared by the API routes and background workers
	pushService := notifications.MustNewSQLiteWebPushService(ctx, queries, "mailto:dev@openplays.app")

	// Nudges participants to review their co-players after a play ends
	prompter := reviews.NewPrompter(queries, pushService)
	go prompter.Run(ctx)

	router := chi.NewMux()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	humaAPI := humachi.New(router, huma.DefaultConfig("OpenPlays API", "0.1.0"))
	apiRouter.Register(humaAPI, queries, svc, avatarService, googleVerifier, facebookVerifier, places, pushService, cookieSecure, devAuthEnabled)

	slog.Info("api server starting", "port", port,
		"docs", "http://localhost:"+port+"/docs",
		"spec", "http://localhost:"+port+"/openapi.json",
	)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
