package plays

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

const playHistoryEventLimit int64 = 50

type playEventWriter interface {
	CreatePlayEvent(ctx context.Context, arg db.CreatePlayEventParams) (db.PlayEvent, error)
}

type playEventReader interface {
	ListHostVisiblePlayEvents(ctx context.Context, arg db.ListHostVisiblePlayEventsParams) ([]db.PlayEvent, error)
	ListParticipantVisiblePlayEvents(ctx context.Context, arg db.ListParticipantVisiblePlayEventsParams) ([]db.PlayEvent, error)
}

type userReader interface {
	GetUserByID(ctx context.Context, id string) (db.User, error)
}

type participantSubject struct {
	UserID        *string
	DisplayName   *string
	ParticipantID *int64
}

type playEventMetadata map[string]any

func recordPlayEvent(ctx context.Context, store playEventWriter, arg db.CreatePlayEventParams) error {
	_, err := store.CreatePlayEvent(ctx, arg)
	return err
}

func playEventActor(user *auth.User) (*string, *string) {
	if user == nil {
		return nil, nil
	}
	return &user.ID, cleanStringPtr(&user.DisplayName)
}

func subjectFromParticipant(ctx context.Context, store userReader, participant db.PlayParticipant) (participantSubject, error) {
	participantID := participant.ID
	subject := participantSubject{
		UserID:        participant.UserID,
		DisplayName:   cleanStringPtr(participant.GuestName),
		ParticipantID: &participantID,
	}
	if participant.UserID == nil {
		return subject, nil
	}

	user, err := store.GetUserByID(ctx, *participant.UserID)
	if err == sql.ErrNoRows {
		return subject, nil
	}
	if err != nil {
		return participantSubject{}, err
	}
	subject.DisplayName = cleanStringPtr(&user.DisplayName)
	return subject, nil
}

func metadataJSON(metadata playEventMetadata) (*string, error) {
	if len(metadata) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	value := string(data)
	return &value, nil
}

func playEventMetadataFromJSON(value *string) (model.Meta, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	var metadata model.Meta
	if err := json.Unmarshal([]byte(*value), &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

func visibleHistoryEvents(ctx context.Context, store playEventReader, playID, viewerState string, canManage bool) ([]PlayHistoryEventPublic, error) {
	var (
		rows []db.PlayEvent
		err  error
	)
	if canManage {
		rows, err = store.ListHostVisiblePlayEvents(ctx, db.ListHostVisiblePlayEventsParams{
			PlayID: playID,
			Limit:  playHistoryEventLimit,
		})
	} else if currentParticipantState(viewerState) {
		rows, err = store.ListParticipantVisiblePlayEvents(ctx, db.ListParticipantVisiblePlayEventsParams{
			PlayID: playID,
			Limit:  playHistoryEventLimit,
		})
	}
	if err != nil {
		return nil, err
	}
	return mapPlayHistoryEvents(rows, time.Now())
}

func currentParticipantState(viewerState string) bool {
	switch viewerState {
	case string(model.ParticipantConfirmed), string(model.ParticipantAdded),
		string(model.ParticipantWaitlisted), string(model.ParticipantRequested):
		return true
	default:
		return false
	}
}

func mapPlayHistoryEvents(rows []db.PlayEvent, now time.Time) ([]PlayHistoryEventPublic, error) {
	events := make([]PlayHistoryEventPublic, 0, len(rows))
	for _, row := range rows {
		metadata, err := playEventMetadataFromJSON(row.Metadata)
		if err != nil {
			return nil, fmt.Errorf("parse play event metadata: %w", err)
		}
		actorDisplayName := row.ActorDisplayName
		if redactPlayHistoryActor(row.EventType) {
			actorDisplayName = nil
		}
		events = append(events, PlayHistoryEventPublic{
			ID:                 row.ID,
			EventType:          row.EventType,
			Message:            playHistoryEventMessage(row.EventType, actorDisplayName, row.SubjectDisplayName),
			ActorDisplayName:   actorDisplayName,
			SubjectDisplayName: row.SubjectDisplayName,
			Metadata:           metadata,
			CreatedAt:          row.CreatedAt.Format(time.RFC3339),
			RelativeTime: formatPlayHistoryRelativeTime(row.CreatedAt, now, playHistoryRelativeTimeOptions{
				AddSuffix: true,
			}),
		})
	}
	return events, nil
}

func redactPlayHistoryActor(eventType model.PlayEventType) bool {
	switch eventType {
	case model.PlayEventParticipantAdded, model.PlayEventParticipantRemoved,
		model.PlayEventParticipantMovedToWaitlist:
		return true
	default:
		return false
	}
}

func playHistoryEventMessage(eventType model.PlayEventType, actorDisplayName, subjectDisplayName *string) string {
	subject := historyDisplayName(subjectDisplayName, historyDisplayName(actorDisplayName, "Someone"))
	actor := historyDisplayName(actorDisplayName, "Host")

	switch eventType {
	case model.PlayEventParticipantJoined:
		return fmt.Sprintf("%s joined the game", subject)
	case model.PlayEventParticipantJoinRequested:
		return fmt.Sprintf("%s requested to join", subject)
	case model.PlayEventParticipantAdded:
		return fmt.Sprintf("%s was added to the game", subject)
	case model.PlayEventParticipantConfirmed:
		return fmt.Sprintf("%s confirmed their spot", subject)
	case model.PlayEventParticipantMovedToWaitlist:
		return fmt.Sprintf("%s was moved to the waitlist", subject)
	case model.PlayEventParticipantLeftConfirmed, model.PlayEventParticipantLeftAdded:
		return fmt.Sprintf("%s left the game", subject)
	case model.PlayEventParticipantLeftWaitlist:
		return fmt.Sprintf("%s left the waitlist", subject)
	case model.PlayEventParticipantRequestWithdrawn:
		return fmt.Sprintf("%s withdrew their request", subject)
	case model.PlayEventParticipantRemoved:
		return fmt.Sprintf("%s was removed from the game", subject)
	case model.PlayEventCancelled:
		if actorDisplayName != nil {
			return fmt.Sprintf("%s cancelled the game", actor)
		}
		return "Game cancelled"
	case model.PlayEventUpdated:
		return "Game updated"
	default:
		return "Game activity"
	}
}

func historyDisplayName(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}

func eventTypeForJoinStatus(status model.PlayParticipantStatus) model.PlayEventType {
	// A direct join reserves a spot as "added"; every other join lands in the
	// pending queue and is a request, on classic and require-waitlist plays alike
	if status == model.ParticipantAdded {
		return model.PlayEventParticipantJoined
	}
	return model.PlayEventParticipantJoinRequested
}

func eventTypeForLeaveStatus(status model.PlayParticipantStatus, requireWaitlist bool) model.PlayEventType {
	switch status {
	case model.ParticipantConfirmed:
		return model.PlayEventParticipantLeftConfirmed
	case model.ParticipantAdded:
		return model.PlayEventParticipantLeftAdded
	case model.ParticipantRequested:
		return model.PlayEventParticipantRequestWithdrawn
	case model.ParticipantWaitlisted:
		// Classic plays present their pending queue as requests; only
		// require-waitlist plays have a real (host-parked) waitlist
		if requireWaitlist {
			return model.PlayEventParticipantLeftWaitlist
		}
		return model.PlayEventParticipantRequestWithdrawn
	default:
		return model.PlayEventParticipantLeftWaitlist
	}
}
