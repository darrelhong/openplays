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

		item := PlayPublic{
			ID:                 r.ID,
			CreatedAt:          r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:          r.UpdatedAt.Format(time.RFC3339),
			ListingType:        r.ListingType,
			Sport:              r.Sport,
			GameType:           r.GameType,
			HostName:           r.HostName,
			StartsAt:           r.StartsAt.Format(time.RFC3339),
			EndsAt:             r.EndsAt.Format(time.RFC3339),
			Timezone:           r.Timezone,
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
		confirmed, err := participantPreviewsForPlayByStatus(ctx, queries, item.ID, item.Sport, model.ParticipantConfirmed, true)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get confirmed participants", err)
		}
		waitlist, err := participantPreviewsForPlayByStatus(ctx, queries, item.ID, item.Sport, model.ParticipantWaitlisted, true)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get waitlisted participants", err)
		}
		item.ParticipantPreview = confirmed
		item.ConfirmedParticipants = confirmed
		item.Waitlist = waitlist

		confirmedCount := int64(len(confirmed))
		waitlistCount := int64(len(waitlist))
		item.ConfirmedCount = &confirmedCount
		item.WaitlistCount = &waitlistCount
		if item.CreatedBy != nil && item.MaxPlayers != nil {
			slotsLeft := *item.MaxPlayers - confirmedCount
			if slotsLeft < 0 {
				slotsLeft = 0
			}
			item.SlotsLeft = &slotsLeft
		}

		viewerState := "not_joined"
		canManage := false
		if viewer := authmw.UserFromContext(ctx); viewer != nil {
			if item.CreatedBy != nil && viewer.ID == *item.CreatedBy {
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
					}
				} else if !errors.Is(perr, sql.ErrNoRows) {
					return nil, huma.Error500InternalServerError("failed to get viewer participation", perr)
				}
			}
		}
		item.ViewerState = &viewerState
		item.CanManage = &canManage

		return &GetOutput{Body: item}, nil
	})
}
