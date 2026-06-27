package plays

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/notifications"
)

type ConfirmParticipantInput struct {
	ID string `path:"id" doc:"Play ID"`
}

type ConfirmParticipantOutput struct {
	Body struct {
		Status    model.PlayParticipantStatus `json:"status"`
		SlotsLeft *int64                      `json:"slots_left,omitempty"`
	}
}

type ConfirmParticipantStore interface {
	GetPlayByID(ctx context.Context, id string) (db.GetPlayByIDRow, error)
	GetPlayParticipantByPlayAndUser(ctx context.Context, arg db.GetPlayParticipantByPlayAndUserParams) (db.PlayParticipant, error)
	UpdatePlayParticipantStatus(ctx context.Context, arg db.UpdatePlayParticipantStatusParams) (db.PlayParticipant, error)
	CountReservedPlayParticipants(ctx context.Context, playID string) (int64, error)
	ListPlayHostUserIDsByPlay(ctx context.Context, playID string) ([]string, error)
	UpdatePlaySlotsLeft(ctx context.Context, id string) error
	CreatePlayEvent(ctx context.Context, arg db.CreatePlayEventParams) (db.PlayEvent, error)
}

func RegisterConfirmParticipant(api huma.API, store ConfirmParticipantStore, authMiddleware func(huma.Context, func(huma.Context)), notifier notifications.Sender) {
	huma.Register(api, huma.Operation{
		OperationID: "confirm-play-participant",
		Summary:     "Confirm an added play spot",
		Description: "Move the authenticated user from added to confirmed after a host offers them a spot.",
		Method:      http.MethodPost,
		Path:        "/{id}/participants/me/confirm",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *ConfirmParticipantInput) (*ConfirmParticipantOutput, error) {
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
			return nil, huma.Error422UnprocessableEntity("cannot confirm imported plays")
		}
		if play.CancelledAt != nil {
			return nil, huma.Error409Conflict("play is cancelled")
		}
		if play.MaxPlayers == nil {
			return nil, huma.Error500InternalServerError("play is missing max_players")
		}

		participant, err := store.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{
			PlayID: input.ID,
			UserID: &user.ID,
		})
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("participant not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get participant")
		}

		status := participant.Status
		if participant.Status == model.ParticipantAdded {
			updated, err := store.UpdatePlayParticipantStatus(ctx, db.UpdatePlayParticipantStatusParams{
				ID:     participant.ID,
				Status: model.ParticipantConfirmed,
			})
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to confirm participant")
			}
			status = updated.Status
			actorUserID, actorDisplayName := playEventActor(user)
			participantID := participant.ID
			if err := recordPlayEvent(ctx, store, db.CreatePlayEventParams{
				PlayID:             input.ID,
				EventType:          model.PlayEventParticipantConfirmed,
				ActorUserID:        actorUserID,
				ActorDisplayName:   actorDisplayName,
				SubjectUserID:      actorUserID,
				SubjectDisplayName: actorDisplayName,
				ParticipantID:      &participantID,
			}); err != nil {
				return nil, huma.Error500InternalServerError("failed to record play event")
			}
		} else if participant.Status != model.ParticipantConfirmed {
			return nil, huma.Error409Conflict("participant has not been added")
		}

		if err := store.UpdatePlaySlotsLeft(ctx, input.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to update slots_left")
		}
		if participant.Status == model.ParticipantAdded {
			if hostUserIDs, err := store.ListPlayHostUserIDsByPlay(ctx, input.ID); err == nil {
				_ = notifications.NotifyHostsPlayerConfirmed(ctx, notifier, notifications.PlaySnapshotFromDB(play), hostUserIDs, user.ID, user.DisplayName)
			}
		}
		reservedCount, err := store.CountReservedPlayParticipants(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count participants")
		}

		slots := deriveSlotsLeft(*play.MaxPlayers, reservedCount)
		out := &ConfirmParticipantOutput{}
		out.Body.Status = status
		out.Body.SlotsLeft = &slots
		return out, nil
	})
}
