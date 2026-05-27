package plays

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
)

type DeletePlayInput struct {
	ID string `path:"id" doc:"Play ID"`
}

type DeletePlayStore interface {
	GetPlayByID(ctx context.Context, id string) (db.GetPlayByIDRow, error)
	DeleteUserCreatedPlay(ctx context.Context, arg db.DeleteUserCreatedPlayParams) error
	DeletePlayParticipantsByPlay(ctx context.Context, playID string) error
}

// RegisterDelete registers DELETE /plays/{id}.
func RegisterDelete(api huma.API, store DeletePlayStore, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-play",
		Summary:     "Delete a hosted play",
		Description: "Delete a user-created play and its roster. Requires the play host.",
		Method:      http.MethodDelete,
		Path:        "/{id}",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *DeletePlayInput) (*struct{}, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}

		play, err := store.GetPlayByID(ctx, input.ID)
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("play not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get play")
		}
		if play.CreatedBy == nil {
			return nil, huma.Error422UnprocessableEntity("cannot delete imported plays")
		}
		if user.ID != *play.CreatedBy {
			return nil, huma.Error403Forbidden("only the host can delete this play")
		}

		if err := store.DeleteUserCreatedPlay(ctx, db.DeleteUserCreatedPlayParams{ID: input.ID, CreatedBy: &user.ID}); err != nil {
			return nil, huma.Error500InternalServerError("failed to delete play")
		}
		if err := store.DeletePlayParticipantsByPlay(ctx, input.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to delete play roster")
		}

		return &struct{}{}, nil
	})
}
