package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"

	apiRouter "openplays/server/internal/api/routes/api"
	"openplays/server/internal/db"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "openplays_local.db"
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	sqlDb, err := sql.Open("sqlite", dbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer sqlDb.Close()

	queries := db.New(sqlDb)

	router := chi.NewMux()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	humaAPI := humachi.New(router, huma.DefaultConfig("OpenPlays API", "0.1.0"))
	apiRouter.Register(humaAPI, queries)

	fmt.Printf("OpenPlays API server on :%s\n", port)
	fmt.Printf("Docs: http://localhost:%s/docs\n", port)
	fmt.Printf("Spec: http://localhost:%s/openapi.json\n", port)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
