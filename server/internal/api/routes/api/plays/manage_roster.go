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

type AcceptParticipantInput struct {
	ID            string `path:"id" doc:"Play ID"`
	ParticipantID int64  `path:"participantID" doc:"Participant ID"`
}

type RemoveParticipantInput struct {
	ID            string `path:"id" doc:"Play ID"`
	ParticipantID int64  `path:"participantID" doc:"Participant ID"`
}

type HostRosterOutput struct {
	Body struct {
		Status    model.PlayParticipantStatus `json:"status"`
		SlotsLeft *int64                      `json:"slots_left,omitempty"`
	}
}

type HostRosterStore interface {
	GetPlayByID(ctx context.Context, id string) (db.GetPlayByIDRow, error)
	GetPlayParticipantByID(ctx context.Context, id int64) (db.PlayParticipant, error)
	CountConfirmedPlayParticipants(ctx context.Context, playID string) (int64, error)
	UpdatePlayParticipantStatus(ctx context.Context, arg db.UpdatePlayParticipantStatusParams) (db.PlayParticipant, error)
	DeletePlayParticipant(ctx context.Context, id int64) error
	UpdatePlaySlotsLeft(ctx context.Context, id string) error
}

func RegisterHostRosterManagement(api huma.API, store HostRosterStore, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "accept-play-participant",
		Summary:     "Accept a waitlisted participant",
		Description: "Move a waitlisted participant into the confirmed roster. Requires the play host and an open slot.",
		Method:      http.MethodPost,
		Path:        "/{id}/participants/{participantID}/accept",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *AcceptParticipantInput) (*HostRosterOutput, error) {
		play, participant, err := loadHostRosterTarget(ctx, store, input.ID, input.ParticipantID)
		if err != nil {
			return nil, err
		}

		if participant.Status != model.ParticipantWaitlisted {
			return nil, huma.Error409Conflict("participant is not waitlisted")
		}
		if play.MaxPlayers == nil {
			return nil, huma.Error500InternalServerError("play is missing max_players")
		}

		confirmedCount, err := store.CountConfirmedPlayParticipants(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count participants")
		}
		if confirmedCount >= *play.MaxPlayers {
			return nil, huma.Error409Conflict("play roster is full")
		}

		updated, err := store.UpdatePlayParticipantStatus(ctx, db.UpdatePlayParticipantStatusParams{
			ID:     participant.ID,
			Status: model.ParticipantConfirmed,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to accept participant")
		}

		if err := store.UpdatePlaySlotsLeft(ctx, input.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to update slots_left")
		}

		slots := deriveSlotsLeft(*play.MaxPlayers, confirmedCount+1)
		out := &HostRosterOutput{}
		out.Body.Status = updated.Status
		out.Body.SlotsLeft = &slots
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "remove-play-participant",
		Summary:     "Remove a play participant",
		Description: "Remove a participant from the confirmed roster or waitlist. Requires the play host.",
		Method:      http.MethodDelete,
		Path:        "/{id}/participants/{participantID}",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *RemoveParticipantInput) (*struct{}, error) {
		play, participant, err := loadHostRosterTarget(ctx, store, input.ID, input.ParticipantID)
		if err != nil {
			return nil, err
		}

		if participant.UserID != nil && play.CreatedBy != nil && *participant.UserID == *play.CreatedBy {
			return nil, huma.Error409Conflict("host cannot remove themselves")
		}

		if err := store.DeletePlayParticipant(ctx, participant.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to remove participant")
		}
		if play.MaxPlayers != nil {
			if err := store.UpdatePlaySlotsLeft(ctx, input.ID); err != nil {
				return nil, huma.Error500InternalServerError("failed to update slots_left")
			}
		}

		return &struct{}{}, nil
	})
}

func loadHostRosterTarget(ctx context.Context, store HostRosterStore, playID string, participantID int64) (db.GetPlayByIDRow, db.PlayParticipant, error) {
	user := authmw.UserFromContext(ctx)
	if user == nil {
		return db.GetPlayByIDRow{}, db.PlayParticipant{}, huma.Error401Unauthorized("not authenticated")
	}

	play, err := store.GetPlayByID(ctx, playID)
	if err == sql.ErrNoRows {
		return db.GetPlayByIDRow{}, db.PlayParticipant{}, huma.Error404NotFound("play not found")
	}
	if err != nil {
		return db.GetPlayByIDRow{}, db.PlayParticipant{}, huma.Error500InternalServerError("failed to get play")
	}
	if play.CreatedBy == nil {
		return db.GetPlayByIDRow{}, db.PlayParticipant{}, huma.Error422UnprocessableEntity("cannot manage imported plays")
	}
	if user.ID != *play.CreatedBy {
		return db.GetPlayByIDRow{}, db.PlayParticipant{}, huma.Error403Forbidden("only the host can manage this roster")
	}

	participant, err := store.GetPlayParticipantByID(ctx, participantID)
	if err == sql.ErrNoRows {
		return db.GetPlayByIDRow{}, db.PlayParticipant{}, huma.Error404NotFound("participant not found")
	}
	if err != nil {
		return db.GetPlayByIDRow{}, db.PlayParticipant{}, huma.Error500InternalServerError("failed to get participant")
	}
	if participant.PlayID != playID {
		return db.GetPlayByIDRow{}, db.PlayParticipant{}, huma.Error404NotFound("participant not found")
	}

	return play, participant, nil
}
