package listener

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/listener/pipeline"
)

// backoff schedule for retries (capped at 15 minutes)
var backoffDurations = []time.Duration{
	30 * time.Second,
	1 * time.Minute,
	2 * time.Minute,
	5 * time.Minute,
	15 * time.Minute,
}

// retryInterval is how often the worker checks for failed jobs due for retry.
const retryInterval = 5 * time.Minute

// WorkerStore is the subset of db.Queries that the Worker needs.
type WorkerStore interface {
	GetPendingJob(ctx context.Context) (db.RawMessage, error)
	GetRetryJob(ctx context.Context) (db.RawMessage, error)
	MarkProcessing(ctx context.Context, id int64) error
	MarkDone(ctx context.Context, arg db.MarkDoneParams) error
	MarkFailed(ctx context.Context, arg db.MarkFailedParams) error
}

// Worker processes raw messages from the job queue asynchronously.
type Worker struct {
	store    WorkerStore
	pipeline *pipeline.Pipeline
	notify   chan struct{}
	timezone string
}

// NewWorker creates a new worker with a unified pipeline for processing candidates.
func NewWorker(store WorkerStore, p *pipeline.Pipeline, timezone string) *Worker {
	return &Worker{
		store:    store,
		pipeline: p,
		notify:   make(chan struct{}, 1),
		timezone: timezone,
	}
}

// Notify signals the worker that new work is available.
func (w *Worker) Notify() {
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

// Run starts the worker loop. Blocks until ctx is cancelled.
// Call this in a goroutine.
func (w *Worker) Run(ctx context.Context) {
	slog.Info("worker started")

	retryTicker := time.NewTicker(retryInterval)
	defer retryTicker.Stop()

	// Process any leftover pending/failed jobs from previous runs
	w.processPending(ctx)
	w.processRetries(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker shutting down")
			return
		case <-w.notify:
			w.processPending(ctx)
		case <-retryTicker.C:
			w.processRetries(ctx)
		}
	}
}

// processPending drains all pending (new) jobs.
func (w *Worker) processPending(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		job, err := w.store.GetPendingJob(ctx)
		if err == sql.ErrNoRows {
			return
		}
		if err != nil {
			slog.Error("error getting pending job", "error", err)
			return
		}
		w.processJob(ctx, job)
	}
}

// processRetries picks up failed jobs whose next_retry_at has passed.
func (w *Worker) processRetries(ctx context.Context) {
	slog.Info("checking for retry jobs")
	for {
		if ctx.Err() != nil {
			return
		}
		job, err := w.store.GetRetryJob(ctx)
		if err == sql.ErrNoRows {
			slog.Info("no retry jobs found")
			return
		}
		if err != nil {
			slog.Error("error getting retry job", "error", err)
			return
		}
		w.processJob(ctx, job)
	}
}

func (w *Worker) processJob(ctx context.Context, job db.RawMessage) {
	slog.Info("processing message", "message_id", job.ID, "sender", job.SenderUsername, "attempt", job.RetryCount+1)

	if err := w.store.MarkProcessing(ctx, job.ID); err != nil {
		slog.Error("error marking processing", "message_id", job.ID, "error", err)
		return
	}

	// Build pipeline input
	input := pipeline.MessageInput{
		Text:            job.MessageText,
		SenderUsername:  job.SenderUsername,
		SenderName:      job.SenderName,
		Timestamp:       job.MessageTime,
		Timezone:        w.timezone,
		Source:          job.Source,
		SourceMessageID: job.SourceMessageID,
		SourceGroup:     job.SourceGroup,
	}

	// Run the unified pipeline: extract → convert → validate → resolve → upsert
	candidates, inserted, err := w.pipeline.Process(ctx, input, job.ID)
	if err != nil {
		w.handleFailure(ctx, job, err)
		return
	}

	// Store LLM response
	llmJSON, _ := json.Marshal(candidates)
	llmStr := string(llmJSON)
	if err := w.store.MarkDone(ctx, db.MarkDoneParams{
		LlmResponse: &llmStr,
		ID:          job.ID,
	}); err != nil {
		slog.Error("error marking done", "message_id", job.ID, "error", err)
		return
	}

	slog.Info("message done", "message_id", job.ID, "plays_inserted", inserted)
}

func (w *Worker) handleFailure(ctx context.Context, job db.RawMessage, err error) {
	errStr := err.Error()
	retryIdx := int(job.RetryCount)
	if retryIdx >= len(backoffDurations) {
		retryIdx = len(backoffDurations) - 1
	}
	nextRetry := time.Now().UTC().Add(backoffDurations[retryIdx])

	slog.Error("message failed", "message_id", job.ID, "attempt", job.RetryCount+1, "error", err, "next_retry_at", nextRetry.Format("15:04:05"))

	if markErr := w.store.MarkFailed(ctx, db.MarkFailedParams{
		NextRetryAt: &nextRetry,
		LastError:   &errStr,
		ID:          job.ID,
	}); markErr != nil {
		slog.Error("error marking failed", "message_id", job.ID, "error", markErr)
	}
}
