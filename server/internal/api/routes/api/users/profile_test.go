package users_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
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
	sports, ok := raw["sports"].([]any)
	if !ok || len(sports) != 1 {
		t.Fatalf("sports = %#v, want one sport summary", raw["sports"])
	}
	badminton, ok := sports[0].(map[string]any)
	if !ok {
		t.Fatalf("sports[0] = %#v, want object", sports[0])
	}
	if badminton["sport"] != "badminton" {
		t.Fatalf("sport = %v, want badminton", badminton["sport"])
	}
	if badminton["rating_code"] != "LI" {
		t.Fatalf("rating_code = %v, want LI", badminton["rating_code"])
	}
	if badminton["rostered_play_count"] != float64(2) {
		t.Fatalf("sport rostered_play_count = %v, want 2", badminton["rostered_play_count"])
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

func createProfileTestPlayWithSport(t *testing.T, ctx context.Context, queries *db.Queries, id, hostID string, sport model.Sport) string {
	t.Helper()
	maxPlayers := int64(4)
	startsAt := time.Now().UTC().Add(-24 * time.Hour)
	play, err := queries.CreatePlay(ctx, db.CreatePlayParams{
		ID:          id,
		ListingType: model.ListingPlay,
		Sport:       sport,
		HostName:    "Host",
		Name:        strPtr("Sunday Session"),
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

func seedProfileReview(t *testing.T, ctx context.Context, queries *db.Queries, playID, reviewerID, revieweeID string, rating *int64, propsJSON string, shoutout *string) {
	t.Helper()
	if _, err := queries.UpsertPlayReview(ctx, db.UpsertPlayReviewParams{
		PlayID:         playID,
		ReviewerUserID: reviewerID,
		RevieweeUserID: revieweeID,
		Rating:         rating,
		Props:          propsJSON,
		Shoutout:       shoutout,
	}); err != nil {
		t.Fatalf("UpsertPlayReview: %v", err)
	}
}

func int64Ptr(v int64) *int64 { return &v }

func TestGetUserProfile_ReviewReputation(t *testing.T) {
	ts, queries := setupSearchTest(t)
	defer ts.Close()
	ctx := context.Background()

	viewer := createSearchTestUser(t, ctx, queries, "rep-viewer", "Rep Viewer", strPtr("rep_viewer"), nil)
	createSearchSession(t, ctx, queries, viewer.ID, "tok")

	username := "rep_target"
	target := createSearchTestUser(t, ctx, queries, "rep-target", "Rep Target", &username, nil)
	silent := createSearchTestUser(t, ctx, queries, "rep-silent", "Silent Reviewer", strPtr("rep_silent"), nil)
	shouter := createSearchTestUser(t, ctx, queries, "rep-shouter", "Shouty Reviewer", strPtr("rep_shouter"), nil)

	badmintonPlayID := createProfileTestPlayWithSport(t, ctx, queries, "rep-play-bad", target.ID, model.SportBadminton)
	tennisPlayID := createProfileTestPlayWithSport(t, ctx, queries, "rep-play-ten", viewer.ID, model.SportTennis)

	// Two rated reviews (5 and 4) and one rating-less props/shoutout review
	seedProfileReview(t, ctx, queries, badmintonPlayID, silent.ID, target.ID, int64Ptr(5), `["great_sport","powerful_smash"]`, nil)
	seedProfileReview(t, ctx, queries, badmintonPlayID, viewer.ID, target.ID, int64Ptr(4), `["great_sport"]`, strPtr("Carried our doubles game"))
	seedProfileReview(t, ctx, queries, tennisPlayID, shouter.ID, target.ID, nil, `["big_serve"]`, strPtr("What a serve"))

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/users/rep_target", nil)
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

	// Rating averages only the rated reviews
	rating, ok := raw["rating"].(map[string]any)
	if !ok {
		t.Fatalf("rating = %#v, want object", raw["rating"])
	}
	if rating["average"] != float64(4.5) {
		t.Fatalf("rating average = %v, want 4.5", rating["average"])
	}
	if rating["count"] != float64(2) {
		t.Fatalf("rating count = %v, want 2", rating["count"])
	}
	// One 4-star and one 5-star review: distribution is [0 0 0 1 1]
	distribution, ok := rating["distribution"].([]any)
	if !ok || len(distribution) != 5 {
		t.Fatalf("distribution = %#v, want 5 buckets", rating["distribution"])
	}
	for star, want := range []float64{0, 0, 0, 1, 1} {
		if distribution[star] != want {
			t.Fatalf("distribution[%d] = %v, want %v", star, distribution[star], want)
		}
	}

	// Props count under the sport they were earned in
	props, ok := raw["props"].([]any)
	if !ok || len(props) != 3 {
		t.Fatalf("props = %#v, want 3 rows", raw["props"])
	}
	propCounts := map[string]float64{}
	for _, entry := range props {
		row := entry.(map[string]any)
		propCounts[row["sport"].(string)+"/"+row["prop"].(string)] = row["count"].(float64)
	}
	if propCounts["badminton/great_sport"] != 2 {
		t.Fatalf("badminton great_sport = %v, want 2", propCounts["badminton/great_sport"])
	}
	if propCounts["badminton/powerful_smash"] != 1 {
		t.Fatalf("badminton powerful_smash = %v, want 1", propCounts["badminton/powerful_smash"])
	}
	if propCounts["tennis/big_serve"] != 1 {
		t.Fatalf("tennis big_serve = %v, want 1", propCounts["tennis/big_serve"])
	}

	// Anonymity: shoutouts live on their own endpoint, so the profile
	// response must contain no reviewer identity at all; ratings are
	// aggregate-only.
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, leaked := range []string{"Silent Reviewer", "rep_silent", silent.ID, "Shouty Reviewer", "rep_shouter", shouter.ID} {
		if strings.Contains(string(body), leaked) {
			t.Fatalf("profile response leaked reviewer identity %q", leaked)
		}
	}
}

func TestGetUserProfile_NoReviewsOmitsRating(t *testing.T) {
	ts, queries := setupSearchTest(t)
	defer ts.Close()
	ctx := context.Background()

	viewer := createSearchTestUser(t, ctx, queries, "norep-viewer", "Viewer", strPtr("norep_viewer"), nil)
	createSearchSession(t, ctx, queries, viewer.ID, "tok")
	username := "norep_target"
	createSearchTestUser(t, ctx, queries, "norep-target", "No Rep", &username, nil)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/users/norep_target", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := raw["rating"]; ok {
		t.Fatalf("rating = %#v, want omitted at zero reviews", raw["rating"])
	}
	if props, ok := raw["props"].([]any); !ok || len(props) != 0 {
		t.Fatalf("props = %#v, want empty array", raw["props"])
	}
}
