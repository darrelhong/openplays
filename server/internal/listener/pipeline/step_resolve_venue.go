package pipeline

import (
	"context"
	"log/slog"
)

// VenueResolved is the result of a venue resolution.
type VenueResolved struct {
	ID   int64
	Name string
}

// VenueResolver resolves a raw venue string to a canonical venue.
type VenueResolver interface {
	Resolve(ctx context.Context, rawVenue *string) *VenueResolved
}

// ResolveVenueStep resolves the raw venue text to a canonical venue from the DB.
type ResolveVenueStep struct {
	resolver VenueResolver
}

func NewResolveVenueStep(r VenueResolver) *ResolveVenueStep {
	return &ResolveVenueStep{resolver: r}
}

func (s *ResolveVenueStep) Name() string { return "resolve-venue" }

func (s *ResolveVenueStep) Process(ctx context.Context, pc *PlayContext) error {
	if s.resolver == nil {
		return nil
	}

	venueName := pc.Params.Venue
	if venueName == "" {
		return nil
	}

	resolved := s.resolver.Resolve(ctx, &venueName)
	if resolved != nil {
		pc.Params.VenueID = &resolved.ID
	} else {
		slog.Warn("venue unresolved", "venue", venueName, "message_id", pc.MessageID)
	}

	return nil
}
