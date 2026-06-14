package plays

import (
	"context"
	"testing"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

func TestHydrateViewerStates(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	viewerID := createListPreviewUser(t, ctx, queries, "viewer-state-user", "Viewer State User")
	ownerID := createListPreviewUser(t, ctx, queries, "viewer-state-owner", "Viewer State Owner")
	hostOnly := createListPreviewPlay(t, ctx, queries, "viewer-state-host", viewerID)
	confirmed := createListPreviewPlay(t, ctx, queries, "viewer-state-confirmed", ownerID)
	waitlisted := createListPreviewPlay(t, ctx, queries, "viewer-state-waitlisted", ownerID)
	createdByOnly := createListPreviewPlay(t, ctx, queries, "viewer-state-created-by", viewerID)
	unjoined := createListPreviewPlay(t, ctx, queries, "viewer-state-unjoined", ownerID)

	if _, err := queries.CreatePlayHost(ctx, db.CreatePlayHostParams{PlayID: hostOnly.ID, UserID: viewerID}); err != nil {
		t.Fatalf("CreatePlayHost: %v", err)
	}
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: confirmed.ID,
		UserID: &viewerID,
		Status: model.ParticipantConfirmed,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant confirmed: %v", err)
	}
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: waitlisted.ID,
		UserID: &viewerID,
		Status: model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant waitlisted: %v", err)
	}

	items := []PlayPublic{
		{ID: hostOnly.ID},
		{ID: confirmed.ID},
		{ID: waitlisted.ID},
		{ID: createdByOnly.ID, CreatedBy: &viewerID},
		{ID: unjoined.ID},
	}
	if err := hydrateViewerStates(ctx, queries, items, viewerID); err != nil {
		t.Fatalf("hydrateViewerStates: %v", err)
	}

	want := map[string]string{
		hostOnly.ID:      "creator",
		confirmed.ID:     "confirmed",
		waitlisted.ID:    "waitlisted",
		createdByOnly.ID: "creator",
		unjoined.ID:      "not_joined",
	}
	for _, item := range items {
		if item.ViewerState == nil {
			t.Fatalf("viewer_state for %s omitted, want %s", item.ID, want[item.ID])
		}
		if *item.ViewerState != want[item.ID] {
			t.Fatalf("viewer_state for %s = %s, want %s", item.ID, *item.ViewerState, want[item.ID])
		}
	}
}

func TestHydrateViewerStatesHostTakesPriority(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	viewerID := createListPreviewUser(t, ctx, queries, "viewer-state-host-priority", "Viewer State Host Priority")
	play := createListPreviewPlay(t, ctx, queries, "viewer-state-host-priority-play", viewerID)

	if _, err := queries.CreatePlayHost(ctx, db.CreatePlayHostParams{PlayID: play.ID, UserID: viewerID}); err != nil {
		t.Fatalf("CreatePlayHost: %v", err)
	}
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: play.ID,
		UserID: &viewerID,
		Status: model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant: %v", err)
	}

	items := []PlayPublic{{ID: play.ID}}
	if err := hydrateViewerStates(ctx, queries, items, viewerID); err != nil {
		t.Fatalf("hydrateViewerStates: %v", err)
	}
	if items[0].ViewerState == nil || *items[0].ViewerState != "creator" {
		t.Fatalf("viewer_state = %v, want creator", items[0].ViewerState)
	}
}
