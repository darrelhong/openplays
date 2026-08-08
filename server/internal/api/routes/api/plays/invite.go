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

type InviteParticipantInput struct {
	ID   string `path:"id" doc:"Play ID"`
	Body struct {
		UserID string `json:"user_id" required:"true" doc:"User ID to invite"`
	}
}

type InviteParticipantOutput struct {
	Body struct {
		Status model.PlayParticipantStatus `json:"status"`
	}
}

type InviteParticipantStore interface {
	GetPlayByID(ctx context.Context, id string) (db.GetPlayByIDRow, error)
	GetPlayHost(ctx context.Context, arg db.GetPlayHostParams) (db.PlayHost, error)
	GetUserByID(ctx context.Context, id string) (db.User, error)
	IsBlocked(ctx context.Context, arg db.IsBlockedParams) (bool, error)
	GetPlayParticipantByPlayAndUser(ctx context.Context, arg db.GetPlayParticipantByPlayAndUserParams) (db.PlayParticipant, error)
	CountInvitedPlayParticipants(ctx context.Context, playID string) (int64, error)
	CreatePlayParticipant(ctx context.Context, arg db.CreatePlayParticipantParams) (db.PlayParticipant, error)
	CreatePlayEvent(ctx context.Context, arg db.CreatePlayEventParams) (db.PlayEvent, error)
}

func RegisterInviteParticipant(api huma.API, store InviteParticipantStore, authMiddleware func(huma.Context, func(huma.Context)), notifier notifications.Sender) {
	huma.Register(api, huma.Operation{
		OperationID: "invite-play-participant",
		Summary:     "Invite a player to a play",
		Description: "Invite an active registered user to a user-created play. Requires the play host.",
		Method:      http.MethodPost,
		Path:        "/{id}/invites",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{authMiddleware},
	}, func(ctx context.Context, input *InviteParticipantInput) (*InviteParticipantOutput, error) {
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
			return nil, huma.Error422UnprocessableEntity("cannot manage imported plays")
		}
		if play.CancelledAt != nil {
			return nil, huma.Error409Conflict("play is cancelled")
		}
		if err := requireRosterOpen(play); err != nil {
			return nil, err
		}
		if err := requirePlayHost(ctx, store, input.ID, user.ID); err != nil {
			return nil, err
		}

		if input.Body.UserID == user.ID {
			return nil, huma.Error422UnprocessableEntity("cannot invite yourself")
		}

		invitee, err := store.GetUserByID(ctx, input.Body.UserID)
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("invitee not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get invitee")
		}
		if invitee.Status != "active" {
			return nil, huma.Error422UnprocessableEntity("invitee is not active")
		}

		blocked, err := store.IsBlocked(ctx, db.IsBlockedParams{
			BlockerID:   user.ID,
			BlockedID:   invitee.ID,
			BlockerID_2: invitee.ID,
			BlockedID_2: user.ID,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to check user blocks")
		}
		if blocked {
			return nil, huma.Error403Forbidden("cannot invite a blocked user")
		}

		inviteeID := invitee.ID
		_, err = store.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{
			PlayID: input.ID,
			UserID: &inviteeID,
		})
		if err == nil {
			return nil, huma.Error409Conflict("user is already on the roster")
		}
		if err != sql.ErrNoRows {
			return nil, huma.Error500InternalServerError("failed to check roster")
		}

		if play.MaxPlayers == nil {
			return nil, huma.Error500InternalServerError("play is missing max_players")
		}
		invitedCount, err := store.CountInvitedPlayParticipants(ctx, input.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count invitations")
		}
		if invitedCount >= *play.MaxPlayers {
			return nil, huma.Error409Conflict("play has reached its invitation limit")
		}

		sportsProfile, err := model.ParseSportsProfile(invitee.SportsProfile)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to parse invitee profile")
		}
		ratingCode, ratingOrd := userRating(play.Sport, &auth.User{SportsProfile: sportsProfile})
		participant, err := store.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
			PlayID:          input.ID,
			UserID:          &inviteeID,
			RatingCode:      ratingCode,
			RatingOrd:       ratingOrd,
			Status:          model.ParticipantInvited,
			InvitedByUserID: &user.ID,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create invitation")
		}

		actorUserID, actorDisplayName := playEventActor(user)
		participantID := participant.ID
		inviteeDisplayName := cleanStringPtr(&invitee.DisplayName)
		if err := recordPlayEvent(ctx, store, db.CreatePlayEventParams{
			PlayID:             input.ID,
			EventType:          model.PlayEventParticipantInvited,
			ActorUserID:        actorUserID,
			ActorDisplayName:   actorDisplayName,
			SubjectUserID:      &inviteeID,
			SubjectDisplayName: inviteeDisplayName,
			ParticipantID:      &participantID,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to record play event")
		}

		_ = notifications.NotifyPlayerInvited(ctx, notifier, notifications.PlaySnapshotFromDB(play), inviteeID, user.DisplayName)

		out := &InviteParticipantOutput{}
		out.Body.Status = model.ParticipantInvited
		return out, nil
	})
}
