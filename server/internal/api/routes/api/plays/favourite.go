package plays

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
)

type FavouriteInput struct {
	ID string `path:"id" doc:"Play ID"`
}

type FavouriteStore interface {
	GetFavouriteablePlayID(ctx context.Context, id string) (string, error)
	FavouritePlay(ctx context.Context, arg db.FavouritePlayParams) error
	UnfavouritePlay(ctx context.Context, arg db.UnfavouritePlayParams) error
}

func RegisterFavourite(api huma.API, store FavouriteStore, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "favourite-play",
		Summary:     "Favourite a play",
		Description: "Save an upcoming active listing to the authenticated user's favourites.",
		Method:      http.MethodPut,
		Path:        "/{id}/favourite",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *FavouriteInput) (*struct{}, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		if _, err := store.GetFavouriteablePlayID(ctx, input.ID); err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("play not found")
		} else if err != nil {
			return nil, huma.Error500InternalServerError("failed to get play")
		}

		if err := store.FavouritePlay(ctx, db.FavouritePlayParams{
			UserID: user.ID,
			PlayID: input.ID,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to favourite play")
		}

		return &struct{}{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "unfavourite-play",
		Summary:     "Unfavourite a play",
		Description: "Remove a listing from the authenticated user's favourites.",
		Method:      http.MethodDelete,
		Path:        "/{id}/favourite",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *FavouriteInput) (*struct{}, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		if err := store.UnfavouritePlay(ctx, db.UnfavouritePlayParams{
			UserID: user.ID,
			PlayID: input.ID,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to unfavourite play")
		}

		return &struct{}{}, nil
	})
}
