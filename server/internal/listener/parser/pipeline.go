package parser

import (
	"context"
	"fmt"
	"math"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

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

// Pipeline orchestrates message splitting and LLM-based extraction.
type Pipeline struct {
	extractor *LLMExtractor
}

// NewPipeline creates a parser pipeline with the given LLM config.
func NewPipeline(cfg LLMConfig) *Pipeline {
	return &Pipeline{
		extractor: NewLLMExtractor(cfg),
	}
}

// Parse takes a raw message and returns parsed play candidates.
// The full message text is sent to the LLM, which extracts all plays from it.
func (p *Pipeline) Parse(ctx context.Context, input MessageInput) ([]model.ParsedPlayCandidate, error) {
	refDate := input.Timestamp.Format("2006-01-02")

	candidates, err := p.extractor.Extract(ctx, input.Text, refDate, input.SenderName)
	if err != nil {
		return nil, err
	}

	return candidates, nil
}

// ResolvedVenue holds the normalized venue data from the venues table.
// Nil means the venue could not be resolved.
type ResolvedVenue struct {
	ID   int64
	Name string // canonical venue name, used as venue_norm
}

// ToUpsertPlayParams converts a ParsedPlayCandidate directly into db.UpsertPlayParams
// for database insertion. This avoids going through db.Play which has different
// nullability for some fields.
func ToUpsertPlayParams(c *model.ParsedPlayCandidate, input MessageInput, rv *ResolvedVenue) db.UpsertPlayParams {
	currency := "SGD"
	if c.Currency != nil {
		currency = *c.Currency
	}

	hostName := input.SenderName
	if c.HostName != nil && *c.HostName != "" {
		hostName = *c.HostName
	}

	listingType := model.ListingPlay
	if c.ListingType != nil && *c.ListingType == string(model.ListingSellBooking) {
		listingType = model.ListingSellBooking
	}

	tz := input.Timezone
	if tz == "" {
		tz = "Asia/Singapore"
	}

	source := input.Source

	params := db.UpsertPlayParams{
		ListingType:          listingType,
		Sport:                model.SportBadminton,
		GameType:             toGameType(c.GameType),
		HostName:             hostName,
		StartsAt:             derefTimeOrZero(model.ToUTC(c.Date, c.StartTime, tz)),
		EndsAt:               derefTimeOrZero(model.ToUTC(c.Date, c.EndTime, tz)),
		Timezone:             tz,
		Venue:                derefStringOrEmpty(c.Venue),
		LevelMin:             c.LevelMin,
		LevelMax:             c.LevelMax,
		LevelMinOrd:          intToInt64(levelToOrd(c.LevelMin)),
		LevelMaxOrd:          intToInt64(levelToOrd(c.LevelMax)),
		Fee:                  intToInt64(c.FeeCents),
		Currency:             currency,
		MaxPlayers:           intToInt64(c.MaxPlayers),
		SlotsLeft:            intToInt64(c.SlotsLeft),
		Courts:               floatToInt64(c.Courts),
		Contacts:             model.Contacts(c.Contacts),
		GenderPref:           toGenderPref(c.GenderPref),
		Meta:                 buildMeta(c),
		Source:               &source,
		SourceSenderUsername: nilIfEmpty(input.SenderUsername),
		SourceSenderName:     nilIfEmpty(input.SenderName),
		SourceRawMessage:     &input.Text,
		SourceMessageTime:    &input.Timestamp,
		SourceMessageID:      input.SourceMessageID,
		SourceGroup:          input.SourceGroup,
	}

	if rv != nil {
		params.VenueID = &rv.ID
	}

	return params
}

func toGenderPref(s *string) *model.GenderPref {
	if s == nil {
		return nil
	}
	gp := model.GenderPref(*s)
	switch gp {
	case model.GenderAll, model.GenderMaleOnly, model.GenderFemaleOnly:
		return &gp
	}
	return nil
}

func toGameType(s *string) *model.GameType {
	if s == nil {
		return nil
	}
	gt := model.GameType(*s)
	switch gt {
	case model.GameDoubles, model.GameSingles, model.GameMixedDoubles:
		return &gt
	}
	return nil
}

// levelToOrd converts a level code string to its numeric ordinal.
func levelToOrd(code *string) *int {
	if code == nil {
		return nil
	}
	return model.LevelOrd(model.SportBadminton, *code)
}

func buildMeta(c *model.ParsedPlayCandidate) model.Meta {
	meta := make(model.Meta)
	if c.Shuttle != nil {
		meta["shuttle"] = *c.Shuttle
	}
	if c.AirCon != nil {
		meta["air_con"] = *c.AirCon
	}
	if c.Details != nil {
		meta["details"] = *c.Details
	}
	if c.FeeMaleCents != nil {
		meta["fee_male"] = *c.FeeMaleCents
	}
	if c.FeeFemaleCents != nil {
		meta["fee_female"] = *c.FeeFemaleCents
	}
	if c.LevelMaleMin != nil {
		meta["level_male_min"] = *c.LevelMaleMin
	}
	if c.LevelMaleMax != nil {
		meta["level_male_max"] = *c.LevelMaleMax
	}
	if c.LevelFemaleMin != nil {
		meta["level_female_min"] = *c.LevelFemaleMin
	}
	if c.LevelFemaleMax != nil {
		meta["level_female_max"] = *c.LevelFemaleMax
	}
	if c.Courts != nil && !isWhole(*c.Courts) {
		courtsNote := fmt.Sprintf("%g courts", *c.Courts)
		if existing, ok := meta["details"].(string); ok && existing != "" {
			meta["details"] = existing + ", " + courtsNote
		} else {
			meta["details"] = courtsNote
		}
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}

// --- type conversion helpers ---

func intToInt64(v *int) *int64 {
	if v == nil {
		return nil
	}
	i := int64(*v)
	return &i
}

func floatToInt64(v *float64) *int64 {
	if v == nil {
		return nil
	}
	i := int64(math.Floor(*v))
	return &i
}

// isWhole returns true if the float has no fractional part.
func isWhole(v float64) bool {
	return v == math.Floor(v)
}

func derefTimeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func derefStringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
