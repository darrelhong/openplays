package pipeline

import (
	"encoding/json"
	"testing"
)

func TestBuildRequest_OmitsReasoningParamsWhenUnset(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cfg    LLMConfig
		want   []string // keys that MUST be present
		absent []string // keys that MUST be absent
	}{
		{
			// The default for gpt-6-luna: reasoning off, output unbounded.
			// Output is deliberately not capped, so the endpoint applies its own
			// limit rather than us guessing one and truncating a long
			// multi-listing reply mid-object.
			name:   "reasoning model, default config",
			cfg:    LLMConfig{Model: "gpt-6-luna", ReasoningEffort: "none"},
			want:   []string{"reasoning_effort"},
			absent: []string{"temperature", "max_tokens", "max_completion_tokens"},
		},
		{
			name:   "no reasoning configured (local model)",
			cfg:    LLMConfig{Model: "qwen3-8b"},
			want:   []string{"temperature"},
			absent: []string{"reasoning_effort", "max_completion_tokens", "max_tokens"},
		},
		{
			name:   "reasoning with explicit cap",
			cfg:    LLMConfig{Model: "gpt-6-luna", ReasoningEffort: "none", MaxCompletionTokens: 8192},
			want:   []string{"max_completion_tokens"},
			absent: []string{"max_tokens"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewLLMExtractor(tc.cfg)
			req := e.buildRequest("hello")
			b, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var m map[string]any
			if err := json.Unmarshal(b, &m); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			for _, k := range tc.want {
				if _, ok := m[k]; !ok {
					t.Errorf("expected %q in body, got %v", k, m)
				}
			}
			for _, k := range tc.absent {
				if _, ok := m[k]; ok {
					t.Errorf("did not expect %q in body, got %v", k, m)
				}
			}
			if tc.cfg.MaxCompletionTokens > 0 {
				if got := m["max_completion_tokens"]; got != float64(tc.cfg.MaxCompletionTokens) {
					t.Errorf("max_completion_tokens = %v, want %d", got, tc.cfg.MaxCompletionTokens)
				}
			}
			if tc.cfg.ReasoningEffort != "" {
				if got := m["reasoning_effort"]; got != tc.cfg.ReasoningEffort {
					t.Errorf("reasoning_effort = %v, want %q", got, tc.cfg.ReasoningEffort)
				}
			}
		})
	}
}
