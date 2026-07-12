package db_test

import (
	"context"
	"testing"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

func TestParticipantPreviewQueries(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	play := createParticipantTestPlay(t, ctx, queries, "Host", "Peirce Sec")
	userID := createParticipantTestUser(t, ctx, queries, "user-1")
	waitlistedUserID := createParticipantTestUser(t, ctx, queries, "user-2")
	photoURL := "https://example.com/user-1.png"
	if _, err := sqlDB.ExecContext(ctx, "UPDATE users SET photo_url = ? WHERE id = ?", photoURL, userID); err != nil {
		t.Fatalf("set user photo: %v", err)
	}
	sportsProfile := `{"tennis":{"level":"4.2"}}`
	if _, err := queries.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID:            userID,
		DisplayName:   "Alice Tan",
		SportsProfile: &sportsProfile,
	}); err != nil {
		t.Fatalf("update user profile: %v", err)
	}

	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: play.ID,
		UserID: &userID,
		Status: model.ParticipantConfirmed,
	}); err != nil {
		t.Fatalf("create confirmed user participant: %v", err)
	}

	guestName := "Guest One"
	guestRating := "3.5"
	guestRatingOrd := int64(35)
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:     play.ID,
		GuestName:  &guestName,
		RatingCode: &guestRating,
		RatingOrd:  &guestRatingOrd,
		Status:     model.ParticipantConfirmed,
	}); err != nil {
		t.Fatalf("create confirmed guest participant: %v", err)
	}

	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: play.ID,
		UserID: &waitlistedUserID,
		Status: model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("create waitlisted participant: %v", err)
	}

	addedGuestName := "Added Guest"
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:    play.ID,
		GuestName: &addedGuestName,
		Status:    model.ParticipantAdded,
	}); err != nil {
		t.Fatalf("create added participant: %v", err)
	}

	otherPlay := createParticipantTestPlay(t, ctx, queries, "Other Host", "Hougang CC")
	otherGuestName := "Other Guest"
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:    otherPlay.ID,
		GuestName: &otherGuestName,
		Status:    model.ParticipantConfirmed,
	}); err != nil {
		t.Fatalf("create other confirmed participant: %v", err)
	}

	// The play-detail roster query returns every participant with their
	// status; callers partition in Go
	rows, err := queries.ListParticipantPreviewsByPlay(ctx, play.ID)
	if err != nil {
		t.Fatalf("ListParticipantPreviewsByPlay: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("preview rows = %d, want all participants", len(rows))
	}
	if rows[0].Status != model.ParticipantConfirmed {
		t.Fatalf("first row status = %q, want confirmed", rows[0].Status)
	}
	if rows[0].UserID == nil || *rows[0].UserID != userID {
		t.Fatalf("first row user_id = %v, want %q", rows[0].UserID, userID)
	}
	if rows[0].DisplayName == nil || *rows[0].DisplayName != "Alice Tan" {
		t.Fatalf("first row display_name = %v, want Alice Tan", rows[0].DisplayName)
	}
	if rows[0].PhotoUrl == nil || *rows[0].PhotoUrl != photoURL {
		t.Fatalf("first row photo_url = %v, want %q", rows[0].PhotoUrl, photoURL)
	}
	if rows[0].SportsProfile == nil || *rows[0].SportsProfile != sportsProfile {
		t.Fatalf("first row sports_profile = %v, want %q", rows[0].SportsProfile, sportsProfile)
	}
	if rows[1].GuestName == nil || *rows[1].GuestName != guestName {
		t.Fatalf("second row guest_name = %v, want %q", rows[1].GuestName, guestName)
	}
	if rows[1].RatingCode == nil || *rows[1].RatingCode != guestRating {
		t.Fatalf("second row rating_code = %v, want %q", rows[1].RatingCode, guestRating)
	}

	// The card preview covers the slot-reserving roster: confirmed first,
	// then added; waitlisted is excluded
	batchRows, err := queries.ListRosteredParticipantPreviewsByPlays(ctx, []string{play.ID, otherPlay.ID})
	if err != nil {
		t.Fatalf("ListRosteredParticipantPreviewsByPlays: %v", err)
	}
	if len(batchRows) != 4 {
		t.Fatalf("batch preview rows = %d, want 4", len(batchRows))
	}
	countsByPlayID := map[string]int{}
	for _, row := range batchRows {
		countsByPlayID[row.PlayID]++
	}
	if countsByPlayID[play.ID] != 3 {
		t.Fatalf("batch rows for first play = %d, want 3 (confirmed + added)", countsByPlayID[play.ID])
	}
	if countsByPlayID[otherPlay.ID] != 1 {
		t.Fatalf("batch rows for other play = %d, want 1", countsByPlayID[otherPlay.ID])
	}
	var firstPlayRows []db.ListRosteredParticipantPreviewsByPlaysRow
	for _, row := range batchRows {
		if row.PlayID == play.ID {
			firstPlayRows = append(firstPlayRows, row)
		}
	}
	last := firstPlayRows[len(firstPlayRows)-1]
	if last.GuestName == nil || *last.GuestName != addedGuestName {
		t.Fatalf("last preview row = %v, want the added participant sorted after confirmed", last.GuestName)
	}
}
