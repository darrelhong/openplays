package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	if err := seedInviteScenarios(ctx, queries); err != nil {
		fatal("seed invite scenarios", err)
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

type seedPlay struct {
	ID              string
	Name            string
	Description     string
	Venue           string
	StartsAt        time.Time
	EndsAt          time.Time
	MaxPlayers      int64
	SlotsLeft       int64
	RequireWaitlist bool
	ConfirmedUsers  []string
	InvitedUsers    []string
}

// seedInviteScenarios creates stable local-only data for exercising the invite
// picker, invite lifecycle, waitlist behavior, and suggested players.
// Existing rows are preserved so rerunning seeddev is safe.
func seedInviteScenarios(ctx context.Context, queries *db.Queries) error {
	now := time.Now().UTC()
	scenarios := []seedPlay{
		{
			ID:             "seed-past-teammates-1",
			Name:           "Suggested players game 1",
			Description:    "Seed data for the suggested-player picker.",
			Venue:          "Seed Sports Hall 1",
			StartsAt:       now.Add(-4 * time.Hour),
			EndsAt:         now.Add(-2 * time.Hour),
			MaxPlayers:     4,
			SlotsLeft:      1,
			ConfirmedUsers: []string{"seed-li", "seed-mi"},
		},
		{
			ID:             "seed-past-teammates-2",
			Name:           "Suggested players game 2",
			Description:    "Seed data for suggested-player ranking.",
			Venue:          "Seed Sports Hall 2",
			StartsAt:       now.Add(-28 * time.Hour),
			EndsAt:         now.Add(-26 * time.Hour),
			MaxPlayers:     4,
			SlotsLeft:      2,
			ConfirmedUsers: []string{"seed-li"},
		},
		{
			ID:           "seed-invite-open",
			Name:         "Invite playground",
			Description:  "Invite seed-advanced here, or invite another player.",
			Venue:        "Seed Sports Hall 3",
			StartsAt:     now.Add(24 * time.Hour),
			EndsAt:       now.Add(26 * time.Hour),
			MaxPlayers:   4,
			SlotsLeft:    3,
			InvitedUsers: []string{"seed-advanced"},
		},
		{
			ID:             "seed-invite-full",
			Name:           "Full invite playground",
			Description:    "Accepting the pending invite lands on the waitlist.",
			Venue:          "Seed Sports Hall 4",
			StartsAt:       now.Add(48 * time.Hour),
			EndsAt:         now.Add(50 * time.Hour),
			MaxPlayers:     2,
			SlotsLeft:      0,
			ConfirmedUsers: []string{"seed-mi"},
			InvitedUsers:   []string{"seed-li"},
		},
		{
			ID:              "seed-invite-request",
			Name:            "Request invite playground",
			Description:     "Accepting the pending invite creates a join request.",
			Venue:           "Seed Sports Hall 5",
			StartsAt:        now.Add(72 * time.Hour),
			EndsAt:          now.Add(74 * time.Hour),
			MaxPlayers:      4,
			SlotsLeft:       3,
			RequireWaitlist: true,
			InvitedUsers:    []string{"seed-tennis"},
		},
	}

	for _, scenario := range scenarios {
		if err := ensureSeedPlay(ctx, queries, scenario); err != nil {
			return fmt.Errorf("%s: %w", scenario.ID, err)
		}
		fmt.Printf("seeded play %s (%s)\n", scenario.ID, scenario.Name)
	}
	return nil
}

func ensureSeedPlay(ctx context.Context, queries *db.Queries, scenario seedPlay) error {
	host := seedUserByID("seed-host")
	play, err := queries.GetPlayByID(ctx, scenario.ID)
	if errors.Is(err, sql.ErrNoRows) {
		maxPlayers := scenario.MaxPlayers
		slotsLeft := scenario.SlotsLeft
		name := scenario.Name
		description := scenario.Description
		createdBy := host.ID
		playRecord, createErr := queries.CreatePlay(ctx, db.CreatePlayParams{
			ID:              scenario.ID,
			ListingType:     model.ListingPlay,
			Sport:           model.SportBadminton,
			HostName:        host.DisplayName,
			Name:            &name,
			Description:     &description,
			StartsAt:        scenario.StartsAt,
			EndsAt:          scenario.EndsAt,
			Timezone:        "Asia/Singapore",
			Venue:           scenario.Venue,
			Currency:        "SGD",
			MaxPlayers:      &maxPlayers,
			SlotsLeft:       &slotsLeft,
			CreatedBy:       &createdBy,
			Visibility:      model.PlayVisibilityPublic,
			RequireWaitlist: scenario.RequireWaitlist,
		})
		if createErr != nil {
			return createErr
		}
		play = db.GetPlayByIDRow{
			ID:              playRecord.ID,
			Sport:           playRecord.Sport,
			CreatedBy:       playRecord.CreatedBy,
			Name:            playRecord.Name,
			HostName:        playRecord.HostName,
			StartsAt:        playRecord.StartsAt,
			EndsAt:          playRecord.EndsAt,
			Timezone:        playRecord.Timezone,
			Venue:           playRecord.Venue,
			VenueName:       playRecord.Venue,
			MaxPlayers:      playRecord.MaxPlayers,
			SlotsLeft:       playRecord.SlotsLeft,
			RequireWaitlist: playRecord.RequireWaitlist,
		}
	} else if err != nil {
		return err
	}

	if _, err := queries.CreatePlayHost(ctx, db.CreatePlayHostParams{PlayID: scenario.ID, UserID: host.ID}); err != nil {
		return err
	}
	if _, _, err := ensureSeedParticipant(ctx, queries, scenario.ID, host.ID, model.ParticipantConfirmed, nil); err != nil {
		return err
	}
	for _, userID := range scenario.ConfirmedUsers {
		if _, _, err := ensureSeedParticipant(ctx, queries, scenario.ID, userID, model.ParticipantConfirmed, nil); err != nil {
			return err
		}
	}
	for _, userID := range scenario.InvitedUsers {
		participant, created, err := ensureSeedParticipant(ctx, queries, scenario.ID, userID, model.ParticipantInvited, &host.ID)
		if err != nil {
			return err
		}
		if created {
			invitee := seedUserByID(userID)
			if _, err := queries.CreatePlayEvent(ctx, db.CreatePlayEventParams{
				PlayID:             scenario.ID,
				EventType:          model.PlayEventParticipantInvited,
				ActorUserID:        &host.ID,
				ActorDisplayName:   &host.DisplayName,
				SubjectUserID:      &invitee.ID,
				SubjectDisplayName: &invitee.DisplayName,
				ParticipantID:      &participant.ID,
			}); err != nil {
				return err
			}
		}
	}
	_ = play // The row is fetched above to make reruns idempotent.
	return nil
}

func ensureSeedParticipant(ctx context.Context, queries *db.Queries, playID, userID string, status model.PlayParticipantStatus, invitedBy *string) (db.PlayParticipant, bool, error) {
	userIDCopy := userID
	participant, err := queries.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{
		PlayID: playID,
		UserID: &userIDCopy,
	})
	if err == nil {
		return participant, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return db.PlayParticipant{}, false, err
	}

	user := seedUserByID(userID)
	ratingCode := user.SportsProfile.LevelFor(model.SportBadminton)
	var ratingOrd *int64
	if ratingCode != nil {
		if ord := model.LevelOrd(model.SportBadminton, *ratingCode); ord != nil {
			value := int64(*ord)
			ratingOrd = &value
		}
	}
	participant, err = queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:          playID,
		UserID:          &userIDCopy,
		RatingCode:      ratingCode,
		RatingOrd:       ratingOrd,
		Status:          status,
		InvitedByUserID: invitedBy,
	})
	return participant, true, err
}

func seedUserByID(id string) seedUser {
	for _, user := range seedUsers {
		if user.ID == id {
			return user
		}
	}
	panic("unknown seed user: " + id)
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
