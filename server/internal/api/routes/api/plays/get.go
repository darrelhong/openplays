package plays

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

type GetInput struct {
	ID string `path:"id" doc:"Play ID"`
}

type GetOutput struct {
	Body PlayPublic
}

func RegisterGet(api huma.API, queries *db.Queries, optionalAuthMiddleware func(huma.Context, func(huma.Context))) {
	huma.Register(api, huma.Operation{
		OperationID: "get-play",
		Summary:     "Get a play by ID",
		Method:      http.MethodGet,
		Path:        "/{id}",
		Tags:        []string{"Plays"},
		Middlewares: huma.Middlewares{optionalAuthMiddleware},
	}, func(ctx context.Context, input *GetInput) (*GetOutput, error) {
		r, err := queries.GetPlayByID(ctx, input.ID)
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("play not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get play", err)
		}

		createdAt, updatedAt := publicPlayTimestamps(r.CreatedBy, r.CreatedAt, r.UpdatedAt)
		item := PlayPublic{
			ID:                 r.ID,
			CreatedAt:          createdAt,
			UpdatedAt:          updatedAt,
			ListingType:        r.ListingType,
			Sport:              r.Sport,
			GameType:           r.GameType,
			HostName:           r.HostName,
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
		viewerState := "not_joined"
		canManage := false
		var viewerID *string
		if viewer := authmw.UserFromContext(ctx); viewer != nil {
			viewerID = &viewer.ID
			if ok, err := isPlayHost(ctx, queries, item.ID, viewer.ID); err != nil {
				return nil, err
			} else if ok {
				viewerState = "creator"
				canManage = true
			} else {
				participant, perr := queries.GetPlayParticipantByPlayAndUser(ctx, db.GetPlayParticipantByPlayAndUserParams{
					PlayID: item.ID,
					UserID: &viewer.ID,
				})
				if perr == nil {
					switch participant.Status {
					case model.ParticipantConfirmed:
						viewerState = "confirmed"
					case model.ParticipantWaitlisted:
						viewerState = "waitlisted"
					case model.ParticipantAdded:
						viewerState = "added"
					}
				} else if !errors.Is(perr, sql.ErrNoRows) {
					return nil, huma.Error500InternalServerError("failed to get viewer participation", perr)
				}
			}
		}
		item.ViewerState = &viewerState
		item.CanManage = &canManage
		if viewerID != nil {
			items := []PlayPublic{item}
			if err := hydrateFavouriteStates(ctx, queries, items, *viewerID); err != nil {
				return nil, huma.Error500InternalServerError("failed to get favourite state", err)
			}
			item = items[0]
		}

		confirmed, err := participantPreviewsForPlayByStatus(ctx, queries, item.ID, item.Sport, model.ParticipantConfirmed, true)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get confirmed participants", err)
		}
		item.ParticipantPreview = confirmed
		item.ConfirmedParticipants = confirmed

		confirmedCount := int64(len(confirmed))
		item.ConfirmedCount = &confirmedCount

		addedCount, err := queries.CountPlayParticipantsByStatus(ctx, db.CountPlayParticipantsByStatusParams{
			PlayID: item.ID,
			Status: model.ParticipantAdded,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count added participants", err)
		}
		item.AddedCount = &addedCount

		waitlistCount, err := queries.CountPlayParticipantsByStatus(ctx, db.CountPlayParticipantsByStatusParams{
			PlayID: item.ID,
			Status: model.ParticipantWaitlisted,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count waitlisted participants", err)
		}
		item.WaitlistCount = &waitlistCount

		if canManage || viewerState == "added" {
			added, err := participantPreviewsForPlayByStatus(ctx, queries, item.ID, item.Sport, model.ParticipantAdded, true)
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to get added participants", err)
			}
			if canManage {
				item.AddedParticipants = added
			} else if viewerID != nil {
				item.AddedParticipants = participantPreviewsForUser(added, *viewerID)
			}
		}

		if canManage || viewerState == "waitlisted" {
			waitlist, err := participantPreviewsForPlayByStatus(ctx, queries, item.ID, item.Sport, model.ParticipantWaitlisted, true)
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to get waitlisted participants", err)
			}
			if canManage {
				item.Waitlist = waitlist
			} else if viewerID != nil {
				item.Waitlist = participantPreviewsForUser(waitlist, *viewerID)
			}
		}

		if item.CreatedBy != nil && item.MaxPlayers != nil {
			slotsLeft := *item.MaxPlayers - confirmedCount - addedCount
			if slotsLeft < 0 {
				slotsLeft = 0
			}
			item.SlotsLeft = &slotsLeft
		}
		historyEvents, err := visibleHistoryEvents(ctx, queries, item.ID, viewerState, canManage)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get play history", err)
		}
		item.HistoryEvents = historyEvents

		return &GetOutput{Body: item}, nil
	})
}

func participantPreviewsForUser(participants []PlayParticipantPreviewPublic, userID string) []PlayParticipantPreviewPublic {
	out := make([]PlayParticipantPreviewPublic, 0, 1)
	for _, participant := range participants {
		if participant.UserID != nil && *participant.UserID == userID {
			out = append(out, participant)
		}
	}
	return out
}
