package pipeline

import (
	"context"

	"openplays/server/internal/venue"
)

// DefaultPipeline builds the standard end-to-end pipeline: LLM extraction →
// convert → validate → resolve venue → upsert.
// Pass nil for venueResolver to skip venue resolution.
func DefaultPipeline(cfg LLMConfig, venueResolver *venue.Resolver, playStore PlayStore) *Pipeline {
	extractor := NewLLMExtractor(cfg)

	var vr VenueResolver
	if venueResolver != nil {
		vr = &venueResolverAdapter{r: venueResolver}
	}

	return New(
		extractor,
		&ValidateStep{},
		NewResolveVenueStep(vr),
		NewUpsertStep(playStore),
	)
}

// venueResolverAdapter wraps venue.Resolver to satisfy the VenueResolver interface,
// keeping the pipeline package decoupled from the venue package's concrete types.
type venueResolverAdapter struct {
	r *venue.Resolver
}

func (a *venueResolverAdapter) Resolve(ctx context.Context, rawVenue *string) *VenueResolved {
	resolved := a.r.Resolve(ctx, rawVenue)
	if resolved == nil {
		return nil
	}
	return &VenueResolved{ID: resolved.ID, Name: resolved.Name}
}
