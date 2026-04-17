package pipeline

import (
	"context"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

// --- Shared test helpers and fakes ---

type spyStep struct {
	name      string
	err       error
	processed []*PlayContext
}

func (s *spyStep) Name() string { return s.name }
func (s *spyStep) Process(_ context.Context, pc *PlayContext) error {
	s.processed = append(s.processed, pc)
	return s.err
}

type fakeExtractor struct {
	candidates []model.ParsedPlayCandidate
	err        error
}

func (f *fakeExtractor) Extract(_ context.Context, _ string, _ string, _ string) ([]model.ParsedPlayCandidate, error) {
	return f.candidates, f.err
}

type fakeVenueResolver struct {
	result *VenueResolved
}

func (f *fakeVenueResolver) Resolve(_ context.Context, _ *string) *VenueResolved {
	return f.result
}

type fakePlayStore struct {
	upserted []db.UpsertPlayParams
	err      error
}

func (f *fakePlayStore) UpsertPlay(_ context.Context, arg db.UpsertPlayParams) (db.Play, error) {
	f.upserted = append(f.upserted, arg)
	return db.Play{}, f.err
}

func makePC() *PlayContext {
	return &PlayContext{
		Params: db.UpsertPlayParams{
			StartsAt: time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC),
			EndsAt:   time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC),
			Venue:    "Test Venue",
		},
		MessageID: 1,
		Index:     0,
		Total:     1,
	}
}
