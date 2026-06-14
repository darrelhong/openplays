package plays

import (
	"context"
	"fmt"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

func hydrateViewerStates(ctx context.Context, store ViewerStateBatchStore, items []PlayPublic, userID string) error {
	if len(items) == 0 {
		return nil
	}

	playIDs := make([]string, 0, len(items))
	for i := range items {
		playIDs = append(playIDs, items[i].ID)
	}

	hostRows, err := store.ListPlayHostUserIDsByPlays(ctx, playIDs)
	if err != nil {
		return fmt.Errorf("list play hosts: %w", err)
	}

	hostPlayIDs := make(map[string]struct{}, len(hostRows))
	for _, row := range hostRows {
		if row.UserID == userID {
			hostPlayIDs[row.PlayID] = struct{}{}
		}
	}

	participantRows, err := store.ListPlayParticipantStatesByUserAndPlays(ctx, db.ListPlayParticipantStatesByUserAndPlaysParams{
		UserID:  &userID,
		PlayIds: playIDs,
	})
	if err != nil {
		return fmt.Errorf("list play participant states: %w", err)
	}

	participantStates := make(map[string]model.PlayParticipantStatus, len(participantRows))
	for _, row := range participantRows {
		participantStates[row.PlayID] = row.Status
	}

	for i := range items {
		state := "not_joined"
		isCreator := items[i].CreatedBy != nil && *items[i].CreatedBy == userID
		if _, ok := hostPlayIDs[items[i].ID]; ok || isCreator {
			state = "creator"
		} else if status, ok := participantStates[items[i].ID]; ok {
			state = string(status)
		}
		items[i].ViewerState = &state
	}

	return nil
}
