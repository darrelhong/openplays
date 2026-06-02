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

func TestUpsertPlay_Insert(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	params := makePlayParams("Daniel", "Peirce Sec", 1, futureTime())
	slotsLeft := int64(6)
	params.SlotsLeft = &slotsLeft

	play, err := queries.UpsertPlay(ctx, params)
	if err != nil {
		t.Fatalf("UpsertPlay insert: %v", err)
	}
	if play.ID == "" {
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

	startsAt := futureTime()
	venueID := int64(1)

	// First insert
	params1 := makePlayParams("Daniel", "Peirce Sec", venueID, startsAt)
	slotsLeft1 := int64(6)
	fee1 := int64(1200)
	params1.SlotsLeft = &slotsLeft1
	params1.Fee = &fee1

	play1, err := queries.UpsertPlay(ctx, params1)
	if err != nil {
		t.Fatalf("first UpsertPlay: %v", err)
	}

	// Second upsert with same dedup key but different slots/fee
	params2 := makePlayParams("Daniel", "Peirce Secondary School", venueID, startsAt)
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
		t.Errorf("expected same ID %s, got %s (new row created instead of update)", play1.ID, play2.ID)
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

func TestUpsertPlay_DifferentVenueID_InsertsBoth(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	startsAt := futureTime()
	params1 := makePlayParams("Daniel", "Peirce Sec", 1, startsAt)
	params2 := makePlayParams("Daniel", "Hougang CC", 2, startsAt)

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
		t.Errorf("expected 2 plays (different venue IDs), got %d", len(plays))
	}
}

func TestUpsertPlay_DifferentLevel_UpdatesSameRow(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	startsAt := futureTime()
	venueID := int64(1)

	params1 := makePlayParams("Daniel", "Peirce Sec", venueID, startsAt)
	levelMin1 := "LB"
	levelMax1 := "HB"
	slotsLeft1 := int64(6)
	params1.LevelMin = &levelMin1
	params1.LevelMax = &levelMax1
	params1.SlotsLeft = &slotsLeft1

	play1, err := queries.UpsertPlay(ctx, params1)
	if err != nil {
		t.Fatalf("first UpsertPlay: %v", err)
	}

	params2 := makePlayParams("Daniel", "Peirce Sec", venueID, startsAt)
	levelMin2 := "HI"
	levelMax2 := "A"
	slotsLeft2 := int64(3)
	params2.LevelMin = &levelMin2
	params2.LevelMax = &levelMax2
	params2.SlotsLeft = &slotsLeft2

	play2, err := queries.UpsertPlay(ctx, params2)
	if err != nil {
		t.Fatalf("second UpsertPlay: %v", err)
	}

	if play2.ID != play1.ID {
		t.Errorf("expected same ID %s, got %s", play1.ID, play2.ID)
	}
	if play2.LevelMin == nil || *play2.LevelMin != "HI" {
		t.Errorf("LevelMin = %v, want HI", play2.LevelMin)
	}
	if play2.LevelMax == nil || *play2.LevelMax != "A" {
		t.Errorf("LevelMax = %v, want A", play2.LevelMax)
	}
	if play2.SlotsLeft == nil || *play2.SlotsLeft != 3 {
		t.Errorf("SlotsLeft = %v, want 3", play2.SlotsLeft)
	}

	plays, err := queries.GetUpcomingPlays(ctx)
	if err != nil {
		t.Fatalf("GetUpcomingPlays: %v", err)
	}
	if len(plays) != 1 {
		t.Errorf("expected 1 play, got %d", len(plays))
	}
}

func TestListUpcomingPlays_DateRange(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	base := time.Now().UTC().Add(24 * time.Hour).Truncate(24 * time.Hour)
	if _, err := queries.UpsertPlay(ctx, makePlayParams("Day 1", "Peirce Sec", 1, base)); err != nil {
		t.Fatalf("insert day 1: %v", err)
	}
	if _, err := queries.UpsertPlay(ctx, makePlayParams("Day 2", "Hougang CC", 2, base.AddDate(0, 0, 1))); err != nil {
		t.Fatalf("insert day 2: %v", err)
	}
	if _, err := queries.UpsertPlay(ctx, makePlayParams("Day 3", "Bishan CC", 3, base.AddDate(0, 0, 2))); err != nil {
		t.Fatalf("insert day 3: %v", err)
	}

	rows, err := queries.ListUpcomingPlays(ctx, db.ListUpcomingPlaysParams{
		StartsAfter:  base.Format("2006-01-02 15:04:05+00:00"),
		StartsBefore: base.AddDate(0, 0, 2).Format("2006-01-02 15:04:05+00:00"),
		PageSize:     100,
	})
	if err != nil {
		t.Fatalf("ListUpcomingPlays: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].HostName != "Day 1" || rows[1].HostName != "Day 2" {
		t.Fatalf("hosts = %q, %q; want Day 1, Day 2", rows[0].HostName, rows[1].HostName)
	}
}

func TestListUpcomingPlays_ExcludesCancelled(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	userID := "cancel-list-host"
	googleID := "google-" + userID
	if _, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID:          userID,
		Email:       userID + "@example.com",
		DisplayName: "Cancel List Host",
		GoogleID:    &googleID,
	}); err != nil {
		t.Fatalf("UpsertUserByGoogleID: %v", err)
	}

	active, err := queries.CreatePlay(ctx, makeUserPlayParams("active-list-play", "Active Host", userID, futureTime()))
	if err != nil {
		t.Fatalf("CreatePlay active: %v", err)
	}
	cancelled, err := queries.CreatePlay(ctx, makeUserPlayParams("cancelled-list-play", "Cancelled Host", userID, futureTime().Add(time.Hour)))
	if err != nil {
		t.Fatalf("CreatePlay cancelled: %v", err)
	}
	if _, err := queries.CancelUserCreatedPlay(ctx, db.CancelUserCreatedPlayParams{
		ID:          cancelled.ID,
		CancelledBy: &userID,
	}); err != nil {
		t.Fatalf("CancelUserCreatedPlay: %v", err)
	}

	rows, err := queries.ListUpcomingPlays(ctx, db.ListUpcomingPlaysParams{PageSize: 100})
	if err != nil {
		t.Fatalf("ListUpcomingPlays: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows len = %d, want 1", len(rows))
	}
	if rows[0].ID != active.ID {
		t.Fatalf("row id = %s, want active %s", rows[0].ID, active.ID)
	}
}

func TestListMyUpcomingPlays_IncludesRosterRelationships(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	userID := createMyPlaysTestUser(t, ctx, queries, "my-user")
	ownerID := createMyPlaysTestUser(t, ctx, queries, "owner-user")
	otherID := createMyPlaysTestUser(t, ctx, queries, "other-user")
	base := futureTime().Truncate(time.Hour)

	hosted := createMyPlaysTestPlay(t, ctx, queries, "hosted-play", "Hosted", ownerID, base)
	if _, err := queries.CreatePlayHost(ctx, db.CreatePlayHostParams{PlayID: hosted.ID, UserID: userID}); err != nil {
		t.Fatalf("CreatePlayHost: %v", err)
	}
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{PlayID: hosted.ID, UserID: &userID, Status: model.ParticipantConfirmed}); err != nil {
		t.Fatalf("create hosted participant: %v", err)
	}

	confirmed := createMyPlaysTestPlay(t, ctx, queries, "confirmed-play", "Confirmed", ownerID, base.Add(time.Hour))
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{PlayID: confirmed.ID, UserID: &userID, Status: model.ParticipantConfirmed}); err != nil {
		t.Fatalf("create confirmed participant: %v", err)
	}

	added := createMyPlaysTestPlay(t, ctx, queries, "added-play", "Added", ownerID, base.Add(2*time.Hour))
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{PlayID: added.ID, UserID: &userID, Status: model.ParticipantAdded}); err != nil {
		t.Fatalf("create added participant: %v", err)
	}

	waitlisted := createMyPlaysTestPlay(t, ctx, queries, "waitlisted-play", "Waitlisted", ownerID, base.Add(3*time.Hour))
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{PlayID: waitlisted.ID, UserID: &userID, Status: model.ParticipantWaitlisted}); err != nil {
		t.Fatalf("create waitlisted participant: %v", err)
	}

	unrelated := createMyPlaysTestPlay(t, ctx, queries, "unrelated-play", "Unrelated", ownerID, base.Add(4*time.Hour))
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{PlayID: unrelated.ID, UserID: &otherID, Status: model.ParticipantConfirmed}); err != nil {
		t.Fatalf("create unrelated participant: %v", err)
	}

	cancelled := createMyPlaysTestPlay(t, ctx, queries, "cancelled-my-play", "Cancelled", ownerID, base.Add(5*time.Hour))
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{PlayID: cancelled.ID, UserID: &userID, Status: model.ParticipantConfirmed}); err != nil {
		t.Fatalf("create cancelled participant: %v", err)
	}
	if _, err := queries.CancelUserCreatedPlay(ctx, db.CancelUserCreatedPlayParams{ID: cancelled.ID, CancelledBy: &ownerID}); err != nil {
		t.Fatalf("CancelUserCreatedPlay: %v", err)
	}

	past := createMyPlaysTestPlay(t, ctx, queries, "past-my-play", "Past", ownerID, time.Now().UTC().Add(-4*time.Hour))
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{PlayID: past.ID, UserID: &userID, Status: model.ParticipantConfirmed}); err != nil {
		t.Fatalf("create past participant: %v", err)
	}

	rows, err := queries.ListMyUpcomingPlays(ctx, db.ListMyUpcomingPlaysParams{UserID: userID, PageSize: 20})
	if err != nil {
		t.Fatalf("ListMyUpcomingPlays: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("rows len = %d, want 4", len(rows))
	}

	gotStates := map[string]string{}
	for _, row := range rows {
		gotStates[row.ID] = row.ViewerState
	}
	wantStates := map[string]string{
		hosted.ID:     "creator",
		confirmed.ID:  "confirmed",
		added.ID:      "added",
		waitlisted.ID: "waitlisted",
	}
	for id, want := range wantStates {
		if gotStates[id] != want {
			t.Fatalf("viewer_state[%s] = %q, want %q", id, gotStates[id], want)
		}
	}
	for _, excluded := range []string{unrelated.ID, cancelled.ID, past.ID} {
		if _, ok := gotStates[excluded]; ok {
			t.Fatalf("unexpected play %s in my plays", excluded)
		}
	}

	total, err := queries.CountMyUpcomingPlays(ctx, &userID)
	if err != nil {
		t.Fatalf("CountMyUpcomingPlays: %v", err)
	}
	if total != 4 {
		t.Fatalf("total = %d, want 4", total)
	}
}

func TestListMyUpcomingPlays_PaginatesByStartsAtAndID(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	userID := createMyPlaysTestUser(t, ctx, queries, "page-user")
	ownerID := createMyPlaysTestUser(t, ctx, queries, "page-owner")
	startsAt := futureTime().Truncate(time.Hour)

	first := createMyPlaysTestPlay(t, ctx, queries, "page-a", "Page A", ownerID, startsAt)
	second := createMyPlaysTestPlay(t, ctx, queries, "page-b", "Page B", ownerID, startsAt)
	third := createMyPlaysTestPlay(t, ctx, queries, "page-c", "Page C", ownerID, startsAt.Add(time.Hour))
	for _, play := range []db.Play{first, second, third} {
		if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{PlayID: play.ID, UserID: &userID, Status: model.ParticipantConfirmed}); err != nil {
			t.Fatalf("create participant for %s: %v", play.ID, err)
		}
	}

	rows, err := queries.ListMyUpcomingPlays(ctx, db.ListMyUpcomingPlaysParams{UserID: userID, PageSize: 2})
	if err != nil {
		t.Fatalf("ListMyUpcomingPlays first page: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("first page rows = %d, want 2", len(rows))
	}
	if rows[0].ID != first.ID || rows[1].ID != second.ID {
		t.Fatalf("first page ids = %q, %q; want %q, %q", rows[0].ID, rows[1].ID, first.ID, second.ID)
	}

	cursorStartsAt := second.StartsAt.Format("2006-01-02 15:04:05+00:00")
	cursorID := second.ID
	rows, err = queries.ListMyUpcomingPlays(ctx, db.ListMyUpcomingPlaysParams{
		UserID:         userID,
		CursorStartsAt: cursorStartsAt,
		CursorID:       &cursorID,
		PageSize:       2,
	})
	if err != nil {
		t.Fatalf("ListMyUpcomingPlays second page: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != third.ID {
		t.Fatalf("second page ids = %#v, want only %s", rows, third.ID)
	}
}

func createMyPlaysTestUser(t *testing.T, ctx context.Context, queries *db.Queries, id string) string {
	t.Helper()
	googleID := "google-" + id
	if _, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID:          id,
		Email:       id + "@example.com",
		DisplayName: id,
		GoogleID:    &googleID,
	}); err != nil {
		t.Fatalf("UpsertUserByGoogleID(%s): %v", id, err)
	}
	return id
}

func createMyPlaysTestPlay(t *testing.T, ctx context.Context, queries *db.Queries, id, hostName, creatorID string, startsAt time.Time) db.Play {
	t.Helper()
	play, err := queries.CreatePlay(ctx, makeUserPlayParams(id, hostName, creatorID, startsAt))
	if err != nil {
		t.Fatalf("CreatePlay(%s): %v", id, err)
	}
	return play
}

func makePlayParams(host, venue string, venueID int64, startsAt time.Time) db.UpsertPlayParams {
	source := "telegram"
	levelMin := "LB"
	levelMax := "HB"
	return db.UpsertPlayParams{
		ID:          uuid.NewString(),
		ListingType: model.ListingPlay,
		Sport:       model.SportBadminton,
		HostName:    host,
		StartsAt:    startsAt,
		EndsAt:      startsAt.Add(2 * time.Hour),
		Timezone:    "Asia/Singapore",
		Venue:       venue,
		VenueID:     &venueID,
		LevelMin:    &levelMin,
		LevelMax:    &levelMax,
		Currency:    "SGD",
		Source:      &source,
	}
}

func makeUserPlayParams(id, host, creatorID string, startsAt time.Time) db.CreatePlayParams {
	maxPlayers := int64(4)
	slotsLeft := int64(4)
	return db.CreatePlayParams{
		ID:          id,
		ListingType: model.ListingPlay,
		Sport:       model.SportBadminton,
		HostName:    host,
		StartsAt:    startsAt,
		EndsAt:      startsAt.Add(2 * time.Hour),
		Timezone:    "Asia/Singapore",
		Venue:       "SBH",
		Currency:    "SGD",
		MaxPlayers:  &maxPlayers,
		SlotsLeft:   &slotsLeft,
		CreatedBy:   &creatorID,
	}
}

// futureTime returns a time guaranteed to be in the future for test stability.
func futureTime() time.Time {
	return time.Now().UTC().Add(24 * time.Hour)
}
