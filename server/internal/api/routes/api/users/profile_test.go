package users_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/model"
)

func TestGetUserProfile_ReturnsMinimalProfileAndRosterCount(t *testing.T) {
	ts, queries := setupSearchTest(t)
	defer ts.Close()
	ctx := context.Background()

	viewer := createSearchTestUser(t, ctx, queries, "profile-viewer", "Profile Viewer", strPtr("profile_viewer"), nil)
	createSearchSession(t, ctx, queries, viewer.ID, "tok")

	username := "alice_tan"
	level := "LI"
	profileRaw := mustSportsProfileRaw(t, &model.SportsProfile{
		Badminton: &model.SportLevelProfile{Level: &level},
	})
	target := createSearchTestUser(t, ctx, queries, "profile-alice", "Alice Tan", &username, profileRaw)

	hostedPlayID := createProfileTestPlay(t, ctx, queries, "profile-play-hosted", target.ID)
	confirmedPlayID := createProfileTestPlay(t, ctx, queries, "profile-play-confirmed", viewer.ID)
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: confirmedPlayID,
		UserID: &target.ID,
		Status: model.ParticipantConfirmed,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant confirmed: %v", err)
	}
	waitlistPlayID := createProfileTestPlay(t, ctx, queries, "profile-play-waitlist", viewer.ID)
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: waitlistPlayID,
		UserID: &target.ID,
		Status: model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant waitlist: %v", err)
	}
	_ = hostedPlayID

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/users/alice_tan", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := raw["email"]; ok {
		t.Fatal("profile response leaked email")
	}
	if _, ok := raw["contact_info"]; ok {
		t.Fatal("profile response leaked contact_info")
	}
	if raw["display_name"] != "Alice Tan" {
		t.Fatalf("display_name = %v, want Alice Tan", raw["display_name"])
	}
	if raw["username"] != "alice_tan" {
		t.Fatalf("username = %v, want alice_tan", raw["username"])
	}
	if raw["rostered_play_count"] != float64(2) {
		t.Fatalf("rostered_play_count = %v, want 2", raw["rostered_play_count"])
	}
}

func TestGetUserProfile_NoAuthReturns401(t *testing.T) {
	ts, _ := setupSearchTest(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/users/alice_tan")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func createProfileTestPlay(t *testing.T, ctx context.Context, queries *db.Queries, id, hostID string) string {
	t.Helper()
	maxPlayers := int64(4)
	startsAt := time.Now().UTC().Add(24 * time.Hour)
	play, err := queries.CreatePlay(ctx, db.CreatePlayParams{
		ID:          id,
		ListingType: model.ListingPlay,
		Sport:       model.SportBadminton,
		HostName:    "Host",
		StartsAt:    startsAt,
		EndsAt:      startsAt.Add(2 * time.Hour),
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
	if _, err := queries.CreatePlayHost(ctx, db.CreatePlayHostParams{
		PlayID: play.ID,
		UserID: hostID,
	}); err != nil {
		t.Fatalf("CreatePlayHost: %v", err)
	}
	return play.ID
}

func strPtr(value string) *string {
	return &value
}
