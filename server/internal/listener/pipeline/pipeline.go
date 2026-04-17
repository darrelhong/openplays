// Package pipeline processes raw messages into persisted plays.
// It orchestrates: LLM extraction → candidate conversion → step chain (validate, resolve, upsert).
package pipeline

import (
	"context"
	"errors"
	"log"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

// ErrSkip is returned by a step to skip this candidate without logging an error.
// Use this for expected conditions like validation failures.
var ErrSkip = errors.New("skip")

// MessageInput holds the raw message data from a source (Telegram, etc.).
type MessageInput struct {
	Text            string    // full raw message text
	SenderUsername  string    // Telegram @username (empty if user has none)
	SenderName      string    // display name (first+last or username fallback)
	Timestamp       time.Time // when the message was sent
	Timezone        string    // IANA timezone of the source, e.g. "Asia/Singapore"
	Source          string    // source of the message, e.g. "telegram"
	SourceMessageID *string   // platform message ID, e.g. Telegram message ID
	SourceGroup     *string   // platform group/channel, e.g. "sgbadmintontelecom"
}

// Extractor sends text to the LLM and returns parsed play candidates.
type Extractor interface {
	Extract(ctx context.Context, block string, referenceDate string, senderName string) ([]model.ParsedPlayCandidate, error)
}

// PlayContext carries the state for a single play candidate through the step chain.
type PlayContext struct {
	// Params is the upsert params being built up. Steps can modify this.
	Params db.UpsertPlayParams

	// MessageID is the raw_messages.id for logging.
	MessageID int64

	// Index is the candidate's position (0-based) within the message's candidates.
	Index int

	// Total is the number of candidates in this message.
	Total int
}

// Step processes a single play candidate. It can:
//   - Return nil to pass the candidate to the next step
//   - Return ErrSkip to silently skip this candidate
//   - Return any other error to log and skip this candidate
//   - Modify ctx.Params to enrich/transform the play
type Step interface {
	// Name returns a short identifier for logging (e.g. "validate", "resolve-venue").
	Name() string

	// Process runs this step on the play context.
	Process(ctx context.Context, pc *PlayContext) error
}

// Pipeline processes raw messages end-to-end: extract → convert → validate → resolve → upsert.
type Pipeline struct {
	extractor Extractor
	steps     []Step
}

// New creates a pipeline with an LLM extractor and candidate processing steps.
func New(extractor Extractor, steps ...Step) *Pipeline {
	return &Pipeline{
		extractor: extractor,
		steps:     steps,
	}
}

// Process takes a raw MessageInput, extracts candidates via the LLM, converts
// each to UpsertPlayParams, and runs the step chain. It returns the raw
// candidates (for the caller to store as LLM response) and the count of
// successfully inserted plays.
func (p *Pipeline) Process(ctx context.Context, input MessageInput, messageID int64) ([]model.ParsedPlayCandidate, int, error) {
	refDate := input.Timestamp.Format("2006-01-02")

	candidates, err := p.extractor.Extract(ctx, input.Text, refDate, input.SenderName)
	if err != nil {
		return nil, 0, err
	}

	inserted := 0
	for i, c := range candidates {
		params := ToUpsertPlayParams(&c, input)
		pc := &PlayContext{
			Params:    params,
			MessageID: messageID,
			Index:     i,
			Total:     len(candidates),
		}
		if err := p.runSteps(ctx, pc); err != nil {
			if !errors.Is(err, ErrSkip) {
				log.Printf("pipeline: error processing play %d/%d for message #%d: %v",
					i+1, len(candidates), messageID, err)
			}
			continue
		}
		inserted++
	}

	return candidates, inserted, nil
}

// runSteps executes all steps on the play context. Stops at the first error.
func (p *Pipeline) runSteps(ctx context.Context, pc *PlayContext) error {
	for _, step := range p.steps {
		if err := step.Process(ctx, pc); err != nil {
			if errors.Is(err, ErrSkip) {
				log.Printf("WARN: message #%d play %d/%d skipped at %s",
					pc.MessageID, pc.Index+1, pc.Total, step.Name())
				return ErrSkip
			}
			log.Printf("WARN: message #%d play %d/%d failed at %s: %v",
				pc.MessageID, pc.Index+1, pc.Total, step.Name(), err)
			return err
		}
	}
	return nil
}
