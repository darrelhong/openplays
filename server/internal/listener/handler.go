package listener

import (
	"context"
	"log"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/dedupe"
)

// HandleResult describes what happened when a message was processed.
type HandleResult int

const (
	HandleInserted HandleResult = iota // new message, inserted into queue
	HandleSkipped                      // duplicate, skipped
	HandleError                        // error occurred
)

// MessageStore is the subset of db.Queries that HandleMessage needs.
// Extracted as an interface for testability.
type MessageStore interface {
	GetRecentMessageTexts(ctx context.Context, createdAt time.Time) ([]db.GetRecentMessageTextsRow, error)
	InsertRawMessage(ctx context.Context, arg db.InsertRawMessageParams) (db.RawMessage, error)
}

// HandleMessage processes an incoming message: checks for duplicates against
// recent messages in the store, and inserts into the job queue if new.
func HandleMessage(ctx context.Context, store MessageStore, source, senderName, msgText string, msgTime time.Time, sourceMessageID, sourceGroup *string) (HandleResult, error) {
	// Dedupe check: compare against recent messages (last 24hrs)
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	recent, err := store.GetRecentMessageTexts(ctx, cutoff)
	if err != nil {
		log.Printf("error fetching recent messages for dedup: %v", err)
		// Continue anyway — better to have a duplicate than to lose a message
	} else {
		for _, r := range recent {
			if dedupe.IsSimilar(msgText, r.MessageText) {
				log.Printf("skipping duplicate message from %s (similar to message #%d)", senderName, r.ID)
				return HandleSkipped, nil
			}
		}
	}

	// Not a duplicate — insert into job queue
	contentHash := dedupe.ContentHash(msgText)
	_, err = store.InsertRawMessage(ctx, db.InsertRawMessageParams{
		Source:          source,
		SenderUsername:  senderName,
		MessageText:     msgText,
		MessageTime:     msgTime,
		ContentHash:     contentHash,
		Status:          "pending",
		SourceMessageID: sourceMessageID,
		SourceGroup:     sourceGroup,
	})
	if err != nil {
		return HandleError, err
	}

	return HandleInserted, nil
}
