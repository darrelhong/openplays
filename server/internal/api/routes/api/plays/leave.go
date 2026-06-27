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

type LeaveInput struct {
	ID string `path:"id" doc:"Play ID"`
}

type LeaveStore interface {
	GetPlayByID(ctx context.Context, id string) (db.GetPlayByIDRow, error)
	GetPlayHost(ctx context.Context, arg db.GetPlayHostParams) (db.PlayHost, error)
	GetPlayParticipantByPlayAndUser(ctx context.Context, arg db.GetPlayParticipantByPlayAndUserParams) (db.PlayParticipant, error)
	DeletePlayParticipantByPlayAndUser(ctx context.Context, arg db.DeletePlayParticipantByPlayAndUserParams) error
	ListPlayHostUserIDsByPlay(ctx context.Context, playID string) ([]string, error)
	UpdatePlaySlotsLeft(ctx context.Context, id string) error
	CreatePlayEvent(ctx context.Context, arg db.CreatePlayEventParams) (db.PlayEvent, error)
}

func RegisterLeave(api huma.API, store LeaveStore, authMiddleware func(huma.Context, func(huma.Context)), notifier notifications.Sender) {
	huma.Register(api, huma.Operation{
		OperationID: "leave-play",
		Summary:     "Leave a play",
		Description: "Remove the authenticated user from a play roster.",
		Method:      http.MethodDelete,
		Path:        "/{id}/participants/me",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *LeaveInput) (*struct{}, error) {
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
		if play.CancelledAt != nil {
			return nil, huma.Error409Conflict("play is cancelled")
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
		if participant.UserID != nil {
			if ok, err := isPlayHost(ctx, store, play.ID, *participant.UserID); err != nil {
				return nil, err
			} else if ok {
				return nil, huma.Error409Conflict("host cannot leave roster")
			}
		}

		if err := store.DeletePlayParticipantByPlayAndUser(ctx, db.DeletePlayParticipantByPlayAndUserParams{
			PlayID: input.ID,
			UserID: &user.ID,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to leave play")
		}
		actorUserID, actorDisplayName := playEventActor(user)
		participantID := participant.ID
		if err := recordPlayEvent(ctx, store, db.CreatePlayEventParams{
			PlayID:             input.ID,
			EventType:          eventTypeForLeaveStatus(participant.Status),
			ActorUserID:        actorUserID,
			ActorDisplayName:   actorDisplayName,
			SubjectUserID:      actorUserID,
			SubjectDisplayName: actorDisplayName,
			ParticipantID:      &participantID,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to record play event")
		}

		if play.CreatedBy != nil && play.MaxPlayers != nil {
			if err := store.UpdatePlaySlotsLeft(ctx, input.ID); err != nil {
				return nil, huma.Error500InternalServerError("failed to update slots_left")
			}
		}
		if participant.Status == model.ParticipantConfirmed || participant.Status == model.ParticipantAdded {
			if hostUserIDs, err := store.ListPlayHostUserIDsByPlay(ctx, input.ID); err == nil {
				_ = notifications.NotifyHostsPlayerLeft(ctx, notifier, notifications.PlaySnapshotFromDB(play), hostUserIDs, user.ID, user.DisplayName)
			}
		}

		return &struct{}{}, nil
	})
}
