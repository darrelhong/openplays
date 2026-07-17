package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

func TestUpcomingPlayAvailabilityFilter(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	venue, err := queries.UpsertVenue(ctx, db.UpsertVenueParams{
		Name:      "Availability Test Venue",
		Address:   "1 Test Street",
		Latitude:  1.3,
		Longitude: 103.8,
		Source:    "manual",
	})
	if err != nil {
		t.Fatalf("UpsertVenue: %v", err)
	}

	insertAvailabilityPlay(t, ctx, queries, venue.ID, "available", int64Ptr(2))
	insertAvailabilityPlay(t, ctx, queries, venue.ID, "full", int64Ptr(0))
	insertAvailabilityPlay(t, ctx, queries, venue.ID, "unknown", nil)

	allRows, err := queries.ListUpcomingPlays(ctx, db.ListUpcomingPlaysParams{PageSize: 100})
	if err != nil {
		t.Fatalf("ListUpcomingPlays without filter: %v", err)
	}
	if len(allRows) != 3 {
		t.Fatalf("ListUpcomingPlays without filter returned %d rows, want 3", len(allRows))
	}

	timeRows, err := queries.ListUpcomingPlays(ctx, db.ListUpcomingPlaysParams{
		Availability: "available",
		PageSize:     100,
	})
	if err != nil {
		t.Fatalf("ListUpcomingPlays: %v", err)
	}
	if len(timeRows) != 1 || timeRows[0].HostName != "available" {
		t.Fatalf("ListUpcomingPlays hosts = %v, want [available]", timeRowHosts(timeRows))
	}

	timeCount, err := queries.CountUpcomingPlays(ctx, db.CountUpcomingPlaysParams{
		Availability: "available",
	})
	if err != nil {
		t.Fatalf("CountUpcomingPlays: %v", err)
	}
	if timeCount != 1 {
		t.Fatalf("CountUpcomingPlays = %d, want 1", timeCount)
	}

	distanceRows, err := queries.ListUpcomingPlaysByDistance(ctx, db.ListUpcomingPlaysByDistanceParams{
		RefLat:       1.3,
		RefLng:       103.8,
		Availability: "available",
		PageSize:     100,
	})
	if err != nil {
		t.Fatalf("ListUpcomingPlaysByDistance: %v", err)
	}
	if len(distanceRows) != 1 || distanceRows[0].HostName != "available" {
		t.Fatalf("ListUpcomingPlaysByDistance returned %d rows, want [available]", len(distanceRows))
	}

	distanceCount, err := queries.CountUpcomingPlaysByDistance(ctx, db.CountUpcomingPlaysByDistanceParams{
		Availability: "available",
	})
	if err != nil {
		t.Fatalf("CountUpcomingPlaysByDistance: %v", err)
	}
	if distanceCount != 1 {
		t.Fatalf("CountUpcomingPlaysByDistance = %d, want 1", distanceCount)
	}
}

func insertAvailabilityPlay(
	t *testing.T,
	ctx context.Context,
	queries *db.Queries,
	venueID int64,
	host string,
	slotsLeft *int64,
) {
	t.Helper()
	source := "telegram"
	startsAt := time.Now().UTC().Add(24 * time.Hour)
	if _, err := queries.UpsertPlay(ctx, db.UpsertPlayParams{
		ID:          uuid.NewString(),
		ListingType: model.ListingPlay,
		Sport:       model.SportBadminton,
		HostName:    host,
		StartsAt:    startsAt,
		EndsAt:      startsAt.Add(2 * time.Hour),
		Timezone:    "Asia/Singapore",
		Venue:       "Availability Test Venue",
		VenueID:     &venueID,
		Currency:    "SGD",
		SlotsLeft:   slotsLeft,
		Source:      &source,
	}); err != nil {
		t.Fatalf("UpsertPlay(%s): %v", host, err)
	}
}

func timeRowHosts(rows []db.ListUpcomingPlaysRow) []string {
	hosts := make([]string, len(rows))
	for i, row := range rows {
		hosts[i] = row.HostName
	}
	return hosts
}

func int64Ptr(value int64) *int64 {
	return &value
}
