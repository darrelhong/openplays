package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestValidateStep(t *testing.T) {
	step := &ValidateStep{}
	base := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		starts  time.Time
		ends    time.Time
		wantErr bool
	}{
		{"2h session passes", base, base.Add(2 * time.Hour), false},
		{"exactly 5h passes", base, base.Add(5 * time.Hour), false},
		{"5h01m exceeds max duration", base, base.Add(5*time.Hour + 1*time.Minute), true},
		{"6h exceeds max duration", base, base.Add(6 * time.Hour), true},
		{"negative duration skipped", base, base.Add(-1 * time.Hour), true},
		{"zero duration skipped", base, base, true},
		{"zero start and end times passes", time.Time{}, time.Time{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := makePC()
			pc.Params.StartsAt = tt.starts
			pc.Params.EndsAt = tt.ends

			err := step.Process(context.Background(), pc)
			if tt.wantErr && !errors.Is(err, ErrSkip) {
				t.Errorf("expected ErrSkip, got %v", err)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}
