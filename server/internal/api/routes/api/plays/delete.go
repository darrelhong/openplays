package plays

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

type DeletePlayInput struct {
	ID string `path:"id" doc:"Play ID"`
}

type DeletePlayStore interface {
	GetPlayByID(ctx context.Context, id string) (db.GetPlayByIDRow, error)
	GetPlayHost(ctx context.Context, arg db.GetPlayHostParams) (db.PlayHost, error)
	CancelUserCreatedPlay(ctx context.Context, arg db.CancelUserCreatedPlayParams) (db.Play, error)
	CreatePlayEvent(ctx context.Context, arg db.CreatePlayEventParams) (db.PlayEvent, error)
}

// RegisterDelete registers DELETE /plays/{id}.
func RegisterDelete(api huma.API, store DeletePlayStore, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-play",
		Summary:     "Cancel a hosted play",
		Description: "Mark a user-created play as cancelled. Requires the play host.",
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
			return nil, huma.Error422UnprocessableEntity("cannot cancel imported plays")
		}
		if err := requirePlayHost(ctx, store, input.ID, user.ID); err != nil {
			return nil, err
		}

		cancelledBy := user.ID
		if _, err := store.CancelUserCreatedPlay(ctx, db.CancelUserCreatedPlayParams{
			ID:          input.ID,
			CancelledBy: &cancelledBy,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to cancel play")
		}
		if play.CancelledAt == nil {
			actorUserID, actorDisplayName := playEventActor(user)
			if err := recordPlayEvent(ctx, store, db.CreatePlayEventParams{
				PlayID:           input.ID,
				EventType:        model.PlayEventCancelled,
				ActorUserID:      actorUserID,
				ActorDisplayName: actorDisplayName,
			}); err != nil {
				return nil, huma.Error500InternalServerError("failed to record play event")
			}
		}

		return &struct{}{}, nil
	})
}
