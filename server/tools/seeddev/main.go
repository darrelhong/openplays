package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

type seedUser struct {
	ID            string
	Email         string
	DisplayName   string
	Username      string
	GoogleID      string
	SportsProfile *model.SportsProfile
}

var seedUsers = []seedUser{
	{
		ID:          "seed-host",
		Email:       "seed-host@example.test",
		DisplayName: "Seed Host",
		Username:    "seedhost",
		GoogleID:    "dev-seed-host",
		SportsProfile: &model.SportsProfile{
			Badminton: &model.SportLevelProfile{Level: strPtr("HB")},
			Tennis:    &model.SportLevelProfile{Level: strPtr("3.5")},
		},
	},
	{
		ID:          "seed-li",
		Email:       "seed-li@example.test",
		DisplayName: "Seed Low Intermediate",
		Username:    "seedli",
		GoogleID:    "dev-seed-li",
		SportsProfile: &model.SportsProfile{
			Badminton: &model.SportLevelProfile{Level: strPtr("LI")},
		},
	},
	{
		ID:          "seed-mi",
		Email:       "seed-mi@example.test",
		DisplayName: "Seed Mid Intermediate",
		Username:    "seedmi",
		GoogleID:    "dev-seed-mi",
		SportsProfile: &model.SportsProfile{
			Badminton: &model.SportLevelProfile{Level: strPtr("MI")},
		},
	},
	{
		ID:          "seed-advanced",
		Email:       "seed-advanced@example.test",
		DisplayName: "Seed Advanced",
		Username:    "seedadvanced",
		GoogleID:    "dev-seed-advanced",
		SportsProfile: &model.SportsProfile{
			Badminton: &model.SportLevelProfile{Level: strPtr("A")},
		},
	},
	{
		ID:          "seed-tennis",
		Email:       "seed-tennis@example.test",
		DisplayName: "Seed Tennis 4.0",
		Username:    "seedtennis",
		GoogleID:    "dev-seed-tennis",
		SportsProfile: &model.SportsProfile{
			Tennis: &model.SportLevelProfile{Level: strPtr("4.0")},
		},
	},
	{
		ID:          "seed-norating",
		Email:       "seed-norating@example.test",
		DisplayName: "Seed No Rating",
		Username:    "seednorating",
		GoogleID:    "dev-seed-norating",
	},
}

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using system environment variables")
	}

	dbURL := os.Getenv("DB_URL")
	if strings.TrimSpace(dbURL) == "" {
		dbURL = "openplays_local.db"
	}

	sqlDB, err := sql.Open("sqlite", dbURL)
	if err != nil {
		fatal("open database", err)
	}
	defer sqlDB.Close()

	if err := goose.SetDialect("sqlite3"); err != nil {
		fatal("set goose dialect", err)
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(sqlDB, migrationsDir()); err != nil {
		fatal("run migrations", err)
	}

	queries := db.New(sqlDB)
	ctx := context.Background()
	for _, user := range seedUsers {
		if err := upsertSeedUser(ctx, queries, user); err != nil {
			fatal("seed user "+user.ID, err)
		}
		fmt.Printf("seeded %s (%s)\n", user.DisplayName, user.ID)
	}
}

func upsertSeedUser(ctx context.Context, queries *db.Queries, user seedUser) error {
	googleID := user.GoogleID
	created, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		GoogleID:    &googleID,
	})
	if err != nil {
		return err
	}

	sportsProfile, err := model.SportsProfileString(user.SportsProfile)
	if err != nil {
		return err
	}
	username := user.Username
	_, err = queries.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID:            created.ID,
		DisplayName:   user.DisplayName,
		Username:      &username,
		SportsProfile: sportsProfile,
	})
	return err
}

func migrationsDir() string {
	candidates := []string{
		"db/migrations",
		filepath.Join("server", "db", "migrations"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "db/migrations"
}

func strPtr(value string) *string {
	return &value
}

func fatal(action string, err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", action, err)
	os.Exit(1)
}
