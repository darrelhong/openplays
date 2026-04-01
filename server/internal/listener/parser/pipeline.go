package parser

import (
	"context"
	"time"

	"openplays/server/internal/model"
)

// MessageInput holds the raw message data from a source (Telegram, etc.).
type MessageInput struct {
	Text       string    // full raw message text
	SenderName string    // username or display name of sender
	Timestamp  time.Time // when the message was sent
	Timezone   string    // IANA timezone of the source, e.g. "Asia/Singapore"
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

// ToPlay converts a ParsedPlayCandidate into a canonical Play record.
// Times are converted from the candidate's local date/time strings to UTC
// using the timezone from MessageInput.
func ToPlay(c *model.ParsedPlayCandidate, input MessageInput) model.Play {
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

	play := model.Play{
		ListingType:          listingType,
		Sport:                model.SportBadminton,
		GameType:             toGameType(c.GameType),
		HostName:             hostName,
		StartsAt:             model.ToUTC(c.Date, c.StartTime, tz),
		EndsAt:               model.ToUTC(c.Date, c.EndTime, tz),
		Timezone:             tz,
		Venue:                c.Venue,
		LevelMin:             c.LevelMin,
		LevelMax:             c.LevelMax,
		Fee:                  c.FeeCents,
		Currency:             currency,
		MaxPlayers:           c.MaxPlayers,
		SlotsLeft:            c.SlotsLeft,
		Courts:               c.Courts,
		Contacts:             c.Contacts,
		GenderPref:           toGenderPref(c.GenderPref),
		Meta:                 buildMeta(c),
		Source:               "telegram",
		SourceSenderUsername: input.SenderName,
		SourceRawMessage:     input.Text,
		SourceMessageTime:    input.Timestamp,
	}

	return play
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

func buildMeta(c *model.ParsedPlayCandidate) map[string]any {
	meta := make(map[string]any)
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
	if len(meta) == 0 {
		return nil
	}
	return meta
}
