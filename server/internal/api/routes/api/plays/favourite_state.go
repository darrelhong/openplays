package plays

import (
	"context"
	"fmt"

	"openplays/server/internal/db"
)

func hydrateFavouriteStates(ctx context.Context, store FavouriteStateBatchStore, items []PlayPublic, userID string) error {
	if len(items) == 0 {
		return nil
	}

	playIDs := make([]string, 0, len(items))
	for i := range items {
		playIDs = append(playIDs, items[i].ID)
	}

	ids, err := store.ListFavouritedPlayIDsByUserAndPlays(ctx, db.ListFavouritedPlayIDsByUserAndPlaysParams{
		UserID:  userID,
		PlayIds: playIDs,
	})
	if err != nil {
		return fmt.Errorf("list favourited play ids: %w", err)
	}

	favourited := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		favourited[id] = struct{}{}
	}

	for i := range items {
		value := false
		if _, ok := favourited[items[i].ID]; ok {
			value = true
		}
		items[i].IsFavourited = &value
	}

	return nil
}

func markFavourited(items []PlayPublic) {
	for i := range items {
		value := true
		items[i].IsFavourited = &value
	}
}
