package pipeline

import (
	"context"
	"log"

	"openplays/server/internal/db"
)

// PlayStore is the subset of db.Queries needed by UpsertStep.
type PlayStore interface {
	UpsertPlay(ctx context.Context, arg db.UpsertPlayParams) (db.Play, error)
}

// UpsertStep inserts or updates the play in the database.
type UpsertStep struct {
	store PlayStore
}

func NewUpsertStep(store PlayStore) *UpsertStep {
	return &UpsertStep{store: store}
}

func (s *UpsertStep) Name() string { return "upsert" }

func (s *UpsertStep) Process(ctx context.Context, pc *PlayContext) error {
	_, err := s.store.UpsertPlay(ctx, pc.Params)
	if err != nil {
		log.Printf("worker: error inserting play %d/%d for message #%d: %v",
			pc.Index+1, pc.Total, pc.MessageID, err)
		return err
	}
	return nil
}
