package plays_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/plays"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

func setupGetDetailTest(authStore *fakeAuthStore, store *db.Queries) *httptest.Server {
	svc := auth.NewService(authStore)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/plays")
	plays.RegisterGet(grp, store, authmw.OptionalAuth(api, svc))

	return httptest.NewServer(r)
}

func TestGetPlayDetail_VisibilityAndDerivedCounts(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "detail-creator-1")
	confirmedID := createRouteTestUser(t, ctx, queries, "detail-confirmed-1")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))

	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedConfirmedParticipant(t, ctx, queries, playID, confirmedID)
	guestName := "Guest Wait"
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:    playID,
		GuestName: &guestName,
		Status:    model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant waitlist guest: %v", err)
	}

	ts := setupGetDetailTest(&fakeAuthStore{sessionErr: context.Canceled}, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		ConfirmedParticipants []struct {
			DisplayName *string `json:"display_name"`
			IsHost      bool    `json:"is_host"`
		} `json:"confirmed_participants"`
		Waitlist []struct {
			DisplayName *string `json:"display_name"`
		} `json:"waitlist"`
		CreatedAt      *string `json:"created_at"`
		UpdatedAt      *string `json:"updated_at"`
		ViewerState    *string `json:"viewer_state"`
		CanManage      *bool   `json:"can_manage"`
		ConfirmedCount *int64  `json:"confirmed_count"`
		WaitlistCount  *int64  `json:"waitlist_count"`
		SlotsLeft      *int64  `json:"slots_left"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(out.ConfirmedParticipants) != 2 {
		t.Fatalf("confirmed_participants len = %d, want 2", len(out.ConfirmedParticipants))
	}
	if !out.ConfirmedParticipants[0].IsHost {
		t.Fatal("confirmed_participants[0].is_host = false, want true")
	}
	if out.ConfirmedParticipants[1].IsHost {
		t.Fatal("confirmed_participants[1].is_host = true, want false")
	}
	if len(out.Waitlist) != 0 {
		t.Fatalf("waitlist len = %d, want 0 for anonymous viewer", len(out.Waitlist))
	}
	if out.CreatedAt != nil {
		t.Fatalf("created_at = %v, want omitted for user-created play", *out.CreatedAt)
	}
	if out.UpdatedAt != nil {
		t.Fatalf("updated_at = %v, want omitted for user-created play", *out.UpdatedAt)
	}
	if out.ConfirmedCount == nil || *out.ConfirmedCount != 2 {
		t.Fatalf("confirmed_count = %v, want 2", out.ConfirmedCount)
	}
	if out.WaitlistCount == nil || *out.WaitlistCount != 1 {
		t.Fatalf("waitlist_count = %v, want 1", out.WaitlistCount)
	}
	if out.SlotsLeft == nil || *out.SlotsLeft != 2 {
		t.Fatalf("slots_left = %v, want 2", out.SlotsLeft)
	}
	if out.ViewerState == nil || *out.ViewerState != "not_joined" {
		t.Fatalf("viewer_state = %v, want not_joined", out.ViewerState)
	}
	if out.CanManage == nil || *out.CanManage {
		t.Fatalf("can_manage = %v, want false", out.CanManage)
	}
}

func TestGetPlayDetail_UnlistedGameIsViewableByLink(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "detail-unlisted-host")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))
	play, err := queries.GetPlayByID(ctx, playID)
	if err != nil {
		t.Fatalf("GetPlayByID seed: %v", err)
	}
	if _, err := queries.UpdateUserCreatedPlay(ctx, db.UpdateUserCreatedPlayParams{
		ID:          playID,
		Visibility:  model.PlayVisibilityUnlisted,
		Name:        play.Name,
		Description: play.Description,
		GameType:    play.GameType,
		StartsAt:    play.StartsAt,
		EndsAt:      play.EndsAt,
		Timezone:    play.Timezone,
		LevelMin:    play.LevelMin,
		LevelMax:    play.LevelMax,
		LevelMinOrd: play.LevelMinOrd,
		LevelMaxOrd: play.LevelMaxOrd,
		Fee:         play.Fee,
		MaxPlayers:  play.MaxPlayers,
		Courts:      play.Courts,
	}); err != nil {
		t.Fatalf("UpdateUserCreatedPlay visibility: %v", err)
	}

	ts := setupGetDetailTest(&fakeAuthStore{sessionErr: context.Canceled}, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		Visibility model.PlayVisibility `json:"visibility"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Visibility != model.PlayVisibilityUnlisted {
		t.Fatalf("visibility = %q, want unlisted", out.Visibility)
	}
}

func TestGetPlayDetail_HistoryEventsVisibleToCurrentParticipantsOnly(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "detail-history-host")
	playerID := createRouteTestUser(t, ctx, queries, "detail-history-player")
	formerID := createRouteTestUser(t, ctx, queries, "detail-history-former")
	waitlistedID := createRouteTestUser(t, ctx, queries, "detail-history-waitlisted")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedConfirmedParticipant(t, ctx, queries, playID, playerID)

	createdName := "Hidden Create"
	waitlistedName := "Waitlisted Player"
	playerName := "Confirmed Player"
	hostName := "Host User"
	formerName := "Former Player"
	if _, err := queries.CreatePlayEvent(ctx, db.CreatePlayEventParams{
		PlayID:           playID,
		EventType:        model.PlayEventCreated,
		ActorUserID:      &creatorID,
		ActorDisplayName: &createdName,
	}); err != nil {
		t.Fatalf("CreatePlayEvent created: %v", err)
	}
	if _, err := queries.CreatePlayEvent(ctx, db.CreatePlayEventParams{
		PlayID:             playID,
		EventType:          model.PlayEventParticipantJoinedWaitlist,
		ActorUserID:        &waitlistedID,
		ActorDisplayName:   &waitlistedName,
		SubjectUserID:      &waitlistedID,
		SubjectDisplayName: &waitlistedName,
	}); err != nil {
		t.Fatalf("CreatePlayEvent waitlist: %v", err)
	}
	if _, err := queries.CreatePlayEvent(ctx, db.CreatePlayEventParams{
		PlayID:             playID,
		EventType:          model.PlayEventParticipantJoinedConfirmed,
		ActorUserID:        &playerID,
		ActorDisplayName:   &playerName,
		SubjectUserID:      &playerID,
		SubjectDisplayName: &playerName,
	}); err != nil {
		t.Fatalf("CreatePlayEvent joined confirmed: %v", err)
	}
	if _, err := queries.CreatePlayEvent(ctx, db.CreatePlayEventParams{
		PlayID:             playID,
		EventType:          model.PlayEventParticipantAdded,
		ActorUserID:        &creatorID,
		ActorDisplayName:   &hostName,
		SubjectUserID:      &waitlistedID,
		SubjectDisplayName: &waitlistedName,
	}); err != nil {
		t.Fatalf("CreatePlayEvent added: %v", err)
	}
	if _, err := queries.CreatePlayEvent(ctx, db.CreatePlayEventParams{
		PlayID:             playID,
		EventType:          model.PlayEventParticipantRemoved,
		ActorUserID:        &creatorID,
		ActorDisplayName:   &hostName,
		SubjectUserID:      &formerID,
		SubjectDisplayName: &formerName,
	}); err != nil {
		t.Fatalf("CreatePlayEvent removed: %v", err)
	}
	if _, err := queries.CreatePlayEvent(ctx, db.CreatePlayEventParams{
		PlayID:           playID,
		EventType:        model.PlayEventUpdated,
		ActorUserID:      &creatorID,
		ActorDisplayName: &hostName,
	}); err != nil {
		t.Fatalf("CreatePlayEvent updated: %v", err)
	}

	playerEvents := getHistoryEvents(t, setupGetDetailTest(sessionWithProfile(playerID, nil), queries), playID)
	playerEventTypes := historyEventTypes(playerEvents)
	wantPlayerEvents := []string{
		string(model.PlayEventParticipantJoinedConfirmed),
		string(model.PlayEventParticipantAdded),
		string(model.PlayEventUpdated),
	}
	if !equalStrings(playerEventTypes, wantPlayerEvents) {
		t.Fatalf("player history event types = %v, want %v", playerEventTypes, wantPlayerEvents)
	}
	if playerEvents[1].ActorDisplayName != nil {
		t.Fatalf("added actor_display_name = %v, want omitted", *playerEvents[1].ActorDisplayName)
	}
	if playerEvents[1].Message != "Waitlisted Player was added to the game" {
		t.Fatalf("added message = %q, want redacted actor copy", playerEvents[1].Message)
	}

	hostEvents := getHistoryEvents(t, setupGetDetailTest(sessionWithProfile(creatorID, nil), queries), playID)
	hostEventTypes := historyEventTypes(hostEvents)
	wantHostEvents := []string{
		string(model.PlayEventParticipantJoinedWaitlist),
		string(model.PlayEventParticipantJoinedConfirmed),
		string(model.PlayEventParticipantAdded),
		string(model.PlayEventParticipantRemoved),
		string(model.PlayEventUpdated),
	}
	if !equalStrings(hostEventTypes, wantHostEvents) {
		t.Fatalf("host history event types = %v, want %v", hostEventTypes, wantHostEvents)
	}
	if hostEvents[2].ActorDisplayName != nil {
		t.Fatalf("host added actor_display_name = %v, want omitted", *hostEvents[2].ActorDisplayName)
	}
	if hostEvents[2].Message != "Waitlisted Player was added to the game" {
		t.Fatalf("host added message = %q, want redacted actor copy", hostEvents[2].Message)
	}
	if hostEvents[3].ActorDisplayName != nil {
		t.Fatalf("removed actor_display_name = %v, want omitted", *hostEvents[3].ActorDisplayName)
	}
	if hostEvents[3].Message != "Former Player was removed from the game" {
		t.Fatalf("removed message = %q, want redacted actor copy", hostEvents[3].Message)
	}

	formerEvents := getHistoryEvents(t, setupGetDetailTest(sessionWithProfile(formerID, nil), queries), playID)
	if len(formerEvents) != 0 {
		t.Fatalf("former participant history event types = %v, want none", formerEvents)
	}
}

type historyEventResponse struct {
	EventType        string  `json:"event_type"`
	Message          string  `json:"message"`
	ActorDisplayName *string `json:"actor_display_name"`
}

func getHistoryEvents(t *testing.T, ts *httptest.Server, playID string) []historyEventResponse {
	t.Helper()
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		HistoryEvents []historyEventResponse `json:"history_events"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return out.HistoryEvents
}

func historyEventTypes(events []historyEventResponse) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType)
	}
	return types
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestGetPlayDetail_ImportedPlayIncludesTimestamps(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	startsAt := time.Now().UTC().Add(24 * time.Hour)
	play, err := queries.CreatePlay(ctx, db.CreatePlayParams{
		ID:          "detail-imported-timestamps",
		ListingType: model.ListingPlay,
		Sport:       model.SportBadminton,
		HostName:    "Imported Host",
		StartsAt:    startsAt,
		EndsAt:      startsAt.Add(2 * time.Hour),
		Timezone:    "Asia/Singapore",
		Venue:       "SBH",
		Currency:    "SGD",
	})
	if err != nil {
		t.Fatalf("CreatePlay imported: %v", err)
	}

	ts := setupGetDetailTest(&fakeAuthStore{sessionErr: context.Canceled}, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+play.ID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		CreatedAt *string `json:"created_at"`
		UpdatedAt *string `json:"updated_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.CreatedAt == nil {
		t.Fatal("created_at omitted, want timestamp for imported play")
	}
	if out.UpdatedAt == nil {
		t.Fatal("updated_at omitted, want timestamp for imported play")
	}
}

func TestGetPlayDetail_CreatorPermissions(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "detail-creator-2")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	guestName := "Creator Visible Wait"
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:    playID,
		GuestName: &guestName,
		Status:    model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant waitlist guest: %v", err)
	}

	authStore := sessionWithProfile(creatorID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupGetDetailTest(authStore, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		Waitlist []struct {
			DisplayName *string `json:"display_name"`
		} `json:"waitlist"`
		ViewerState   *string `json:"viewer_state"`
		CanManage     *bool   `json:"can_manage"`
		WaitlistCount *int64  `json:"waitlist_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.ViewerState == nil || *out.ViewerState != "creator" {
		t.Fatalf("viewer_state = %v, want creator", out.ViewerState)
	}
	if out.CanManage == nil || !*out.CanManage {
		t.Fatalf("can_manage = %v, want true", out.CanManage)
	}
	if len(out.Waitlist) != 1 {
		t.Fatalf("waitlist len = %d, want 1 for creator", len(out.Waitlist))
	}
	if out.Waitlist[0].DisplayName == nil || *out.Waitlist[0].DisplayName != guestName {
		t.Fatalf("waitlist[0].display_name = %v, want %q", out.Waitlist[0].DisplayName, guestName)
	}
	if out.WaitlistCount == nil || *out.WaitlistCount != 1 {
		t.Fatalf("waitlist_count = %v, want 1", out.WaitlistCount)
	}
}

func TestGetPlayDetail_ConfirmedParticipantsPutHostsFirst(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "detail-host-order-host")
	confirmedID := createRouteTestUser(t, ctx, queries, "detail-host-order-player")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))

	seedConfirmedParticipant(t, ctx, queries, playID, confirmedID)
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)

	ts := setupGetDetailTest(&fakeAuthStore{sessionErr: context.Canceled}, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		ConfirmedParticipants []struct {
			DisplayName *string `json:"display_name"`
			IsHost      bool    `json:"is_host"`
		} `json:"confirmed_participants"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(out.ConfirmedParticipants) != 2 {
		t.Fatalf("confirmed_participants len = %d, want 2", len(out.ConfirmedParticipants))
	}
	if !out.ConfirmedParticipants[0].IsHost {
		t.Fatal("confirmed_participants[0].is_host = false, want host first")
	}
	if out.ConfirmedParticipants[0].DisplayName == nil || *out.ConfirmedParticipants[0].DisplayName != "Test "+creatorID {
		t.Fatalf("first display_name = %v, want host", out.ConfirmedParticipants[0].DisplayName)
	}
	if out.ConfirmedParticipants[1].IsHost {
		t.Fatal("confirmed_participants[1].is_host = true, want non-host second")
	}
}

func TestGetPlayDetail_HostCanManageWithoutRosterSlot(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "detail-host-not-player")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))
	guestName := "Host Visible Wait"
	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:    playID,
		GuestName: &guestName,
		Status:    model.ParticipantWaitlisted,
	}); err != nil {
		t.Fatalf("CreatePlayParticipant waitlist guest: %v", err)
	}

	authStore := sessionWithProfile(creatorID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupGetDetailTest(authStore, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		ConfirmedParticipants []struct {
			DisplayName *string `json:"display_name"`
			IsHost      bool    `json:"is_host"`
		} `json:"confirmed_participants"`
		Waitlist []struct {
			DisplayName *string `json:"display_name"`
		} `json:"waitlist"`
		ViewerState    *string `json:"viewer_state"`
		CanManage      *bool   `json:"can_manage"`
		ConfirmedCount *int64  `json:"confirmed_count"`
		SlotsLeft      *int64  `json:"slots_left"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.ViewerState == nil || *out.ViewerState != "creator" {
		t.Fatalf("viewer_state = %v, want creator", out.ViewerState)
	}
	if out.CanManage == nil || !*out.CanManage {
		t.Fatalf("can_manage = %v, want true", out.CanManage)
	}
	if len(out.Waitlist) != 1 {
		t.Fatalf("waitlist len = %d, want 1 for host", len(out.Waitlist))
	}
	if len(out.ConfirmedParticipants) != 1 {
		t.Fatalf("confirmed_participants len = %d, want 1 host", len(out.ConfirmedParticipants))
	}
	if !out.ConfirmedParticipants[0].IsHost {
		t.Fatal("confirmed_participants[0].is_host = false, want true")
	}
	if out.ConfirmedParticipants[0].DisplayName == nil || *out.ConfirmedParticipants[0].DisplayName != "Test "+creatorID {
		t.Fatalf("confirmed_participants[0].display_name = %v, want host", out.ConfirmedParticipants[0].DisplayName)
	}
	if out.ConfirmedCount == nil || *out.ConfirmedCount != 1 {
		t.Fatalf("confirmed_count = %v, want 1", out.ConfirmedCount)
	}
	if out.SlotsLeft == nil || *out.SlotsLeft != 3 {
		t.Fatalf("slots_left = %v, want 3", out.SlotsLeft)
	}
}

func TestGetPlayDetail_HostCanSeeAddedParticipantsAndReservedSlots(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "detail-added-host")
	addedID := createRouteTestUser(t, ctx, queries, "detail-added-player")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedAddedParticipant(t, ctx, queries, playID, addedID)

	authStore := sessionWithProfile(creatorID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupGetDetailTest(authStore, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		AddedParticipants []struct {
			DisplayName *string `json:"display_name"`
		} `json:"added_participants"`
		AddedCount *int64 `json:"added_count"`
		SlotsLeft  *int64 `json:"slots_left"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(out.AddedParticipants) != 1 {
		t.Fatalf("added_participants len = %d, want 1", len(out.AddedParticipants))
	}
	if out.AddedCount == nil || *out.AddedCount != 1 {
		t.Fatalf("added_count = %v, want 1", out.AddedCount)
	}
	if out.SlotsLeft == nil || *out.SlotsLeft != 2 {
		t.Fatalf("slots_left = %v, want 2", out.SlotsLeft)
	}
}

func TestGetPlayDetail_AddedViewerSeesOnlyOwnAddedParticipant(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "detail-own-added-host")
	viewerID := createRouteTestUser(t, ctx, queries, "detail-own-added-viewer")
	otherAddedID := createRouteTestUser(t, ctx, queries, "detail-own-added-other")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedAddedParticipant(t, ctx, queries, playID, viewerID)
	seedAddedParticipant(t, ctx, queries, playID, otherAddedID)

	authStore := sessionWithProfile(viewerID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupGetDetailTest(authStore, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		AddedParticipants []struct {
			DisplayName *string `json:"display_name"`
		} `json:"added_participants"`
		ViewerState *string `json:"viewer_state"`
		CanManage   *bool   `json:"can_manage"`
		AddedCount  *int64  `json:"added_count"`
		SlotsLeft   *int64  `json:"slots_left"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.ViewerState == nil || *out.ViewerState != "added" {
		t.Fatalf("viewer_state = %v, want added", out.ViewerState)
	}
	if out.CanManage == nil || *out.CanManage {
		t.Fatalf("can_manage = %v, want false", out.CanManage)
	}
	if len(out.AddedParticipants) != 1 {
		t.Fatalf("added_participants len = %d, want own row only", len(out.AddedParticipants))
	}
	if out.AddedParticipants[0].DisplayName == nil || *out.AddedParticipants[0].DisplayName != "Test "+viewerID {
		t.Fatalf("added participant display_name = %v, want viewer", out.AddedParticipants[0].DisplayName)
	}
	if out.AddedCount == nil || *out.AddedCount != 2 {
		t.Fatalf("added_count = %v, want 2", out.AddedCount)
	}
	if out.SlotsLeft == nil || *out.SlotsLeft != 1 {
		t.Fatalf("slots_left = %v, want 1", out.SlotsLeft)
	}
}

func TestGetPlayDetail_WaitlistedViewerSeesOnlyOwnWaitlistParticipant(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "detail-own-waitlist-host")
	viewerID := createRouteTestUser(t, ctx, queries, "detail-own-waitlist-viewer")
	otherWaitlistedID := createRouteTestUser(t, ctx, queries, "detail-own-waitlist-other")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))
	seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
	seedWaitlistedParticipant(t, ctx, queries, playID, viewerID)
	seedWaitlistedParticipant(t, ctx, queries, playID, otherWaitlistedID)

	authStore := sessionWithProfile(viewerID, ptrString(`{"badminton":{"level":"HB"}}`))
	ts := setupGetDetailTest(authStore, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		Waitlist []struct {
			DisplayName *string `json:"display_name"`
		} `json:"waitlist"`
		ViewerState   *string `json:"viewer_state"`
		CanManage     *bool   `json:"can_manage"`
		WaitlistCount *int64  `json:"waitlist_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.ViewerState == nil || *out.ViewerState != "waitlisted" {
		t.Fatalf("viewer_state = %v, want waitlisted", out.ViewerState)
	}
	if out.CanManage == nil || *out.CanManage {
		t.Fatalf("can_manage = %v, want false", out.CanManage)
	}
	if len(out.Waitlist) != 1 {
		t.Fatalf("waitlist len = %d, want own row only", len(out.Waitlist))
	}
	if out.Waitlist[0].DisplayName == nil || *out.Waitlist[0].DisplayName != "Test "+viewerID {
		t.Fatalf("waitlist display_name = %v, want viewer", out.Waitlist[0].DisplayName)
	}
	if out.WaitlistCount == nil || *out.WaitlistCount != 2 {
		t.Fatalf("waitlist_count = %v, want 2", out.WaitlistCount)
	}
}

func TestGetPlayDetail_CancelledPlayIncludesCancellation(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	creatorID := createRouteTestUser(t, ctx, queries, "detail-cancelled-host")
	playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))
	if _, err := queries.CancelUserCreatedPlay(ctx, db.CancelUserCreatedPlayParams{
		ID:          playID,
		CancelledBy: &creatorID,
	}); err != nil {
		t.Fatalf("CancelUserCreatedPlay: %v", err)
	}

	ts := setupGetDetailTest(&fakeAuthStore{sessionErr: context.Canceled}, queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		CancelledAt *string `json:"cancelled_at"`
		CanManage   *bool   `json:"can_manage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.CancelledAt == nil || *out.CancelledAt == "" {
		t.Fatalf("cancelled_at = %v, want timestamp", out.CancelledAt)
	}
	if out.CanManage == nil || *out.CanManage {
		t.Fatalf("can_manage = %v, want false for anonymous viewer", out.CanManage)
	}
}

func TestGetPlayDetail_ViewerStateByParticipantStatus(t *testing.T) {
	tests := []struct {
		name            string
		participantSt   model.PlayParticipantStatus
		wantState       string
		wantWaitlistLen int
	}{
		{name: "confirmed", participantSt: model.ParticipantConfirmed, wantState: "confirmed"},
		{name: "waitlisted", participantSt: model.ParticipantWaitlisted, wantState: "waitlisted", wantWaitlistLen: 1},
		{name: "added", participantSt: model.ParticipantAdded, wantState: "added"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB := testdb.New(t)
			queries := db.New(sqlDB)
			ctx := context.Background()

			creatorID := createRouteTestUser(t, ctx, queries, "detail-creator-"+tc.name)
			viewerID := createRouteTestUser(t, ctx, queries, "detail-viewer-"+tc.name)
			playID := createUserPlay(t, ctx, queries, creatorID, 4, ptrString("MB"), ptrString("HI"))
			seedConfirmedParticipant(t, ctx, queries, playID, creatorID)
			if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
				PlayID: playID,
				UserID: &viewerID,
				Status: tc.participantSt,
			}); err != nil {
				t.Fatalf("CreatePlayParticipant viewer: %v", err)
			}
			if tc.participantSt != model.ParticipantWaitlisted {
				guestName := "Hidden Wait"
				if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
					PlayID:    playID,
					GuestName: &guestName,
					Status:    model.ParticipantWaitlisted,
				}); err != nil {
					t.Fatalf("CreatePlayParticipant waitlist guest: %v", err)
				}
			}

			authStore := sessionWithProfile(viewerID, ptrString(`{"badminton":{"level":"HB"}}`))
			ts := setupGetDetailTest(authStore, queries)
			defer ts.Close()

			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
			req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}

			var out struct {
				Waitlist      []struct{} `json:"waitlist"`
				ViewerState   *string    `json:"viewer_state"`
				CanManage     *bool      `json:"can_manage"`
				WaitlistCount *int64     `json:"waitlist_count"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.ViewerState == nil || *out.ViewerState != tc.wantState {
				t.Fatalf("viewer_state = %v, want %s", out.ViewerState, tc.wantState)
			}
			if out.CanManage == nil || *out.CanManage {
				t.Fatalf("can_manage = %v, want false", out.CanManage)
			}
			if len(out.Waitlist) != tc.wantWaitlistLen {
				t.Fatalf("waitlist len = %d, want %d", len(out.Waitlist), tc.wantWaitlistLen)
			}
			if out.WaitlistCount == nil || *out.WaitlistCount != 1 {
				t.Fatalf("waitlist_count = %v, want 1", out.WaitlistCount)
			}
		})
	}
}
