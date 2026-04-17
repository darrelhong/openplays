package pipeline

import (
	"context"
	"fmt"
	"time"
)

const maxPlayDuration = 5 * time.Hour

// ValidateStep checks that the play params are sane before insertion.
type ValidateStep struct{}

func (s *ValidateStep) Name() string { return "validate" }

func (s *ValidateStep) Process(_ context.Context, pc *PlayContext) error {
	p := &pc.Params

	// Duration check
	if !p.StartsAt.IsZero() && !p.EndsAt.IsZero() {
		duration := p.EndsAt.Sub(p.StartsAt)
		if duration > maxPlayDuration {
			return fmt.Errorf("%w: duration %s exceeds max %s (%s to %s)",
				ErrSkip, duration.Round(time.Minute), maxPlayDuration,
				p.StartsAt.Format("15:04"), p.EndsAt.Format("15:04"))
		}
		if duration <= 0 {
			return fmt.Errorf("%w: invalid duration %s (ends before it starts)", ErrSkip, duration)
		}
	}

	return nil
}
