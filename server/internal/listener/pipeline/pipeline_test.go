package pipeline

import (
	"context"
	"errors"
	"testing"

	"openplays/server/internal/model"
)

func TestPipeline_AllStepsPass(t *testing.T) {
	s1 := &spyStep{name: "step1"}
	s2 := &spyStep{name: "step2"}
	ext := &fakeExtractor{candidates: []model.ParsedPlayCandidate{{}}}
	p := New(ext, s1, s2)

	candidates, inserted, err := p.Process(context.Background(), MessageInput{
		SenderName: "Test",
	}, 1)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if inserted != 1 {
		t.Errorf("expected 1 inserted, got %d", inserted)
	}
	if len(s1.processed) != 1 {
		t.Error("step1 should have been called")
	}
	if len(s2.processed) != 1 {
		t.Error("step2 should have been called")
	}
}

func TestPipeline_StopOnError(t *testing.T) {
	s1 := &spyStep{name: "step1", err: errors.New("boom")}
	s2 := &spyStep{name: "step2"}
	ext := &fakeExtractor{candidates: []model.ParsedPlayCandidate{{}}}
	p := New(ext, s1, s2)

	_, inserted, err := p.Process(context.Background(), MessageInput{
		SenderName: "Test",
	}, 1)

	if err != nil {
		t.Fatalf("Process should not return step errors, got %v", err)
	}
	if inserted != 0 {
		t.Errorf("expected 0 inserted (step failed), got %d", inserted)
	}
	if len(s2.processed) != 0 {
		t.Error("step2 should not have been called after step1 error")
	}
}

func TestPipeline_StopOnSkip(t *testing.T) {
	s1 := &spyStep{name: "step1", err: ErrSkip}
	s2 := &spyStep{name: "step2"}
	ext := &fakeExtractor{candidates: []model.ParsedPlayCandidate{{}}}
	p := New(ext, s1, s2)

	_, inserted, err := p.Process(context.Background(), MessageInput{
		SenderName: "Test",
	}, 1)

	if err != nil {
		t.Fatalf("Process should not return skip errors, got %v", err)
	}
	if inserted != 0 {
		t.Errorf("expected 0 inserted (skipped), got %d", inserted)
	}
	if len(s2.processed) != 0 {
		t.Error("step2 should not have been called after skip")
	}
}

func TestPipeline_EmptySteps(t *testing.T) {
	ext := &fakeExtractor{candidates: []model.ParsedPlayCandidate{{}}}
	p := New(ext)

	_, inserted, err := p.Process(context.Background(), MessageInput{
		SenderName: "Test",
	}, 1)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if inserted != 1 {
		t.Errorf("expected 1 inserted with no steps, got %d", inserted)
	}
}

func TestPipeline_ExtractorError(t *testing.T) {
	ext := &fakeExtractor{err: errors.New("LLM down")}
	p := New(ext)

	_, _, err := p.Process(context.Background(), MessageInput{
		SenderName: "Test",
	}, 1)

	if err == nil {
		t.Fatal("expected error from extractor")
	}
}

func TestPipeline_MultipleCandidates(t *testing.T) {
	s1 := &spyStep{name: "step1"}
	ext := &fakeExtractor{candidates: []model.ParsedPlayCandidate{{}, {}, {}}}
	p := New(ext, s1)

	candidates, inserted, err := p.Process(context.Background(), MessageInput{
		SenderName: "Test",
	}, 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 3 {
		t.Errorf("expected 3 candidates, got %d", len(candidates))
	}
	if inserted != 3 {
		t.Errorf("expected 3 inserted, got %d", inserted)
	}
	if len(s1.processed) != 3 {
		t.Errorf("step1 should have been called 3 times, got %d", len(s1.processed))
	}
}
