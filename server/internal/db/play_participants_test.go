package db_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

func TestCreatePlayParticipant_UserAndGuestInvariants(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	play := createParticipantTestPlay(t, ctx, queries, "Host", "Peirce Sec")
	userID := createParticipantTestUser(t, ctx, queries, "user-1")

	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: play.ID,
		UserID: &userID,
		Status: model.ParticipantConfirmed,
	}); err != nil {
		t.Fatalf("create registered participant: %v", err)
	}

	guestName := "Guest One"
	ratingCode := "HB"
	ratingOrd := int64(30)
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:     play.ID,
		GuestName:  &guestName,
		RatingCode: &ratingCode,
		RatingOrd:  &ratingOrd,
		Status:     model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("create guest participant: %v", err)
	}

	anotherUserID := createParticipantTestUser(t, ctx, queries, "user-2")
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:    play.ID,
		UserID:    &anotherUserID,
		GuestName: &guestName,
		Status:    model.ParticipantWaitlisted,
	}); err == nil {
		t.Fatal("expected error when both user_id and guest_name are set")
	}

	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: play.ID,
		Status: model.ParticipantWaitlisted,
	}); err == nil {
		t.Fatal("expected error when neither user_id nor guest_name is set")
	}

	blankGuestName := "  "
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:    play.ID,
		GuestName: &blankGuestName,
		Status:    model.ParticipantWaitlisted,
	}); err == nil {
		t.Fatal("expected error for blank guest_name")
	}

	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:     play.ID,
		GuestName:  &guestName,
		RatingCode: &ratingCode,
		Status:     model.ParticipantWaitlisted,
	}); err == nil {
		t.Fatal("expected error when rating_code is set without rating_ord")
	}
}

func TestCreatePlayParticipant_OneRowPerUserPerPlay(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	play := createParticipantTestPlay(t, ctx, queries, "Host", "Peirce Sec")
	otherPlay := createParticipantTestPlay(t, ctx, queries, "Other Host", "Hougang CC")
	userID := createParticipantTestUser(t, ctx, queries, "user-1")

	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: play.ID,
		UserID: &userID,
		Status: model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("create first participant: %v", err)
	}

	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: play.ID,
		UserID: &userID,
		Status: model.ParticipantWaitlisted,
	}); err == nil {
		t.Fatal("expected duplicate user participant on the same play to fail")
	}

	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: otherPlay.ID,
		UserID: &userID,
		Status: model.ParticipantConfirmed,
	}); err != nil {
		t.Fatalf("same user on different play should be allowed: %v", err)
	}
}

func TestPlayParticipantStatusQueries(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	play := createParticipantTestPlay(t, ctx, queries, "Host", "Peirce Sec")
	userID := createParticipantTestUser(t, ctx, queries, "user-1")
	waitlistedUserID := createParticipantTestUser(t, ctx, queries, "user-2")
	addedUserID := createParticipantTestUser(t, ctx, queries, "user-3")
	guestName := "Guest One"

	confirmed, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: play.ID,
		UserID: &userID,
		Status: model.ParticipantConfirmed,
	})
	if err != nil {
		t.Fatalf("create confirmed participant: %v", err)
	}
	waitlisted, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: play.ID,
		UserID: &waitlistedUserID,
		Status: model.ParticipantWaitlisted,
	})
	if err != nil {
		t.Fatalf("create waitlisted participant: %v", err)
	}
	added, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: play.ID,
		UserID: &addedUserID,
		Status: model.ParticipantAdded,
	})
	if err != nil {
		t.Fatalf("create added participant: %v", err)
	}
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:    play.ID,
		GuestName: &guestName,
		Status:    model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("create waitlisted participant: %v", err)
	}

	updated, err := queries.UpdatePlayParticipantStatus(ctx, db.UpdatePlayParticipantStatusParams{
		ID:     waitlisted.ID,
		Status: model.ParticipantConfirmed,
	})
	if err != nil {
		t.Fatalf("update participant status: %v", err)
	}
	if updated.Status != model.ParticipantConfirmed {
		t.Fatalf("updated status = %q, want confirmed", updated.Status)
	}

	confirmedCount, err := queries.CountConfirmedPlayParticipants(ctx, play.ID)
	if err != nil {
		t.Fatalf("CountConfirmedPlayParticipants: %v", err)
	}
	if confirmedCount != 2 {
		t.Fatalf("confirmed count = %d, want 2", confirmedCount)
	}

	waitlistCount, err := queries.CountPlayParticipantsByStatus(ctx, db.CountPlayParticipantsByStatusParams{
		PlayID: play.ID,
		Status: model.ParticipantWaitlisted,
	})
	if err != nil {
		t.Fatalf("CountPlayParticipantsByStatus: %v", err)
	}
	if waitlistCount != 1 {
		t.Fatalf("waitlist count = %d, want 1", waitlistCount)
	}

	reservedCount, err := queries.CountReservedPlayParticipants(ctx, play.ID)
	if err != nil {
		t.Fatalf("CountReservedPlayParticipants: %v", err)
	}
	if reservedCount != 3 {
		t.Fatalf("reserved count = %d, want 3", reservedCount)
	}

	confirmedRows, err := queries.ListPlayParticipantsByPlayAndStatus(ctx, db.ListPlayParticipantsByPlayAndStatusParams{
		PlayID: play.ID,
		Status: model.ParticipantConfirmed,
	})
	if err != nil {
		t.Fatalf("ListPlayParticipantsByPlayAndStatus: %v", err)
	}
	if len(confirmedRows) != 2 {
		t.Fatalf("confirmed rows = %d, want 2", len(confirmedRows))
	}

	allRows, err := queries.ListPlayParticipantsByPlay(ctx, play.ID)
	if err != nil {
		t.Fatalf("ListPlayParticipantsByPlay: %v", err)
	}
	if len(allRows) != 4 {
		t.Fatalf("all rows = %d, want 4", len(allRows))
	}
	if allRows[0].ID != confirmed.ID || allRows[1].ID != updated.ID || allRows[2].ID != added.ID || allRows[3].Status != model.ParticipantWaitlisted {
		t.Fatalf("unexpected participant ordering: %#v", allRows)
	}
}

func TestPlayParticipantLookupAndDeleteByUser(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	play := createParticipantTestPlay(t, ctx, queries, "Host", "Peirce Sec")
	otherPlay := createParticipantTestPlay(t, ctx, queries, "Other Host", "Hougang CC")
	userID := createParticipantTestUser(t, ctx, queries, "user-1")

	first, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: play.ID,
		UserID: &userID,
		Status: model.ParticipantWaitlisted,
	})
	if err != nil {
		t.Fatalf("create first participant: %v", err)
	}
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: otherPlay.ID,
		UserID: &userID,
		Status: model.ParticipantConfirmed,
	}); err != nil {
		t.Fatalf("create second participant: %v", err)
	}

	found, err := queries.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{
		PlayID: play.ID,
		UserID: &userID,
	})
	if err != nil {
		t.Fatalf("GetPlayParticipantByPlayAndUser: %v", err)
	}
	if found.ID != first.ID {
		t.Fatalf("found participant ID = %d, want %d", found.ID, first.ID)
	}

	rows, err := queries.ListPlayParticipantsByUser(ctx, &userID)
	if err != nil {
		t.Fatalf("ListPlayParticipantsByUser: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("participant rows for user = %d, want 2", len(rows))
	}

	if err := queries.DeletePlayParticipantByPlayAndUser(ctx, db.DeletePlayParticipantByPlayAndUserParams{
		PlayID: play.ID,
		UserID: &userID,
	}); err != nil {
		t.Fatalf("DeletePlayParticipantByPlayAndUser: %v", err)
	}

	_, err = queries.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{
		PlayID: play.ID,
		UserID: &userID,
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetPlayParticipantByPlayAndUser after delete err = %v, want sql.ErrNoRows", err)
	}

	if err := queries.DeletePlayParticipant(ctx, rows[0].ID); err != nil {
		t.Fatalf("DeletePlayParticipant: %v", err)
	}
}

func createParticipantTestPlay(t *testing.T, ctx context.Context, queries *db.Queries, host, venue string) db.Play {
	t.Helper()

	play, err := queries.UpsertPlay(ctx, makePlayParams(host, venue, 1, futureTime()))
	if err != nil {
		t.Fatalf("create test play: %v", err)
	}
	return play
}

func createParticipantTestUser(t *testing.T, ctx context.Context, queries *db.Queries, id string) string {
	t.Helper()

	googleID := "google-" + id
	user, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID:          id,
		Email:       id + "@example.com",
		DisplayName: "Test " + id,
		GoogleID:    &googleID,
	})
	if err != nil {
		t.Fatalf("create test user %q: %v", id, err)
	}
	return user.ID
}
