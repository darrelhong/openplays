package pipeline

import (
	"context"
	"errors"
	"testing"
)

func TestUpsertStep_Success(t *testing.T) {
	store := &fakePlayStore{}
	step := NewUpsertStep(store)
	pc := makePC()

	if err := step.Process(context.Background(), pc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.upserted) != 1 {
		t.Errorf("expected 1 upsert, got %d", len(store.upserted))
	}
}

func TestUpsertStep_Error(t *testing.T) {
	store := &fakePlayStore{err: errors.New("db error")}
	step := NewUpsertStep(store)
	pc := makePC()

	err := step.Process(context.Background(), pc)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "db error" {
		t.Errorf("expected 'db error', got %v", err)
	}
}
