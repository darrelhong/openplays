package listener

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/listener/parser"
	"openplays/server/internal/model"
)

// --- Spy implementations ---

// SpyWorkerStore is an in-memory fake that records calls and returns
// pre-configured jobs/errors.
type SpyWorkerStore struct {
	Calls []string

	// Pending jobs returned by GetPendingJob (consumed in order)
	PendingJobs []db.RawMessage
	// Retry jobs returned by GetRetryJob (consumed in order)
	RetryJobs []db.RawMessage

	// Collected state transitions
	ProcessingIDs []int64
	DoneParams    []db.MarkDoneParams
	FailedParams  []db.MarkFailedParams
	UpsertedPlays []db.UpsertPlayParams

	// Error injection
	MarkProcessingErr error
	MarkDoneErr       error
	MarkFailedErr     error
	UpsertPlayErr     error
}

func (s *SpyWorkerStore) GetPendingJob(ctx context.Context) (db.RawMessage, error) {
	s.Calls = append(s.Calls, "GetPendingJob")
	if len(s.PendingJobs) == 0 {
		return db.RawMessage{}, sql.ErrNoRows
	}
	job := s.PendingJobs[0]
	s.PendingJobs = s.PendingJobs[1:]
	return job, nil
}

func (s *SpyWorkerStore) GetRetryJob(ctx context.Context) (db.RawMessage, error) {
	s.Calls = append(s.Calls, "GetRetryJob")
	if len(s.RetryJobs) == 0 {
		return db.RawMessage{}, sql.ErrNoRows
	}
	job := s.RetryJobs[0]
	s.RetryJobs = s.RetryJobs[1:]
	return job, nil
}

func (s *SpyWorkerStore) MarkProcessing(ctx context.Context, id int64) error {
	s.Calls = append(s.Calls, "MarkProcessing")
	s.ProcessingIDs = append(s.ProcessingIDs, id)
	return s.MarkProcessingErr
}

func (s *SpyWorkerStore) MarkDone(ctx context.Context, arg db.MarkDoneParams) error {
	s.Calls = append(s.Calls, "MarkDone")
	s.DoneParams = append(s.DoneParams, arg)
	return s.MarkDoneErr
}

func (s *SpyWorkerStore) MarkFailed(ctx context.Context, arg db.MarkFailedParams) error {
	s.Calls = append(s.Calls, "MarkFailed")
	s.FailedParams = append(s.FailedParams, arg)
	return s.MarkFailedErr
}

func (s *SpyWorkerStore) UpsertPlay(ctx context.Context, arg db.UpsertPlayParams) (db.Play, error) {
	s.Calls = append(s.Calls, "UpsertPlay")
	s.UpsertedPlays = append(s.UpsertedPlays, arg)
	return db.Play{}, s.UpsertPlayErr
}

// SpyParser records calls and returns pre-configured results/errors.
type SpyParser struct {
	Calls      []string
	Candidates []model.ParsedPlayCandidate
	Err        error
}

func (s *SpyParser) Parse(ctx context.Context, input parser.MessageInput) ([]model.ParsedPlayCandidate, error) {
	s.Calls = append(s.Calls, "Parse")
	return s.Candidates, s.Err
}

// --- Helpers ---

func makeJob(id int64, text string, retryCount int64) db.RawMessage {
	return db.RawMessage{
		ID:             id,
		Source:         "telegram",
		SenderUsername: "test_user",
		MessageText:    text,
		MessageTime:    time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC),
		Status:         "pending",
		RetryCount:     retryCount,
	}
}

func assertWorkerCalls(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d calls %v, want %d calls %v", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("call[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

// --- Tests ---

func TestProcessPending_HappyPath(t *testing.T) {
	store := &SpyWorkerStore{
		PendingJobs: []db.RawMessage{makeJob(1, "looking for players", 0)},
	}
	p := &SpyParser{
		Candidates: []model.ParsedPlayCandidate{{}},
	}
	w := NewWorker(store, p, "Asia/Singapore")

	w.processPending(context.Background())

	assertWorkerCalls(t, store.Calls, []string{
		"GetPendingJob",
		"MarkProcessing",
		"MarkDone",
		"UpsertPlay",
		"GetPendingJob", // second call returns ErrNoRows, loop exits
	})
	assertWorkerCalls(t, p.Calls, []string{"Parse"})

	if len(store.ProcessingIDs) != 1 || store.ProcessingIDs[0] != 1 {
		t.Errorf("expected MarkProcessing for job #1, got %v", store.ProcessingIDs)
	}
	if len(store.DoneParams) != 1 || store.DoneParams[0].ID != 1 {
		t.Errorf("expected MarkDone for job #1, got %v", store.DoneParams)
	}
}

func TestProcessPending_NoPendingJobs(t *testing.T) {
	store := &SpyWorkerStore{}
	p := &SpyParser{}
	w := NewWorker(store, p, "Asia/Singapore")

	w.processPending(context.Background())

	assertWorkerCalls(t, store.Calls, []string{"GetPendingJob"})
	assertWorkerCalls(t, p.Calls, nil)
}

func TestProcessRetries_PicksUpFailedJobs(t *testing.T) {
	retryJob := makeJob(42, "retry me", 1)
	retryJob.Status = "failed"

	store := &SpyWorkerStore{
		RetryJobs: []db.RawMessage{retryJob},
	}
	p := &SpyParser{
		Candidates: []model.ParsedPlayCandidate{{}},
	}
	w := NewWorker(store, p, "Asia/Singapore")

	w.processRetries(context.Background())

	assertWorkerCalls(t, store.Calls, []string{
		"GetRetryJob",
		"MarkProcessing",
		"MarkDone",
		"UpsertPlay",
		"GetRetryJob", // second call returns ErrNoRows, loop exits
	})
	assertWorkerCalls(t, p.Calls, []string{"Parse"})

	if store.ProcessingIDs[0] != 42 {
		t.Errorf("expected MarkProcessing for job #42, got %v", store.ProcessingIDs)
	}
}

func TestProcessRetries_NoRetryJobs(t *testing.T) {
	store := &SpyWorkerStore{}
	p := &SpyParser{}
	w := NewWorker(store, p, "Asia/Singapore")

	w.processRetries(context.Background())

	assertWorkerCalls(t, store.Calls, []string{"GetRetryJob"})
	assertWorkerCalls(t, p.Calls, nil)
}

func TestProcessJob_ParseFailure_MarksFailedWithBackoff(t *testing.T) {
	job := makeJob(10, "bad message", 0)
	store := &SpyWorkerStore{}
	p := &SpyParser{Err: fmt.Errorf("LLM returned status 429: rate limited")}
	w := NewWorker(store, p, "Asia/Singapore")

	w.processJob(context.Background(), job)

	assertWorkerCalls(t, store.Calls, []string{"MarkProcessing", "MarkFailed"})

	if len(store.FailedParams) != 1 {
		t.Fatalf("expected 1 MarkFailed call, got %d", len(store.FailedParams))
	}
	fp := store.FailedParams[0]
	if fp.ID != 10 {
		t.Errorf("MarkFailed job ID = %d, want 10", fp.ID)
	}
	if fp.LastError == nil || *fp.LastError == "" {
		t.Error("expected LastError to be set")
	}
	if fp.NextRetryAt == nil {
		t.Fatal("expected NextRetryAt to be set")
	}
	// First failure (retry_count=0) should use 30s backoff
	if fp.NextRetryAt.Before(time.Now().UTC()) {
		t.Error("NextRetryAt should be in the future")
	}
}

func TestProcessJob_BackoffCapsAtMaxDuration(t *testing.T) {
	job := makeJob(10, "bad message", 99) // retry_count well past backoff slice length
	store := &SpyWorkerStore{}
	p := &SpyParser{Err: fmt.Errorf("LLM error")}
	w := NewWorker(store, p, "Asia/Singapore")

	before := time.Now().UTC()
	w.processJob(context.Background(), job)
	after := time.Now().UTC()

	fp := store.FailedParams[0]
	// Should cap at 15 minutes (last entry in backoffDurations)
	earliest := before.Add(15 * time.Minute)
	latest := after.Add(15*time.Minute + time.Second)
	if fp.NextRetryAt.Before(earliest) || fp.NextRetryAt.After(latest) {
		t.Errorf("NextRetryAt = %v, want between %v and %v", fp.NextRetryAt, earliest, latest)
	}
}

func TestProcessJob_NextRetryAt_IsUTC(t *testing.T) {
	job := makeJob(10, "bad message", 0)
	store := &SpyWorkerStore{}
	p := &SpyParser{Err: fmt.Errorf("LLM error")}
	w := NewWorker(store, p, "Asia/Singapore")

	w.processJob(context.Background(), job)

	fp := store.FailedParams[0]
	if fp.NextRetryAt.Location() != time.UTC {
		t.Errorf("NextRetryAt timezone = %v, want UTC", fp.NextRetryAt.Location())
	}

	// Verify the formatted string has no timezone offset suffix.
	// This is the actual bug: time.Now() in SGT produces "2026-04-04 18:53:33+08:00"
	// which SQLite compares as a string against CURRENT_TIMESTAMP ("2026-04-04 10:53:33").
	// The "+08:00" time looks later than the UTC time, so the retry is never picked up.
	formatted := fp.NextRetryAt.Format(time.RFC3339)
	if formatted[len(formatted)-1] != 'Z' {
		t.Errorf("NextRetryAt formatted = %s, want UTC suffix 'Z' (no +HH:MM offset)", formatted)
	}
}

func TestProcessJob_MultipleCandidates_InsertsAll(t *testing.T) {
	job := makeJob(5, "two plays", 0)
	store := &SpyWorkerStore{}
	p := &SpyParser{
		Candidates: []model.ParsedPlayCandidate{{}, {}},
	}
	w := NewWorker(store, p, "Asia/Singapore")

	w.processJob(context.Background(), job)

	assertWorkerCalls(t, store.Calls, []string{
		"MarkProcessing",
		"MarkDone",
		"UpsertPlay",
		"UpsertPlay",
	})
	if len(store.UpsertedPlays) != 2 {
		t.Errorf("expected 2 UpsertPlay calls, got %d", len(store.UpsertedPlays))
	}
}

func TestProcessJob_ZeroCandidates_StillMarksDone(t *testing.T) {
	job := makeJob(7, "no plays here", 0)
	store := &SpyWorkerStore{}
	p := &SpyParser{Candidates: nil}
	w := NewWorker(store, p, "Asia/Singapore")

	w.processJob(context.Background(), job)

	assertWorkerCalls(t, store.Calls, []string{"MarkProcessing", "MarkDone"})
	if len(store.UpsertedPlays) != 0 {
		t.Errorf("expected 0 UpsertPlay calls, got %d", len(store.UpsertedPlays))
	}
}

func TestPendingAndRetry_AreIndependent(t *testing.T) {
	pendingJob := makeJob(1, "new message", 0)
	retryJob := makeJob(2, "old failed message", 2)
	retryJob.Status = "failed"

	store := &SpyWorkerStore{
		PendingJobs: []db.RawMessage{pendingJob},
		RetryJobs:   []db.RawMessage{retryJob},
	}
	p := &SpyParser{Candidates: []model.ParsedPlayCandidate{{}}}
	w := NewWorker(store, p, "Asia/Singapore")

	// Process pending — should only touch pending queue
	w.processPending(context.Background())
	pendingCalls := make([]string, len(store.Calls))
	copy(pendingCalls, store.Calls)

	for _, call := range pendingCalls {
		if call == "GetRetryJob" {
			t.Error("processPending should not call GetRetryJob")
		}
	}

	// Reset and process retries — should only touch retry queue
	store.Calls = nil
	w.processRetries(context.Background())

	for _, call := range store.Calls {
		if call == "GetPendingJob" {
			t.Error("processRetries should not call GetPendingJob")
		}
	}
}

func TestRunStartup_ProcessesBothQueues(t *testing.T) {
	pendingJob := makeJob(1, "pending", 0)
	retryJob := makeJob(2, "retry", 1)

	store := &SpyWorkerStore{
		PendingJobs: []db.RawMessage{pendingJob},
		RetryJobs:   []db.RawMessage{retryJob},
	}
	p := &SpyParser{Candidates: []model.ParsedPlayCandidate{{}}}
	w := NewWorker(store, p, "Asia/Singapore")

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	// Give Run time to process startup jobs
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	// Verify both queues were checked
	var gotPending, gotRetry bool
	for _, call := range store.Calls {
		if call == "GetPendingJob" {
			gotPending = true
		}
		if call == "GetRetryJob" {
			gotRetry = true
		}
	}
	if !gotPending {
		t.Error("Run should call GetPendingJob on startup")
	}
	if !gotRetry {
		t.Error("Run should call GetRetryJob on startup")
	}

	// Both jobs should have been processed
	if len(store.ProcessingIDs) < 2 {
		t.Errorf("expected at least 2 jobs processed on startup, got %d", len(store.ProcessingIDs))
	}
}

func TestNotify_TriggersPendingProcessing(t *testing.T) {
	store := &SpyWorkerStore{}
	p := &SpyParser{Candidates: []model.ParsedPlayCandidate{{}}}
	w := NewWorker(store, p, "Asia/Singapore")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	// Wait for startup processing to finish
	time.Sleep(50 * time.Millisecond)
	store.Calls = nil

	// Add a job and notify
	store.PendingJobs = []db.RawMessage{makeJob(99, "new one", 0)}
	w.Notify()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	var gotPending bool
	for _, call := range store.Calls {
		if call == "GetPendingJob" {
			gotPending = true
		}
	}
	if !gotPending {
		t.Error("Notify should trigger GetPendingJob")
	}
}
