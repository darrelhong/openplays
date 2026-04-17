package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"openplays/server/internal/model"
)

// LLMConfig holds configuration for the LLM endpoint.
// Works with LM Studio, OpenAI, Together, or any OpenAI-compatible API.
type LLMConfig struct {
	BaseURL   string        // e.g. "http://localhost:1234/v1" or "https://api.openai.com/v1"
	Model     string        // model name, e.g. "qwen3-8b", "gpt-4o-mini"
	APIKey    string        // optional — required for cloud providers, empty for local
	Timeout   time.Duration // per-request timeout
	RateLimit int           // max requests per minute (0 = unlimited)
}

// DefaultLLMConfig returns config for a local LM Studio instance.
func DefaultLLMConfig() LLMConfig {
	return LLMConfig{
		BaseURL:   "http://localhost:1234/v1",
		Model:     "",
		Timeout:   500 * time.Second,
		RateLimit: 0,
	}
}

// LLMExtractor extracts structured play data from text blocks using a local LLM.
type LLMExtractor struct {
	config  LLMConfig
	client  *http.Client
	limiter *rate.Limiter
}

// NewLLMExtractor creates a new extractor with the given config.
func NewLLMExtractor(cfg LLMConfig) *LLMExtractor {
	var limiter *rate.Limiter
	if cfg.RateLimit > 0 {
		interval := time.Minute / time.Duration(cfg.RateLimit)
		limiter = rate.NewLimiter(rate.Every(interval), 1)
		slog.Info("llm rate limiter configured", "requests_per_min", cfg.RateLimit, "interval", interval)
	}

	return &LLMExtractor{
		config:  cfg,
		client:  &http.Client{Timeout: cfg.Timeout},
		limiter: limiter,
	}
}

const systemPrompt = `You are a structured data extractor for sports session listings (usually badminton in Singapore).

Given a text block, extract ALL individual listings into a JSON array. A single block may contain multiple listings — for example different time slots, different levels at the same time, or different venues under the same date.

There are two types of listings:
- "play": organising a game, looking for players to join. Fee is per person.
- "sell_booking": reselling/letting go a booked facility (court, pitch, etc.). Fee is total cost. Indicators: "court let go", "letting go court", "court to let go", "letting go at cost". These typically have no level, no shuttle, no max_players.

SKIP and do NOT extract these — they are not play sessions:
- Coaching/training advertisements (e.g. "badminton training/coaching", "trial classes available", "certified coach", "stringing services")
- Listings with no specific venue (e.g. "anywhere in SG", "location to be discussed", "venue TBD")
- Generic recurring schedules with no specific date (e.g. "Monday-Sunday", "every weekend")

A single message may contain BOTH skippable and extractable content. For example, a coaching ad followed by "letting go court: today, 12/4, edgefield primary 4-5pm" — skip the coaching part, extract only the court let-go. If the entire message is skippable, return an empty array [].

IMPORTANT: Each unique combination of (date + venue + time slot + level) is a SEPARATE listing. You MUST output one JSON object per combination. Common patterns:
- Multiple dates listed under the same time/level: EACH DATE is a separate listing.
  Example: "7pm-9pm, LI, $12, 📅 31/3, 📅 6/4, 📅 7/4" = 3 listings (one per date, all LI $12)
- Multiple level/price tiers with their own dates: EACH DATE within EACH TIER is separate.
  Example: "7pm-9pm LI $12: 📅31/3 📅6/4 📅7/4 📅8/4" AND "7pm-9pm MB-HB $10: 📅31/3 📅6/4 📅8/4"
  = 4 + 3 = 7 total listings. Dates appearing in both groups (31/3, 6/4, 8/4) produce TWO listings each — one for each level/price.
- Multiple time slots: EACH TIME SLOT is a separate listing.
  Example: "10am-12pm/1-3pm/3-5pm $13 LI" = 3 listings

Field rules:
- listing_type: "play" or "sell_booking". Default "play" unless the text indicates reselling/letting go a booking.
- host_name: the name of the person hosting/organising the session. Use the sender name provided. If the text mentions a different organiser name, use that instead.
- game_type: "doubles", "singles", or "mixed_doubles". Infer from text: "doubles game", "mixed doubles", "singles". Default "doubles" for badminton if not specified. Null if truly unclear.
- date: "YYYY-MM-DD". If year missing, assume 2026. Resolve "today"/"tomorrow" relative to the reference date.
- start_time, end_time: "HH:MM" 24-hour. "4pm"->"16:00", "8-10pm"->start="20:00" end="22:00", "0730pm"->"19:30", "730"(in pm context)->"19:30".
- venue: the venue NAME only, without parenthetical details. "Hougang Sec (Rubber floor)" -> "Hougang Sec". "SBH Expo (Air Con)" -> "SBH Expo". "Farrer Park (Rubber flooring)" -> "Farrer Park". Strip descriptions like floor type, nearby MRT, area info from the venue name.
- level_min, level_max: standard badminton codes: LB, MB, HB, LI, MI, HI, A. "High Beginner"->"HB", "Low Intermediate & above"->level_min="LI" level_max=null, "Mid-high beginners"->level_min="MB" level_max="HB", "High Beginners-LI"->level_min="HB" level_max="LI". If single level, set both min and max equal. "X & above" means level_max=null. Null for sell_booking listings. If different levels per gender (e.g. "🚺MB-HB 🚹HB-LI"), use the broadest range for level_min/level_max and put the gendered breakdown in level_male_min/level_male_max/level_female_min/level_female_max.
- level_raw: original level text as written.
- fee_cents: dollar to cents. "$10"->1000, "$9.50"->950. CJK "九"->900. For sell_booking, this is the total facility cost, not per person. Must be null if no specific fee is mentioned.
- fee_male_cents, fee_female_cents: for gendered pricing with ABSOLUTE prices only, e.g. "👨 $12, 👩 $11"->fee_male_cents=1200, fee_female_cents=1100. If the text gives a RELATIVE discount (e.g. "$2 lower for females", "ladies $3 off"), do NOT compute fee_male_cents/fee_female_cents — put the discount note in "details" instead. Never output negative fee values.
- currency: default "SGD".
- max_players: "6 pax max"->6, "Max 6, including host"->6.
- slots_left: "2 slot left"->2, "Slots: 3"->3.
- courts: "2 Courts"->2, "one court"->1, "3.5 courts"->3.5. Use the exact number from the text.
- gender_pref: "all", "male_only", "female_only", or null.
- shuttle: brand/model e.g. "RSL Supreme". Strip "New" prefix.
- air_con: true if mentioned, null otherwise.
- contacts: [{type, value}]. Types: "whatsapp", "phone", "telegram_pm". Username or digits only for phone numbers.
- details: venue details and other notable info as a short string. Include floor type, air-con, parking, facilities, relative pricing discounts (e.g. "Female $2 less on weekends"), and anything else stripped from the venue name. E.g. "Rubber floor", "Air Con, Female $2 less on weekends", "Parquet flooring, Free parking". Null if nothing notable.

Output ONLY a JSON array of objects. No explanation, no markdown fences. Even for a single listing, wrap it in an array.

Each object schema:
{
  "listing_type": "string",
  "host_name": "string",
  "game_type": "string or null",
  "date": "string or null",
  "start_time": "string or null",
  "end_time": "string or null",
  "venue": "string or null",
  "level_min": "string or null",
  "level_max": "string or null",
  "level_raw": "string or null",
  "level_male_min": "string or null",
  "level_male_max": "string or null",
  "level_female_min": "string or null",
  "level_female_max": "string or null",
  "fee_cents": "integer or null",
  "fee_male_cents": "integer or null",
  "fee_female_cents": "integer or null",
  "currency": "string",
  "max_players": "integer or null",
  "slots_left": "integer or null",
  "courts": "number or null",
  "gender_pref": "string or null",
  "shuttle": "string or null",
  "air_con": "boolean or null",
  "details": "string or null",
  "contacts": [{"type": "string", "value": "string"}]
}`

// chatRequest is the OpenAI-compatible chat completion request.
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the OpenAI-compatible chat completion response.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// Extract sends a text block to the local LLM and returns one or more
// ParsedPlayCandidates. A single block may contain multiple plays (e.g.
// different time slots, different levels at the same time/venue, etc.).
func (e *LLMExtractor) Extract(ctx context.Context, block string, referenceDate string, senderName string) ([]model.ParsedPlayCandidate, error) {
	if e.limiter != nil {
		if err := e.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}
	}

	userPrompt := fmt.Sprintf("Sender name: %s\nReference date (today): %s\n\nText block:\n%s", senderName, referenceDate, block)

	reqBody := chatRequest{
		Model: e.config.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.0,
		MaxTokens:   25600,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(e.config.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if e.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.config.APIKey)
	}

	start := time.Now()
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read LLM response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("decode LLM response JSON: %w (body: %.500s)", err, string(respBody))
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("LLM returned no choices (body: %.500s)", string(respBody))
	}

	content := chatResp.Choices[0].Message.Content
	elapsed := time.Since(start)
	if chatResp.Usage != nil {
		slog.Info("llm response", "latency", elapsed.Round(time.Millisecond), "tokens_total", chatResp.Usage.TotalTokens, "tokens_in", chatResp.Usage.PromptTokens, "tokens_out", chatResp.Usage.CompletionTokens)
	} else {
		slog.Info("llm response", "latency", elapsed.Round(time.Millisecond))
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("LLM returned empty content (body: %.500s)", string(respBody))
	}

	content = cleanJSONResponse(content)

	candidates, err := parseResponse(content, block)
	if err != nil {
		return nil, err
	}

	return candidates, nil
}

// parseResponse handles both array and single-object JSON responses.
func parseResponse(content string, block string) ([]model.ParsedPlayCandidate, error) {
	content = strings.TrimSpace(content)

	// Try array first (most common response format)
	var candidates []model.ParsedPlayCandidate
	if err := json.Unmarshal([]byte(content), &candidates); err != nil {
		// Try single object as fallback
		var single model.ParsedPlayCandidate
		if singleErr := json.Unmarshal([]byte(content), &single); singleErr != nil {
			// Report the array error since that's the expected format
			return nil, fmt.Errorf("parse LLM JSON output: %w (raw: %.500s)", err, content)
		}
		single.RawBlock = block
		return []model.ParsedPlayCandidate{single}, nil
	}
	for i := range candidates {
		candidates[i].RawBlock = block
	}
	return candidates, nil
}

// cleanJSONResponse strips markdown code fences and whitespace that LLMs
// sometimes wrap around JSON output.
func cleanJSONResponse(s string) string {
	s = strings.TrimSpace(s)

	// Strip ```json ... ``` wrapper
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
	}

	return strings.TrimSpace(s)
}
