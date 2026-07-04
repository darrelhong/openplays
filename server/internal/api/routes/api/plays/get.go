package plays

import (
	"context"
	"database/sql"
	"net/http"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
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
			Name:               r.Name,
			Description:        r.Description,
			Visibility:         r.Visibility,
			RequireWaitlist:    r.RequireWaitlist,
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
			VenueGooglePlaceID: r.VenueGooglePlaceID,
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
		// One host lookup and one roster query cover viewer state, all counts,
		// and every preview list
		hostIDs, err := queries.ListPlayHostUserIDsByPlay(ctx, item.ID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get play hosts", err)
		}

		canManage := false
		var viewerID *string
		if viewer := authmw.UserFromContext(ctx); viewer != nil {
			viewerID = &viewer.ID
			canManage = slices.Contains(hostIDs, viewer.ID)
		}

		roster, err := rosterPreviewsForPlay(ctx, queries, item.ID, item.Sport, true, hostIDs, viewerID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get participants", err)
		}

		viewerState := "not_joined"
		if canManage {
			viewerState = "creator"
		} else if viewerID != nil {
			viewerState = viewerRosterState(roster)
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

		item.ParticipantPreview = roster.Confirmed
		item.ConfirmedParticipants = roster.Confirmed
		confirmedCount := int64(len(roster.Confirmed))
		item.ConfirmedCount = &confirmedCount
		addedCount := int64(len(roster.Added))
		item.AddedCount = &addedCount
		waitlistCount := int64(len(roster.Waitlisted))
		item.WaitlistCount = &waitlistCount
		requestedCount := int64(len(roster.Requested))
		item.RequestedCount = &requestedCount

		// Added players hold reserved spots, so like confirmed players they are
		// visible to everyone; is_viewer lets the UI scope self-serve actions
		item.AddedParticipants = roster.Added

		// Pending-queue identities are host-only; a pending viewer sees only
		// their own row
		if canManage {
			item.Waitlist = roster.Waitlisted
			item.Requests = roster.Requested
		} else if viewerID != nil {
			if viewerState == "waitlisted" {
				item.Waitlist = participantPreviewsForUser(roster.Waitlisted, *viewerID)
			}
			if viewerState == "requested" {
				item.Requests = participantPreviewsForUser(roster.Requested, *viewerID)
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

// viewerRosterState maps the viewer's marked roster row to their viewer_state.
func viewerRosterState(roster playRosterPreviews) string {
	groups := []struct {
		state    string
		previews []PlayParticipantPreviewPublic
	}{
		{"confirmed", roster.Confirmed},
		{"added", roster.Added},
		{"requested", roster.Requested},
		{"waitlisted", roster.Waitlisted},
	}
	for _, group := range groups {
		for _, preview := range group.previews {
			if preview.IsViewer {
				return group.state
			}
		}
	}
	return "not_joined"
}

func markViewerParticipant(participants []PlayParticipantPreviewPublic, userID string) {
	for i := range participants {
		if participants[i].UserID != nil && *participants[i].UserID == userID {
			participants[i].IsViewer = true
		}
	}
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
