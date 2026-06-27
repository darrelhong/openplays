package plays

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/notifications"
)

type JoinInput struct {
	ID string `path:"id" doc:"Play ID"`
}

type JoinOutput struct {
	Body struct {
		Status    model.PlayParticipantStatus `json:"status"`
		SlotsLeft *int64                      `json:"slots_left,omitempty"`
	}
}

type JoinStore interface {
	GetPlayByID(ctx context.Context, id string) (db.GetPlayByIDRow, error)
	GetPlayParticipantByPlayAndUser(ctx context.Context, arg db.GetPlayParticipantByPlayAndUserParams) (db.PlayParticipant, error)
	CreatePlayParticipant(ctx context.Context, arg db.CreatePlayParticipantParams) (db.PlayParticipant, error)
	CountReservedPlayParticipants(ctx context.Context, playID string) (int64, error)
	ListPlayHostUserIDsByPlay(ctx context.Context, playID string) ([]string, error)
	UpdatePlaySlotsLeft(ctx context.Context, id string) error
	CreatePlayEvent(ctx context.Context, arg db.CreatePlayEventParams) (db.PlayEvent, error)
}

func RegisterJoin(api huma.API, store JoinStore, authMiddleware func(huma.Context, func(huma.Context)), notifier notifications.Sender) {
	huma.Register(api, huma.Operation{
		OperationID: "join-play",
		Summary:     "Join a play",
		Description: "Join a user-created play. Auto-confirms if rating matches and slots exist; otherwise waitlists.",
		Method:      http.MethodPost,
		Path:        "/{id}/join",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *JoinInput) (*JoinOutput, error) {
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
			return nil, huma.Error422UnprocessableEntity("cannot join imported plays")
		}
		if play.CancelledAt != nil {
			return nil, huma.Error409Conflict("play is cancelled")
		}
		if play.MaxPlayers == nil {
			return nil, huma.Error500InternalServerError("play is missing max_players")
		}

		existing, err := store.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{
			PlayID: input.ID,
			UserID: &user.ID,
		})
		if err != nil && err != sql.ErrNoRows {
			return nil, huma.Error500InternalServerError("failed to get participant")
		}
		if err == nil {
			if syncErr := store.UpdatePlaySlotsLeft(ctx, input.ID); syncErr != nil {
				return nil, huma.Error500InternalServerError("failed to update slots_left")
			}
			reservedCount, countErr := store.CountReservedPlayParticipants(ctx, input.ID)
			if countErr != nil {
				return nil, huma.Error500InternalServerError("failed to count participants")
			}
			slots := deriveSlotsLeft(*play.MaxPlayers, reservedCount)
			out := &JoinOutput{}
			out.Body.Status = existing.Status
			out.Body.SlotsLeft = &slots
			return out, nil
		}

		reservedCount, err := store.CountReservedPlayParticipants(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count participants")
		}

		status, ratingCode, ratingOrd := resolveJoinStatus(play, user, reservedCount)
		participant, err := store.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
			PlayID:     input.ID,
			UserID:     &user.ID,
			RatingCode: ratingCode,
			RatingOrd:  ratingOrd,
			Status:     status,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to add participant")
		}
		actorUserID, actorDisplayName := playEventActor(user)
		participantID := participant.ID
		if err := recordPlayEvent(ctx, store, db.CreatePlayEventParams{
			PlayID:             input.ID,
			EventType:          eventTypeForJoinStatus(status),
			ActorUserID:        actorUserID,
			ActorDisplayName:   actorDisplayName,
			SubjectUserID:      actorUserID,
			SubjectDisplayName: actorDisplayName,
			ParticipantID:      &participantID,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to record play event")
		}

		if err := store.UpdatePlaySlotsLeft(ctx, input.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to update slots_left")
		}
		if hostUserIDs, err := store.ListPlayHostUserIDsByPlay(ctx, input.ID); err == nil {
			playSnapshot := notifications.PlaySnapshotFromDB(play)
			if status == model.ParticipantWaitlisted {
				_ = notifications.NotifyHostWaitlistJoined(ctx, notifier, playSnapshot, hostUserIDs, user.DisplayName)
			} else if status == model.ParticipantConfirmed {
				_ = notifications.NotifyHostsPlayerJoined(ctx, notifier, playSnapshot, hostUserIDs, user.ID, user.DisplayName)
			}
		}

		finalReservedCount := reservedCount
		if status == model.ParticipantConfirmed {
			finalReservedCount++
		}
		slots := deriveSlotsLeft(*play.MaxPlayers, finalReservedCount)
		out := &JoinOutput{}
		out.Body.Status = status
		out.Body.SlotsLeft = &slots
		return out, nil
	})
}

func resolveJoinStatus(play db.GetPlayByIDRow, user *auth.User, reservedCount int64) (model.PlayParticipantStatus, *string, *int64) {
	ratingCode, ratingOrd := userRating(play.Sport, user)
	if reservedCount >= *play.MaxPlayers || ratingOrd == nil || !ratingMatches(play, *ratingOrd) {
		return model.ParticipantWaitlisted, ratingCode, ratingOrd
	}
	return model.ParticipantConfirmed, ratingCode, ratingOrd
}

func userRating(sport model.Sport, user *auth.User) (*string, *int64) {
	if user == nil || user.SportsProfile == nil {
		return nil, nil
	}
	ratingCode := user.SportsProfile.LevelFor(sport)
	if ratingCode == nil {
		return nil, nil
	}
	ord := model.LevelOrd(sport, *ratingCode)
	if ord == nil {
		return nil, nil
	}
	v := int64(*ord)
	return ratingCode, &v
}

func ratingMatches(play db.GetPlayByIDRow, ratingOrd int64) bool {
	if play.LevelMinOrd != nil && ratingOrd < *play.LevelMinOrd {
		return false
	}
	if play.LevelMaxOrd != nil && ratingOrd > *play.LevelMaxOrd {
		return false
	}
	return true
}

func deriveSlotsLeft(maxPlayers, confirmedCount int64) int64 {
	slots := maxPlayers - confirmedCount
	if slots < 0 {
		return 0
	}
	return slots
}
