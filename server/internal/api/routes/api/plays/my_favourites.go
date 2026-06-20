package plays

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/pagination"
	"openplays/server/internal/db"
)

type MyFavouritesInput struct {
	Cursor string `query:"cursor" doc:"Opaque cursor from previous page"`
	Limit  int64  `query:"limit" default:"20" minimum:"1" maximum:"100" doc:"Number of results per page"`
}

type MyFavouritesOutput struct {
	Body pagination.Page[PlayPublic]
}

type MyFavouritesStore interface {
	ParticipantPreviewBatchStore
	ViewerStateBatchStore
	ListFavouriteUpcomingPlays(ctx context.Context, arg db.ListFavouriteUpcomingPlaysParams) ([]db.ListFavouriteUpcomingPlaysRow, error)
	CountFavouriteUpcomingPlays(ctx context.Context, userID string) (int64, error)
}

func RegisterMyFavourites(api huma.API, store MyFavouritesStore, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "list-my-favourites",
		Summary:     "List current user's favourite plays",
		Description: "Returns upcoming active listings favourited by the current user.",
		Method:      http.MethodGet,
		Path:        "/me/favourites",
		Tags:        []string{"Me"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *MyFavouritesInput) (*MyFavouritesOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		items, total, err := listMyFavouritesByTime(ctx, store, user.ID, input)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list my favourites", err)
		}

		page := pagination.Paginate(items, input.Limit, total, func(p PlayPublic) string {
			return encodeTimeCursor(p.StartsAt, p.ID)
		})
		markFavourited(page.Items)
		if err := hydrateViewerStates(ctx, store, page.Items, user.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to list viewer states", err)
		}
		if err := hydrateParticipantPreviews(ctx, store, page.Items, false); err != nil {
			return nil, huma.Error500InternalServerError("failed to list participant previews", err)
		}

		return &MyFavouritesOutput{Body: page}, nil
	})
}

func listMyFavouritesByTime(ctx context.Context, store MyFavouritesStore, userID string, input *MyFavouritesInput) ([]PlayPublic, int64, error) {
	pageSize := input.Limit + 1

	var cursorStartsAt interface{}
	var cursorID *string
	if startsAt, id, ok := decodeTimeCursor(input.Cursor); ok {
		if dbStartsAt, ok := cursorStartsAtForDB(startsAt); ok {
			cursorStartsAt = dbStartsAt
			cursorID = &id
		}
	}

	rows, err := store.ListFavouriteUpcomingPlays(ctx, db.ListFavouriteUpcomingPlaysParams{
		UserID:         userID,
		CursorStartsAt: cursorStartsAt,
		CursorID:       cursorID,
		PageSize:       pageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list favourite plays: %w", err)
	}

	total, err := store.CountFavouriteUpcomingPlays(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("count favourite plays: %w", err)
	}

	items := make([]PlayPublic, len(rows))
	for i, r := range rows {
		items[i] = mapFavouriteTimeRow(r)
	}

	return items, total, nil
}

func mapFavouriteTimeRow(r db.ListFavouriteUpcomingPlaysRow) PlayPublic {
	createdAt, updatedAt := publicPlayTimestamps(r.CreatedBy, r.CreatedAt, r.UpdatedAt)
	return PlayPublic{
		ID:                 r.ID,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		ListingType:        r.ListingType,
		Sport:              r.Sport,
		GameType:           r.GameType,
		HostName:           r.HostName,
		Name:               r.Name,
		Description:        r.Description,
		Visibility:         r.Visibility,
		StartsAt:           r.StartsAt.Format(time.RFC3339),
		EndsAt:             r.EndsAt.Format(time.RFC3339),
		Timezone:           r.Timezone,
		CancelledAt:        publicOptionalTimestamp(r.CancelledAt),
		Venue:              r.Venue,
		VenueName:          r.VenueName,
		VenueID:            r.VenueID,
		VenuePostalCode:    r.VenuePostalCode,
		VenueLatitude:      r.VenueLatitude,
		VenueLongitude:     r.VenueLongitude,
		VenueGooglePlaceID: r.VenueGooglePlaceID,
		LevelMin:           r.LevelMin,
		LevelMax:           r.LevelMax,
		Fee:                r.Fee,
		Currency:           r.Currency,
		MaxPlayers:         r.MaxPlayers,
		SlotsLeft:          r.SlotsLeft,
		Courts:             r.Courts,
		Contacts:           r.Contacts,
		GenderPref:         r.GenderPref,
		Meta:               r.Meta,
		Source:             r.Source,
		SourceSenderLink:   buildSenderLink(r.Source, r.SourceSenderUsername),
		SourceMessageID:    r.SourceMessageID,
		SourceGroup:        r.SourceGroup,
		SourceLink:         buildSourceLink(r.Source, r.SourceGroup, r.SourceMessageID),
		CreatedBy:          r.CreatedBy,
		CreatorDisplayName: r.CreatorDisplayName,
		CreatorUsername:    r.CreatorUsername,
		CreatorPhotoURL:    r.CreatorPhotoUrl,
	}
}
