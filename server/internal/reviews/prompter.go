package reviews

import (
	"context"
	"log/slog"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/notifications"
)

// promptScanInterval spaces the prompter's scans. Each scan is aligned one
// minute past a five-minute clock mark (5:01, 5:06, …): games end on whole
// hours, so their nudge lands a minute later while the database is only
// queried a dozen times an hour.
const promptScanInterval = 5 * time.Minute

// PrompterStore is the subset of db.Queries that the Prompter needs.
type PrompterStore interface {
	ListPlaysNeedingReviewPrompt(ctx context.Context) ([]db.ListPlaysNeedingReviewPromptRow, error)
	ListReviewEligibleUsersByPlay(ctx context.Context, playID string) ([]db.ListReviewEligibleUsersByPlayRow, error)
	MarkReviewPromptSent(ctx context.Context, arg db.MarkReviewPromptSentParams) (int64, error)
}

// Prompter notifies a play's eligible participants to review their
// co-players once the play has ended.
type Prompter struct {
	store  PrompterStore
	sender notifications.Sender
}

func NewPrompter(store PrompterStore, sender notifications.Sender) *Prompter {
	return &Prompter{store: store, sender: sender}
}

// Run ticks until the context is cancelled: once immediately so prompts
// missed across a deploy go out right away, then at every aligned mark
// (5:01, 5:06, …). Ticks never overlap — the next timer is only armed after
// the previous pass finishes, and a slow pass just skips to the next future
// mark.
func (p *Prompter) Run(ctx context.Context) {
	p.Tick(ctx)

	for {
		timer := time.NewTimer(time.Until(NextTickAfter(time.Now())))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			p.Tick(ctx)
		}
	}
}

// NextTickAfter returns the first instant strictly after now that lies one
// minute past a five-minute clock mark: …, 5:01, 5:06, 5:11, …
func NextTickAfter(now time.Time) time.Time {
	tick := now.Truncate(promptScanInterval).Add(time.Minute)
	for !tick.After(now) {
		tick = tick.Add(promptScanInterval)
	}
	return tick
}

// Tick runs a single scan-and-notify pass. Plays are handled sequentially:
// the slow part (web push delivery) is already dispatched async inside the
// sender, so each play job is only a few fast SQLite writes. Parallelize
// per-play if Notify ever grows a synchronous slow path.
func (p *Prompter) Tick(ctx context.Context) {
	plays, err := p.store.ListPlaysNeedingReviewPrompt(ctx)
	if err != nil {
		slog.Error("review prompter: list candidate plays", "error", err)
		return
	}

	for _, play := range plays {
		p.promptPlay(ctx, play)
	}
}

// promptPlay nudges every eligible participant of one ended play. Safe to
// run twice for the same play (rescans, multiple instances): the marker
// insert is the atomic gate, so at most one runner ever notifies a given
// (play, user).
func (p *Prompter) promptPlay(ctx context.Context, play db.ListPlaysNeedingReviewPromptRow) {
	members, err := p.store.ListReviewEligibleUsersByPlay(ctx, play.ID)
	if err != nil {
		slog.Error("review prompter: list eligible users", "play_id", play.ID, "error", err)
		return
	}
	// A lone rostered player has no co-players to review
	if len(members) < 2 {
		return
	}

	snapshot := notifications.PlaySnapshot{ID: play.ID, Name: play.Name, VenueName: play.VenueName}
	for _, member := range members {
		// Marking BEFORE notifying makes the prompt at-most-once: a crash
		// in between loses one nudge instead of ever double-sending
		inserted, err := p.store.MarkReviewPromptSent(ctx, db.MarkReviewPromptSentParams{
			PlayID: play.ID,
			UserID: member.ID,
		})
		if err != nil {
			slog.Error("review prompter: mark prompt sent", "play_id", play.ID, "user_id", member.ID, "error", err)
			continue
		}
		if inserted == 0 {
			continue // already prompted on an earlier pass
		}
		if err := notifications.NotifyReviewPrompt(ctx, p.sender, snapshot, member.ID); err != nil {
			slog.Error("review prompter: notify", "play_id", play.ID, "user_id", member.ID, "error", err)
		}
	}
}
