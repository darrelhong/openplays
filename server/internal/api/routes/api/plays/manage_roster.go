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
	GetPlayHost(ctx context.Context, arg db.GetPlayHostParams) (db.PlayHost, error)
	GetPlayParticipantByID(ctx context.Context, id int64) (db.PlayParticipant, error)
	GetUserByID(ctx context.Context, id string) (db.User, error)
	CountReservedPlayParticipants(ctx context.Context, playID string) (int64, error)
	UpdatePlayParticipantStatus(ctx context.Context, arg db.UpdatePlayParticipantStatusParams) (db.PlayParticipant, error)
	DeletePlayParticipant(ctx context.Context, id int64) error
	UpdatePlaySlotsLeft(ctx context.Context, id string) error
	CreatePlayEvent(ctx context.Context, arg db.CreatePlayEventParams) (db.PlayEvent, error)
}

func RegisterHostRosterManagement(api huma.API, store HostRosterStore, authMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "accept-play-participant",
		Summary:     "Add a waitlisted participant",
		Description: "Move a waitlisted participant into an added state pending player confirmation. Requires the play host and an open slot.",
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

		reservedCount, err := store.CountReservedPlayParticipants(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count participants")
		}
		if reservedCount >= *play.MaxPlayers {
			return nil, huma.Error409Conflict("play roster is full")
		}

		subject, err := subjectFromParticipant(ctx, store, participant)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get participant user")
		}

		updated, err := store.UpdatePlayParticipantStatus(ctx, db.UpdatePlayParticipantStatusParams{
			ID:     participant.ID,
			Status: model.ParticipantAdded,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to add participant")
		}

		if err := store.UpdatePlaySlotsLeft(ctx, input.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to update slots_left")
		}
		actorUserID, actorDisplayName := playEventActor(authmw.UserFromContext(ctx))
		metadata, err := metadataJSON(playEventMetadata{
			"from_status": string(model.ParticipantWaitlisted),
			"to_status":   string(model.ParticipantAdded),
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to record play event")
		}
		if err := recordPlayEvent(ctx, store, db.CreatePlayEventParams{
			PlayID:             input.ID,
			EventType:          model.PlayEventParticipantAdded,
			ActorUserID:        actorUserID,
			ActorDisplayName:   actorDisplayName,
			SubjectUserID:      subject.UserID,
			SubjectDisplayName: subject.DisplayName,
			ParticipantID:      subject.ParticipantID,
			Metadata:           metadata,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to record play event")
		}

		slots := deriveSlotsLeft(*play.MaxPlayers, reservedCount+1)
		out := &HostRosterOutput{}
		out.Body.Status = updated.Status
		out.Body.SlotsLeft = &slots
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "remove-play-participant",
		Summary:     "Remove a play participant",
		Description: "Remove a participant from the confirmed roster, added list, or waitlist. Requires the play host.",
		Method:      http.MethodDelete,
		Path:        "/{id}/participants/{participantID}",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *RemoveParticipantInput) (*struct{}, error) {
		play, participant, err := loadHostRosterTarget(ctx, store, input.ID, input.ParticipantID)
		if err != nil {
			return nil, err
		}

		if participant.UserID != nil {
			if ok, err := isPlayHost(ctx, store, play.ID, *participant.UserID); err != nil {
				return nil, err
			} else if ok {
				return nil, huma.Error409Conflict("host cannot be removed from roster")
			}
		}

		subject, err := subjectFromParticipant(ctx, store, participant)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get participant user")
		}

		if err := store.DeletePlayParticipant(ctx, participant.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to remove participant")
		}
		actorUserID, actorDisplayName := playEventActor(authmw.UserFromContext(ctx))
		metadata, err := metadataJSON(playEventMetadata{
			"from_status": string(participant.Status),
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to record play event")
		}
		if err := recordPlayEvent(ctx, store, db.CreatePlayEventParams{
			PlayID:             input.ID,
			EventType:          model.PlayEventParticipantRemoved,
			ActorUserID:        actorUserID,
			ActorDisplayName:   actorDisplayName,
			SubjectUserID:      subject.UserID,
			SubjectDisplayName: subject.DisplayName,
			ParticipantID:      subject.ParticipantID,
			Metadata:           metadata,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to record play event")
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
	if play.CancelledAt != nil {
		return db.GetPlayByIDRow{}, db.PlayParticipant{}, huma.Error409Conflict("play is cancelled")
	}
	if err := requirePlayHost(ctx, store, playID, user.ID); err != nil {
		return db.GetPlayByIDRow{}, db.PlayParticipant{}, err
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
