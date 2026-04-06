package listener

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/listener/parser"
	"openplays/server/internal/model"
	"openplays/server/internal/venue"
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
	UpsertPlay(ctx context.Context, arg db.UpsertPlayParams) (db.Play, error)
}

// Parser is the subset of parser.Pipeline that the Worker needs.
type Parser interface {
	Parse(ctx context.Context, input parser.MessageInput) ([]model.ParsedPlayCandidate, error)
}

// Worker processes raw messages from the job queue asynchronously.
type Worker struct {
	store    WorkerStore
	parser   Parser
	venue    *venue.Resolver // nil if venue resolution is disabled
	notify   chan struct{}
	timezone string
}

// NewWorker creates a new worker.
func NewWorker(store WorkerStore, parser Parser, venue *venue.Resolver, timezone string) *Worker {
	return &Worker{
		store:    store,
		parser:   parser,
		venue:    venue,
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
	log.Println("worker: started")

	retryTicker := time.NewTicker(retryInterval)
	defer retryTicker.Stop()

	// Process any leftover pending/failed jobs from previous runs
	w.processPending(ctx)
	w.processRetries(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("worker: shutting down")
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
			log.Printf("worker: error getting pending job: %v", err)
			return
		}
		w.processJob(ctx, job)
	}
}

// processRetries picks up failed jobs whose next_retry_at has passed.
func (w *Worker) processRetries(ctx context.Context) {
	log.Println("worker: checking for retry jobs")
	for {
		if ctx.Err() != nil {
			return
		}
		job, err := w.store.GetRetryJob(ctx)
		if err == sql.ErrNoRows {
			log.Println("worker: no retry jobs found")
			return
		}
		if err != nil {
			log.Printf("worker: error getting retry job: %v", err)
			return
		}
		w.processJob(ctx, job)
	}
}

func (w *Worker) processJob(ctx context.Context, job db.RawMessage) {
	log.Printf("worker: processing message #%d from %s (attempt %d)",
		job.ID, job.SenderUsername, job.RetryCount+1)

	if err := w.store.MarkProcessing(ctx, job.ID); err != nil {
		log.Printf("worker: error marking processing: %v", err)
		return
	}

	// Build parser input
	input := parser.MessageInput{
		Text:       job.MessageText,
		SenderName: job.SenderUsername,
		Timestamp:  job.MessageTime,
		Timezone:   w.timezone,
		Source:     job.Source,
	}

	// Call LLM
	parseCtx, cancel := context.WithTimeout(ctx, 500*time.Second)
	candidates, err := w.parser.Parse(parseCtx, input)
	cancel()

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
		log.Printf("worker: error marking done: %v", err)
		return
	}

	// Convert candidates to plays and insert
	for i, c := range candidates {
		var rv *parser.ResolvedVenue
		if w.venue != nil {
			if resolved := w.venue.Resolve(ctx, c.Venue); resolved != nil {
				rv = &parser.ResolvedVenue{
					PostalCode: resolved.PostalCode,
					Name:       resolved.Name,
				}
			}
		}
		params := parser.ToUpsertPlayParams(&c, input, rv)
		_, err := w.store.UpsertPlay(ctx, params)
		if err != nil {
			log.Printf("worker: error inserting play %d/%d for message #%d: %v",
				i+1, len(candidates), job.ID, err)
		}
	}

	log.Printf("worker: message #%d done — %d play(s) extracted", job.ID, len(candidates))
}

func (w *Worker) handleFailure(ctx context.Context, job db.RawMessage, err error) {
	errStr := err.Error()
	retryIdx := int(job.RetryCount)
	if retryIdx >= len(backoffDurations) {
		retryIdx = len(backoffDurations) - 1
	}
	nextRetry := time.Now().UTC().Add(backoffDurations[retryIdx])

	log.Printf("worker: message #%d failed (attempt %d): %v — next retry at %s",
		job.ID, job.RetryCount+1, err, nextRetry.Format("15:04:05"))

	if markErr := w.store.MarkFailed(ctx, db.MarkFailedParams{
		NextRetryAt: &nextRetry,
		LastError:   &errStr,
		ID:          job.ID,
	}); markErr != nil {
		log.Printf("worker: error marking failed: %v", markErr)
	}
}
