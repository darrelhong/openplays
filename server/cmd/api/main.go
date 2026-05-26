package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"

	_ "github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"

	apiRouter "openplays/server/internal/api/routes/api"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/logging"
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

	router := chi.NewMux()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	humaAPI := humachi.New(router, huma.DefaultConfig("OpenPlays API", "0.1.0"))
	apiRouter.Register(humaAPI, queries, svc, googleVerifier, facebookVerifier, cookieSecure, devAuthEnabled)

	slog.Info("api server starting", "port", port,
		"docs", "http://localhost:"+port+"/docs",
		"spec", "http://localhost:"+port+"/openapi.json",
	)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
