package db_test

import (
	"context"
	"testing"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

func TestListSuggestedPlayers_RanksEndedCoPlayersAndAppliesExclusions(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	viewerID := createParticipantTestUser(t, ctx, queries, "suggested-player-viewer")
	frequentID := createParticipantTestUser(t, ctx, queries, "frequent-suggested-player")
	occasionalID := createParticipantTestUser(t, ctx, queries, "occasional-suggested-player")
	blockedID := createParticipantTestUser(t, ctx, queries, "blocked-suggested-player")

	endedOne := createParticipantTestPlay(t, ctx, queries, "Ended Host One", "Venue One")
	endedTwo := createParticipantTestPlay(t, ctx, queries, "Ended Host Two", "Venue Two")
	upcoming := createParticipantTestPlay(t, ctx, queries, "Upcoming Host", "Venue Three")
	for _, play := range []db.Play{endedOne, endedTwo} {
		endedAt := time.Now().UTC().Add(-time.Hour)
		startedAt := endedAt.Add(-2 * time.Hour)
		if _, err := sqlDB.ExecContext(ctx, `UPDATE plays SET starts_at = ?, ends_at = ? WHERE id = ?`, startedAt, endedAt, play.ID); err != nil {
			t.Fatalf("end play %s: %v", play.ID, err)
		}
	}

	addParticipant := func(playID, userID string) {
		t.Helper()
		if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
			PlayID: playID,
			UserID: &userID,
			Status: model.ParticipantConfirmed,
		}); err != nil {
			t.Fatalf("add participant %s to %s: %v", userID, playID, err)
		}
	}

	for _, playID := range []string{endedOne.ID, endedTwo.ID, upcoming.ID} {
		addParticipant(playID, viewerID)
	}
	addParticipant(endedOne.ID, frequentID)
	addParticipant(endedTwo.ID, frequentID)
	addParticipant(endedOne.ID, occasionalID)
	addParticipant(endedOne.ID, blockedID)
	addParticipant(upcoming.ID, frequentID)

	if err := queries.CreateBlock(ctx, db.CreateBlockParams{BlockerID: viewerID, BlockedID: blockedID}); err != nil {
		t.Fatalf("block suggested player: %v", err)
	}

	rows, err := queries.ListSuggestedPlayers(ctx, db.ListSuggestedPlayersParams{
		UserID:        viewerID,
		ExcludePlayID: upcoming.ID,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("list suggested players: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != occasionalID || rows[0].PlayCount != 1 {
		t.Fatalf("excluded suggested players = %#v, want occasional player with one ended play", rows)
	}

	rows, err = queries.ListSuggestedPlayers(ctx, db.ListSuggestedPlayersParams{UserID: viewerID, Limit: 10})
	if err != nil {
		t.Fatalf("list suggested players without play exclusion: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("suggested players = %#v, want frequent and occasional players", rows)
	}
	if rows[0].ID != frequentID || rows[0].PlayCount != 2 {
		t.Fatalf("first suggested player = %#v, want frequent player with two ended plays", rows[0])
	}
	if rows[1].ID != occasionalID || rows[1].PlayCount != 1 {
		t.Fatalf("second suggested player = %#v, want occasional player with one ended play", rows[1])
	}
}
