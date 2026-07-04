package plays

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/pagination"
	"openplays/server/internal/db"
)

func RegisterMyPastList(api huma.API, store MyPlayStore, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "list-my-past-plays",
		Summary:     "List current user's past plays",
		Description: "Returns ended plays, cancelled ones included, where the current user was hosting or on the roster. Newest first.",
		Method:      http.MethodGet,
		Path:        "/me/plays/past",
		Tags:        []string{"Me"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *MyListInput) (*MyListOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		items, total, err := listMyPastPlays(ctx, store, user.ID, input)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list my past plays", err)
		}

		page := pagination.Paginate(items, input.Limit, total, func(p PlayPublic) string {
			return encodeTimeCursor(p.StartsAt, p.ID)
		})
		if err := hydrateFavouriteStates(ctx, store, page.Items, user.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to list favourite states", err)
		}
		if err := hydrateParticipantPreviews(ctx, store, page.Items, true); err != nil {
			return nil, huma.Error500InternalServerError("failed to list participant previews", err)
		}

		return &MyListOutput{Body: page}, nil
	})
}

func listMyPastPlays(ctx context.Context, store MyPlayStore, userID string, input *MyListInput) ([]PlayPublic, int64, error) {
	pageSize := input.Limit + 1

	var cursorStartsAt interface{}
	var cursorID *string
	if startsAt, id, ok := decodeTimeCursor(input.Cursor); ok {
		if dbStartsAt, ok := cursorStartsAtForDB(startsAt); ok {
			cursorStartsAt = dbStartsAt
			cursorID = &id
		}
	}

	rows, err := store.ListMyPastPlays(ctx, db.ListMyPastPlaysParams{
		UserID:         userID,
		CursorStartsAt: cursorStartsAt,
		CursorID:       cursorID,
		PageSize:       pageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list my past plays: %w", err)
	}

	total, err := store.CountMyPastPlays(ctx, &userID)
	if err != nil {
		return nil, 0, fmt.Errorf("count my past plays: %w", err)
	}

	items := make([]PlayPublic, len(rows))
	for i, r := range rows {
		// The row shape matches the upcoming list's exactly
		items[i] = mapMyTimeRow(db.ListMyUpcomingPlaysRow(r))
	}

	return items, total, nil
}
