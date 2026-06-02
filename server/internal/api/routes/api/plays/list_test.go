package plays

import (
	"context"
	"testing"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

func TestBuildFiltersDateRange(t *testing.T) {
	f := buildFilters(&ListInput{
		StartsAfter:  "2026-04-10",
		StartsBefore: "2026-04-11",
		Timezone:     "Asia/Singapore",
	})

	if got, want := f.startsAfter, "2026-04-09 16:00:00+00:00"; got != want {
		t.Errorf("starts_after = %v, want %v", got, want)
	}
	if got, want := f.startsBefore, "2026-04-11 16:00:00+00:00"; got != want {
		t.Errorf("starts_before = %v, want exclusive next-day bound %v", got, want)
	}
}

func TestBuildFiltersDateRange_DefaultUTC(t *testing.T) {
	f := buildFilters(&ListInput{
		StartsAfter:  "2026-04-10",
		StartsBefore: "2026-04-11",
	})

	if got, want := f.startsAfter, "2026-04-10 00:00:00+00:00"; got != want {
		t.Errorf("starts_after = %v, want %v", got, want)
	}
	if got, want := f.startsBefore, "2026-04-12 00:00:00+00:00"; got != want {
		t.Errorf("starts_before = %v, want exclusive next-day bound %v", got, want)
	}
}

func TestBuildFiltersDateRange_InvalidTimezoneFallsBackToUTC(t *testing.T) {
	f := buildFilters(&ListInput{
		StartsAfter:  "2026-04-10",
		StartsBefore: "2026-04-11",
		Timezone:     "Mars/Olympus",
	})

	if got, want := f.startsAfter, "2026-04-10 00:00:00+00:00"; got != want {
		t.Errorf("starts_after = %v, want %v", got, want)
	}
	if got, want := f.startsBefore, "2026-04-12 00:00:00+00:00"; got != want {
		t.Errorf("starts_before = %v, want exclusive next-day bound %v", got, want)
	}
}

func TestEncodeTimeCursor(t *testing.T) {
	tests := []struct {
		name     string
		startsAt string
		id       string
		want     string
	}{
		{
			name:     "standard RFC3339 stays RFC3339",
			startsAt: "2026-04-10T12:00:00Z",
			id:       "018f9f0e-5d2d-7777-9b9c-111111111111",
			want:     "2026-04-10T12:00:00Z,018f9f0e-5d2d-7777-9b9c-111111111111",
		},
		{
			name:     "with timezone offset",
			startsAt: "2026-04-10T20:00:00+08:00",
			id:       "018f9f0e-5d2d-7777-9b9c-222222222222",
			want:     "2026-04-10T12:00:00Z,018f9f0e-5d2d-7777-9b9c-222222222222", // normalized to UTC RFC3339
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeTimeCursor(tt.startsAt, tt.id)
			if got != tt.want {
				t.Errorf("encodeTimeCursor(%q, %q) = %q, want %q", tt.startsAt, tt.id, got, tt.want)
			}
		})
	}
}

func TestDecodeTimeCursor(t *testing.T) {
	tests := []struct {
		name     string
		cursor   string
		wantTime string
		wantID   string
		wantOK   bool
	}{
		{
			name:     "valid cursor",
			cursor:   "2026-04-10T12:00:00Z,018f9f0e-5d2d-7777-9b9c-111111111111",
			wantTime: "2026-04-10T12:00:00Z",
			wantID:   "018f9f0e-5d2d-7777-9b9c-111111111111",
			wantOK:   true,
		},
		{
			name:   "empty cursor",
			cursor: "",
			wantOK: false,
		},
		{
			name:   "no comma",
			cursor: "invalid",
			wantOK: false,
		},
		{
			name:   "missing id",
			cursor: "2026-04-10T12:00:00Z,",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, gotID, gotOK := decodeTimeCursor(tt.cursor)
			if gotOK != tt.wantOK {
				t.Fatalf("decodeTimeCursor(%q) ok = %v, want %v", tt.cursor, gotOK, tt.wantOK)
			}
			if !gotOK {
				return
			}
			if gotTime != tt.wantTime {
				t.Errorf("decodeTimeCursor(%q) time = %q, want %q", tt.cursor, gotTime, tt.wantTime)
			}
			if gotID != tt.wantID {
				t.Errorf("decodeTimeCursor(%q) id = %q, want %q", tt.cursor, gotID, tt.wantID)
			}
		})
	}
}

func TestCursorRoundTrip(t *testing.T) {
	// Encode from API format (RFC3339), decode, and verify the cursor stays
	// in RFC3339 externally.
	wantID := "018f9f0e-5d2d-7777-9b9c-111111111111"
	cursor := encodeTimeCursor("2026-04-10T12:00:00Z", wantID)

	startsAt, id, ok := decodeTimeCursor(cursor)
	if !ok {
		t.Fatalf("decodeTimeCursor(%q) failed", cursor)
	}
	if id != wantID {
		t.Errorf("round-trip id = %q, want %q", id, wantID)
	}
	if startsAt != "2026-04-10T12:00:00Z" {
		t.Errorf("round-trip time = %q, want RFC3339", startsAt)
	}
}

func TestCursorStartsAtForDB(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{
			name:   "convert RFC3339 UTC to sqlite format",
			input:  "2026-04-10T12:00:00Z",
			want:   "2026-04-10 12:00:00+00:00",
			wantOK: true,
		},
		{
			name:   "convert offset time to sqlite format",
			input:  "2026-04-10T20:00:00+08:00",
			want:   "2026-04-10 12:00:00+00:00",
			wantOK: true,
		},
		{
			name:   "invalid time",
			input:  "not-a-time",
			want:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := cursorStartsAtForDB(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("cursorStartsAtForDB(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("cursorStartsAtForDB(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildFilters_SportAndTennisLevel(t *testing.T) {
	f := buildFilters(&ListInput{
		Sport:    "tennis",
		LevelMin: "3.0",
		LevelMax: "4.0",
	})

	if got, want := f.sport, "tennis"; got != want {
		t.Errorf("sport = %v, want %v", got, want)
	}
	if got, want := f.filterLevelMinOrd, 30; got != want {
		t.Errorf("filter_level_min_ord = %v, want %v", got, want)
	}
	if got, want := f.filterLevelMaxOrd, 40; got != want {
		t.Errorf("filter_level_max_ord = %v, want %v", got, want)
	}
}

func TestMapTimeRowOmitsTimestampsForUserCreatedPlays(t *testing.T) {
	creatorID := "creator-1"
	item := mapTimeRow(listUpcomingPlayRowWithCreatedBy(&creatorID))

	if item.CreatedAt != nil {
		t.Fatalf("created_at = %v, want omitted for user-created play", *item.CreatedAt)
	}
	if item.UpdatedAt != nil {
		t.Fatalf("updated_at = %v, want omitted for user-created play", *item.UpdatedAt)
	}
}

func TestMapTimeRowIncludesTimestampsForImportedPlays(t *testing.T) {
	item := mapTimeRow(listUpcomingPlayRowWithCreatedBy(nil))

	if item.CreatedAt == nil || *item.CreatedAt != "2026-05-01T10:00:00Z" {
		t.Fatalf("created_at = %v, want 2026-05-01T10:00:00Z", item.CreatedAt)
	}
	if item.UpdatedAt == nil || *item.UpdatedAt != "2026-05-02T10:00:00Z" {
		t.Fatalf("updated_at = %v, want 2026-05-02T10:00:00Z", item.UpdatedAt)
	}
}

func TestMapDistanceRowOmitsTimestampsForUserCreatedPlays(t *testing.T) {
	creatorID := "creator-1"
	item := mapDistanceRow(listUpcomingPlayByDistanceRowWithCreatedBy(&creatorID))

	if item.CreatedAt != nil {
		t.Fatalf("created_at = %v, want omitted for user-created play", *item.CreatedAt)
	}
	if item.UpdatedAt != nil {
		t.Fatalf("updated_at = %v, want omitted for user-created play", *item.UpdatedAt)
	}
}

func TestMapDistanceRowIncludesTimestampsForImportedPlays(t *testing.T) {
	item := mapDistanceRow(listUpcomingPlayByDistanceRowWithCreatedBy(nil))

	if item.CreatedAt == nil || *item.CreatedAt != "2026-05-01T10:00:00Z" {
		t.Fatalf("created_at = %v, want 2026-05-01T10:00:00Z", item.CreatedAt)
	}
	if item.UpdatedAt == nil || *item.UpdatedAt != "2026-05-02T10:00:00Z" {
		t.Fatalf("updated_at = %v, want 2026-05-02T10:00:00Z", item.UpdatedAt)
	}
}

func TestHydrateParticipantPreviewsIncludesMissingHostFirst(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	hostID := createListPreviewUser(t, ctx, queries, "list-preview-host", "List Preview Host")
	playerID := createListPreviewUser(t, ctx, queries, "list-preview-player", "List Preview Player")
	play := createListPreviewPlay(t, ctx, queries, "list-preview-play", hostID)

	if _, err := queries.CreatePlayHost(ctx, db.CreatePlayHostParams{PlayID: play.ID, UserID: hostID}); err != nil {
		t.Fatalf("CreatePlayHost: %v", err)
	}
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: play.ID,
		UserID: &playerID,
		Status: model.ParticipantConfirmed,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant: %v", err)
	}

	items := []PlayPublic{{ID: play.ID, Sport: play.Sport}}
	if err := hydrateParticipantPreviews(ctx, queries, items, true); err != nil {
		t.Fatalf("hydrateParticipantPreviews: %v", err)
	}

	previews := items[0].ParticipantPreview
	if len(previews) != 2 {
		t.Fatalf("participant_preview len = %d, want 2", len(previews))
	}
	if !previews[0].IsHost {
		t.Fatal("participant_preview[0].is_host = false, want host first")
	}
	if previews[0].DisplayName == nil || *previews[0].DisplayName != "List Preview Host" {
		t.Fatalf("participant_preview[0].display_name = %v, want host", previews[0].DisplayName)
	}
	if previews[1].IsHost {
		t.Fatal("participant_preview[1].is_host = true, want non-host second")
	}
	if previews[1].DisplayName == nil || *previews[1].DisplayName != "List Preview Player" {
		t.Fatalf("participant_preview[1].display_name = %v, want player", previews[1].DisplayName)
	}
}

func createListPreviewUser(t *testing.T, ctx context.Context, queries *db.Queries, id, displayName string) string {
	t.Helper()
	googleID := "google-" + id
	user, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID:          id,
		Email:       id + "@example.com",
		DisplayName: displayName,
		GoogleID:    &googleID,
	})
	if err != nil {
		t.Fatalf("UpsertUserByGoogleID(%s): %v", id, err)
	}
	return user.ID
}

func createListPreviewPlay(t *testing.T, ctx context.Context, queries *db.Queries, id, hostID string) db.Play {
	t.Helper()
	maxPlayers := int64(4)
	play, err := queries.CreatePlay(ctx, db.CreatePlayParams{
		ID:          id,
		ListingType: model.ListingPlay,
		Sport:       model.SportBadminton,
		HostName:    "List Preview Host",
		StartsAt:    time.Now().UTC().Add(24 * time.Hour),
		EndsAt:      time.Now().UTC().Add(26 * time.Hour),
		Timezone:    "Asia/Singapore",
		Venue:       "SBH",
		Currency:    "SGD",
		MaxPlayers:  &maxPlayers,
		SlotsLeft:   &maxPlayers,
		CreatedBy:   &hostID,
	})
	if err != nil {
		t.Fatalf("CreatePlay: %v", err)
	}
	return play
}

func listUpcomingPlayRowWithCreatedBy(createdBy *string) db.ListUpcomingPlaysRow {
	createdAt := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	startsAt := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	return db.ListUpcomingPlaysRow{
		ID:          "play-1",
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt.Add(24 * time.Hour),
		ListingType: model.ListingPlay,
		Sport:       model.SportBadminton,
		HostName:    "Host",
		StartsAt:    startsAt,
		EndsAt:      startsAt.Add(2 * time.Hour),
		Timezone:    "Asia/Singapore",
		Venue:       "SBH",
		VenueName:   "SBH",
		CreatedBy:   createdBy,
		Currency:    "SGD",
	}
}

func listUpcomingPlayByDistanceRowWithCreatedBy(createdBy *string) db.ListUpcomingPlaysByDistanceRow {
	row := listUpcomingPlayRowWithCreatedBy(createdBy)
	return db.ListUpcomingPlaysByDistanceRow{
		ID:             row.ID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		ListingType:    row.ListingType,
		Sport:          row.Sport,
		HostName:       row.HostName,
		StartsAt:       row.StartsAt,
		EndsAt:         row.EndsAt,
		Timezone:       row.Timezone,
		Venue:          row.Venue,
		VenueName:      row.VenueName,
		CreatedBy:      row.CreatedBy,
		Currency:       row.Currency,
		VenueLatitude:  1.3,
		VenueLongitude: 103.8,
		DistanceKm:     1.2,
	}
}
