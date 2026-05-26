package plays

import (
	"testing"

	"openplays/server/internal/model"
)

func TestMapParticipantPreviewRows_HidesNamesForListCards(t *testing.T) {
	userID := "user-1"
	userName := "Alice Tan"
	photoURL := "https://example.com/alice.png"
	profile := `{"tennis":{"level":"4.2"},"badminton":{"level":"LI"}}`
	guestName := "Guest One"
	guestRating := "3.5"

	previews := mapParticipantPreviewRows(model.SportTennis, []participantPreviewRow{
		{
			ID:            1,
			UserID:        &userID,
			DisplayName:   &userName,
			PhotoUrl:      &photoURL,
			SportsProfile: &profile,
		},
		{
			ID:         2,
			GuestName:  &guestName,
			RatingCode: &guestRating,
		},
	}, false)

	if len(previews) != 2 {
		t.Fatalf("previews = %d, want 2", len(previews))
	}
	if previews[0].DisplayName != nil || previews[1].DisplayName != nil {
		t.Fatalf("display names should be hidden in list previews: %#v", previews)
	}
	if got := stringPtrValue(previews[0].PhotoURL); got != photoURL {
		t.Fatalf("photo_url = %q, want %q", got, photoURL)
	}
	if got := stringPtrValue(previews[0].RatingCode); got != "4.2" {
		t.Fatalf("user rating = %q, want 4.2", got)
	}
	if got := stringPtrValue(previews[1].RatingCode); got != guestRating {
		t.Fatalf("guest rating = %q, want %q", got, guestRating)
	}
	if !previews[1].IsGuest {
		t.Fatal("guest preview IsGuest = false, want true")
	}
}

func TestMapParticipantPreviewRows_IncludesNamesForDetailAndPrefersSnapshotRating(t *testing.T) {
	userID := "user-1"
	userName := "Alice Tan"
	profile := `{"tennis":{"level":"4.2"}}`
	snapshotRating := "4.0"

	previews := mapParticipantPreviewRows(model.SportTennis, []participantPreviewRow{
		{
			ID:            1,
			UserID:        &userID,
			RatingCode:    &snapshotRating,
			DisplayName:   &userName,
			SportsProfile: &profile,
		},
	}, true)

	if len(previews) != 1 {
		t.Fatalf("previews = %d, want 1", len(previews))
	}
	if got := stringPtrValue(previews[0].DisplayName); got != userName {
		t.Fatalf("display_name = %q, want %q", got, userName)
	}
	if got := stringPtrValue(previews[0].RatingCode); got != snapshotRating {
		t.Fatalf("rating_code = %q, want participant snapshot %q", got, snapshotRating)
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
