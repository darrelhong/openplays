package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/listener/parser"
)

// backoff schedule for retries (capped at 15 minutes)
var backoffDurations = []time.Duration{
	30 * time.Second,
	1 * time.Minute,
	2 * time.Minute,
	5 * time.Minute,
	15 * time.Minute,
}

// Worker processes raw messages from the job queue asynchronously.
type Worker struct {
	queries  *db.Queries
	pipeline *parser.Pipeline
	notify   chan struct{}
	timezone string
}

// New creates a new worker.
func New(queries *db.Queries, pipeline *parser.Pipeline, timezone string) *Worker {
	return &Worker{
		queries:  queries,
		pipeline: pipeline,
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

	// Process any leftover pending/failed jobs from previous runs
	w.processAll(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("worker: shutting down")
			return
		case <-w.notify:
			w.processAll(ctx)
		}
	}
}

func (w *Worker) processAll(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		if !w.processOne(ctx) {
			return
		}
	}
}

func (w *Worker) processOne(ctx context.Context) bool {
	job, err := w.queries.GetPendingJob(ctx)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		log.Printf("worker: error getting pending job: %v", err)
		return false
	}

	log.Printf("worker: processing message #%d from %s (attempt %d)",
		job.ID, job.SenderUsername, job.RetryCount+1)

	if err := w.queries.MarkProcessing(ctx, job.ID); err != nil {
		log.Printf("worker: error marking processing: %v", err)
		return false
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
	candidates, err := w.pipeline.Parse(parseCtx, input)
	cancel()

	if err != nil {
		w.handleFailure(ctx, job, err)
		return true
	}

	// Store LLM response
	llmJSON, _ := json.Marshal(candidates)
	llmStr := string(llmJSON)
	if err := w.queries.MarkDone(ctx, db.MarkDoneParams{
		LlmResponse: &llmStr,
		ID:          job.ID,
	}); err != nil {
		log.Printf("worker: error marking done: %v", err)
		return true
	}

	// Convert candidates to plays and insert
	for i, c := range candidates {
		params := parser.ToInsertPlayParams(&c, input)
		_, err := w.queries.InsertPlay(ctx, params)
		if err != nil {
			log.Printf("worker: error inserting play %d/%d for message #%d: %v",
				i+1, len(candidates), job.ID, err)
		}
	}

	log.Printf("worker: message #%d done — %d play(s) extracted", job.ID, len(candidates))
	return true
}

func (w *Worker) handleFailure(ctx context.Context, job db.RawMessage, err error) {
	errStr := err.Error()
	retryIdx := int(job.RetryCount)
	if retryIdx >= len(backoffDurations) {
		retryIdx = len(backoffDurations) - 1
	}
	nextRetry := time.Now().Add(backoffDurations[retryIdx])

	log.Printf("worker: message #%d failed (attempt %d): %v — next retry at %s",
		job.ID, job.RetryCount+1, err, nextRetry.Format("15:04:05"))

	if markErr := w.queries.MarkFailed(ctx, db.MarkFailedParams{
		NextRetryAt: &nextRetry,
		LastError:   &errStr,
		ID:          job.ID,
	}); markErr != nil {
		log.Printf("worker: error marking failed: %v", markErr)
	}
}
