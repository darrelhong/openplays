package db_test

import (
	"context"
	"testing"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

func TestUpsertPlay_Insert(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	params := makePlayParams("Daniel", "Peirce Sec", time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC))
	slotsLeft := int64(6)
	params.SlotsLeft = &slotsLeft

	play, err := queries.UpsertPlay(ctx, params)
	if err != nil {
		t.Fatalf("UpsertPlay insert: %v", err)
	}
	if play.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if play.SlotsLeft == nil || *play.SlotsLeft != 6 {
		t.Errorf("SlotsLeft = %v, want 6", play.SlotsLeft)
	}
}

func TestUpsertPlay_UpdateOnConflict(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	startsAt := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)

	// First insert
	params1 := makePlayParams("Daniel", "Peirce Sec", startsAt)
	slotsLeft1 := int64(6)
	fee1 := int64(1200)
	params1.SlotsLeft = &slotsLeft1
	params1.Fee = &fee1

	play1, err := queries.UpsertPlay(ctx, params1)
	if err != nil {
		t.Fatalf("first UpsertPlay: %v", err)
	}

	// Second upsert with same (host_name, starts_at, venue) but different slots/fee
	params2 := makePlayParams("Daniel", "Peirce Sec", startsAt)
	slotsLeft2 := int64(3)
	fee2 := int64(1500)
	params2.SlotsLeft = &slotsLeft2
	params2.Fee = &fee2

	play2, err := queries.UpsertPlay(ctx, params2)
	if err != nil {
		t.Fatalf("second UpsertPlay: %v", err)
	}

	// Should be the same row, not a new one
	if play2.ID != play1.ID {
		t.Errorf("expected same ID %d, got %d (new row created instead of update)", play1.ID, play2.ID)
	}

	// Slots and fee should be updated
	if play2.SlotsLeft == nil || *play2.SlotsLeft != 3 {
		t.Errorf("SlotsLeft = %v, want 3", play2.SlotsLeft)
	}
	if play2.Fee == nil || *play2.Fee != 1500 {
		t.Errorf("Fee = %v, want 1500", play2.Fee)
	}

	// updated_at should be refreshed
	if !play2.UpdatedAt.After(play1.CreatedAt) && play2.UpdatedAt != play1.CreatedAt {
		t.Errorf("updated_at should be >= created_at")
	}

	// Verify only one row exists
	plays, err := queries.GetUpcomingPlays(ctx)
	if err != nil {
		t.Fatalf("GetUpcomingPlays: %v", err)
	}
	if len(plays) != 1 {
		t.Errorf("expected 1 play, got %d", len(plays))
	}
}

func TestUpsertPlay_DifferentVenue_InsertsBoth(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	startsAt := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)

	params1 := makePlayParams("Daniel", "Peirce Sec", startsAt)
	params2 := makePlayParams("Daniel", "Hougang CC", startsAt)

	if _, err := queries.UpsertPlay(ctx, params1); err != nil {
		t.Fatalf("first UpsertPlay: %v", err)
	}
	if _, err := queries.UpsertPlay(ctx, params2); err != nil {
		t.Fatalf("second UpsertPlay: %v", err)
	}

	plays, err := queries.GetUpcomingPlays(ctx)
	if err != nil {
		t.Fatalf("GetUpcomingPlays: %v", err)
	}
	if len(plays) != 2 {
		t.Errorf("expected 2 plays (different venues), got %d", len(plays))
	}
}

func makePlayParams(host, venue string, startsAt time.Time) db.UpsertPlayParams {
	source := "telegram"
	return db.UpsertPlayParams{
		ListingType: model.ListingPlay,
		Sport:       model.SportBadminton,
		HostName:    host,
		StartsAt:    startsAt,
		EndsAt:      startsAt.Add(2 * time.Hour),
		Timezone:    "Asia/Singapore",
		Venue:       venue,
		VenueNorm:   venue,
		Currency:    "SGD",
		Source:      &source,
	}
}
