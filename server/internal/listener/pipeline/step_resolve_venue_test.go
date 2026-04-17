package pipeline

import (
	"context"
	"testing"
)

func TestResolveVenueStep_Resolved(t *testing.T) {
	step := NewResolveVenueStep(&fakeVenueResolver{
		result: &VenueResolved{ID: 42, Name: "Peirce Secondary School"},
	})
	pc := makePC()
	pc.Params.Venue = "Peirce Sec"

	if err := step.Process(context.Background(), pc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.Params.VenueID == nil || *pc.Params.VenueID != 42 {
		t.Errorf("expected VenueID 42, got %v", pc.Params.VenueID)
	}
}

func TestResolveVenueStep_Unresolved(t *testing.T) {
	step := NewResolveVenueStep(&fakeVenueResolver{result: nil})
	pc := makePC()
	pc.Params.Venue = "Unknown Place"

	if err := step.Process(context.Background(), pc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.Params.VenueID != nil {
		t.Errorf("expected nil VenueID, got %v", pc.Params.VenueID)
	}
}

func TestResolveVenueStep_EmptyVenue(t *testing.T) {
	step := NewResolveVenueStep(&fakeVenueResolver{
		result: &VenueResolved{ID: 1, Name: "Should Not Be Called"},
	})
	pc := makePC()
	pc.Params.Venue = ""

	if err := step.Process(context.Background(), pc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.Params.VenueID != nil {
		t.Error("expected nil VenueID for empty venue")
	}
}

func TestResolveVenueStep_NilResolver(t *testing.T) {
	step := NewResolveVenueStep(nil)
	pc := makePC()

	if err := step.Process(context.Background(), pc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
