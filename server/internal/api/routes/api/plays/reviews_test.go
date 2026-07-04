package plays_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/plays"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/testdb"
)

func setupReviewsTest(authStore *fakeAuthStore, store *db.Queries) *httptest.Server {
	svc := auth.NewService(authStore)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/plays")
	plays.RegisterPlayReviews(grp, store, authmw.RequireAuth(api, svc))

	return httptest.NewServer(r)
}

// setPlayEndsAt moves a play's window so its review window state can be
// controlled from tests.
func setPlayEndsAt(t *testing.T, ctx context.Context, sqlDB *sql.DB, playID string, endsAt time.Time) {
	t.Helper()

	startsAt := endsAt.Add(-2 * time.Hour)
	if _, err := sqlDB.ExecContext(ctx, "UPDATE plays SET starts_at = ?, ends_at = ? WHERE id = ?", startsAt, endsAt, playID); err != nil {
		t.Fatalf("set play ends_at: %v", err)
	}
}

func seedGuestParticipant(t *testing.T, ctx context.Context, queries *db.Queries, playID, guestName string) {
	t.Helper()

	if _, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID:    playID,
		GuestName: &guestName,
		Status:    "confirmed",
	}); err != nil {
		t.Fatalf("CreatePlayParticipant guest: %v", err)
	}
}

func doReviewRequest(t *testing.T, method, url, body string, authed bool) *http.Response {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authed {
		req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

type reviewSheet struct {
	Window struct {
		State    string `json:"state"`
		ClosesAt string `json:"closes_at"`
	} `json:"window"`
	PeerProps []string `json:"peer_props"`
	HostProps []string `json:"host_props"`
	Reviewees []struct {
		UserID      string `json:"user_id"`
		DisplayName string `json:"display_name"`
		IsHost      bool   `json:"is_host"`
		MyReview    *struct {
			Rating   *int64   `json:"rating"`
			Props    []string `json:"props"`
			Shoutout *string  `json:"shoutout"`
		} `json:"my_review"`
	} `json:"reviewees"`
}

func decodeReviewSheet(t *testing.T, resp *http.Response) reviewSheet {
	t.Helper()
	defer resp.Body.Close()

	var sheet reviewSheet
	if err := json.NewDecoder(resp.Body).Decode(&sheet); err != nil {
		t.Fatalf("decode review sheet: %v", err)
	}
	return sheet
}

func TestGetPlayReviews_OnlyParticipantsAndOnlyEligibleCoPlayers(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	hostID := createRouteTestUser(t, ctx, queries, "rev-host")
	viewerID := createRouteTestUser(t, ctx, queries, "rev-viewer")
	addedID := createRouteTestUser(t, ctx, queries, "rev-added")
	waitlistedID := createRouteTestUser(t, ctx, queries, "rev-waitlisted")
	requestedID := createRouteTestUser(t, ctx, queries, "rev-requested")
	outsiderID := createRouteTestUser(t, ctx, queries, "rev-outsider")

	playID := createUserPlay(t, ctx, queries, hostID, 6, nil, nil)
	// The host has no participant row: hosts are eligible via play_hosts alone
	seedConfirmedParticipant(t, ctx, queries, playID, viewerID)
	seedAddedParticipant(t, ctx, queries, playID, addedID)
	seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)
	seedRequestedParticipant(t, ctx, queries, playID, requestedID)
	seedGuestParticipant(t, ctx, queries, playID, "Guest One")
	markPlayEnded(t, ctx, sqlDB, playID)

	ts := setupReviewsTest(sessionWithProfile(viewerID, nil), queries)
	defer ts.Close()

	resp := doReviewRequest(t, http.MethodGet, ts.URL+"/api/plays/"+playID+"/reviews", "", true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	sheet := decodeReviewSheet(t, resp)

	if sheet.Window.State != "open" {
		t.Fatalf("window state = %q, want open", sheet.Window.State)
	}
	if len(sheet.PeerProps) == 0 || len(sheet.HostProps) == 0 {
		t.Fatalf("props missing: peer=%v host=%v", sheet.PeerProps, sheet.HostProps)
	}

	// Reviewees: host + added player. The viewer, pending players, and guests
	// are out.
	if len(sheet.Reviewees) != 2 {
		t.Fatalf("reviewees = %d, want 2 (%+v)", len(sheet.Reviewees), sheet.Reviewees)
	}
	byID := map[string]bool{}
	for _, reviewee := range sheet.Reviewees {
		byID[reviewee.UserID] = reviewee.IsHost
	}
	if isHost, ok := byID[hostID]; !ok || !isHost {
		t.Fatalf("host reviewee missing or not flagged host: %v", byID)
	}
	if isHost, ok := byID[addedID]; !ok || isHost {
		t.Fatalf("added reviewee missing or wrongly flagged host: %v", byID)
	}

	// A non-participant is locked out entirely
	outsiderTS := setupReviewsTest(sessionWithProfile(outsiderID, nil), queries)
	defer outsiderTS.Close()
	outsiderResp := doReviewRequest(t, http.MethodGet, outsiderTS.URL+"/api/plays/"+playID+"/reviews", "", true)
	outsiderResp.Body.Close()
	if outsiderResp.StatusCode != http.StatusForbidden {
		t.Fatalf("outsider status = %d, want 403", outsiderResp.StatusCode)
	}

	// Unauthenticated requests never see review data
	unauthResp := doReviewRequest(t, http.MethodGet, ts.URL+"/api/plays/"+playID+"/reviews", "", false)
	unauthResp.Body.Close()
	if unauthResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", unauthResp.StatusCode)
	}
}

func TestPutPlayReview_WindowAndCancelledEnforcement(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	hostID := createRouteTestUser(t, ctx, queries, "rev-win-host")
	viewerID := createRouteTestUser(t, ctx, queries, "rev-win-viewer")
	playID := createUserPlay(t, ctx, queries, hostID, 4, nil, nil)
	seedConfirmedParticipant(t, ctx, queries, playID, hostID)
	seedConfirmedParticipant(t, ctx, queries, playID, viewerID)

	ts := setupReviewsTest(sessionWithProfile(viewerID, nil), queries)
	defer ts.Close()
	url := ts.URL + "/api/plays/" + playID + "/reviews/" + hostID

	// Upcoming play: window not open yet
	resp := doReviewRequest(t, http.MethodPut, url, `{"rating":5}`, true)
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("before end status = %d, want 409", resp.StatusCode)
	}

	// Ended more than 14 days ago: window closed
	setPlayEndsAt(t, ctx, sqlDB, playID, time.Now().UTC().Add(-15*24*time.Hour))
	resp = doReviewRequest(t, http.MethodPut, url, `{"rating":5}`, true)
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("after window status = %d, want 409", resp.StatusCode)
	}

	// Within the window: accepted
	setPlayEndsAt(t, ctx, sqlDB, playID, time.Now().UTC().Add(-2*time.Hour))
	resp = doReviewRequest(t, http.MethodPut, url, `{"rating":5}`, true)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("within window status = %d, want 200", resp.StatusCode)
	}

	// Cancelled plays are never reviewable
	if _, err := queries.CancelUserCreatedPlay(ctx, db.CancelUserCreatedPlayParams{
		ID:          playID,
		CancelledBy: &hostID,
	}); err != nil {
		t.Fatalf("cancel play: %v", err)
	}
	resp = doReviewRequest(t, http.MethodPut, url, `{"rating":5}`, true)
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("cancelled status = %d, want 409", resp.StatusCode)
	}
}

func TestPutPlayReview_Validation(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	hostID := createRouteTestUser(t, ctx, queries, "rev-val-host")
	viewerID := createRouteTestUser(t, ctx, queries, "rev-val-viewer")
	peerID := createRouteTestUser(t, ctx, queries, "rev-val-peer")
	waitlistedID := createRouteTestUser(t, ctx, queries, "rev-val-waitlisted")
	playID := createUserPlay(t, ctx, queries, hostID, 4, nil, nil)
	seedConfirmedParticipant(t, ctx, queries, playID, hostID)
	seedConfirmedParticipant(t, ctx, queries, playID, viewerID)
	seedConfirmedParticipant(t, ctx, queries, playID, peerID)
	seedWaitlistedParticipant(t, ctx, queries, playID, waitlistedID)
	markPlayEnded(t, ctx, sqlDB, playID)

	ts := setupReviewsTest(sessionWithProfile(viewerID, nil), queries)
	defer ts.Close()
	reviewURL := func(revieweeID string) string {
		return ts.URL + "/api/plays/" + playID + "/reviews/" + revieweeID
	}

	cases := []struct {
		name       string
		revieweeID string
		body       string
		wantStatus int
	}{
		{"self review", viewerID, `{"rating":5}`, http.StatusUnprocessableEntity},
		{"reviewee not on final roster", waitlistedID, `{"rating":5}`, http.StatusNotFound},
		{"rating above bounds", peerID, `{"rating":6}`, http.StatusUnprocessableEntity},
		{"rating below bounds", peerID, `{"rating":0}`, http.StatusUnprocessableEntity},
		{"unknown prop", peerID, `{"props":["free_beer"]}`, http.StatusUnprocessableEntity},
		{"another sport's prop", peerID, `{"props":["big_serve"]}`, http.StatusUnprocessableEntity},
		{"too many props", peerID, `{"props":["great_sport","humble","punctual"]}`, http.StatusUnprocessableEntity},
		{"host prop on non-host", peerID, `{"props":["well_organized"]}`, http.StatusUnprocessableEntity},
		{"empty review", peerID, `{}`, http.StatusUnprocessableEntity},
		{"whitespace shoutout only", peerID, `{"shoutout":"   "}`, http.StatusUnprocessableEntity},
		{"host prop on host", hostID, `{"props":["well_organized"]}`, http.StatusOK},
		{"props only", peerID, `{"props":["great_sport"]}`, http.StatusOK},
		{"sport skill prop", peerID, `{"props":["powerful_smash"]}`, http.StatusOK},
		{"shoutout only", peerID, `{"shoutout":"carried the doubles"}`, http.StatusOK},
		{"rating only", peerID, `{"rating":1}`, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doReviewRequest(t, http.MethodPut, reviewURL(tc.revieweeID), tc.body, true)
			resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestPutPlayReview_EditOverwritesSingleRow(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	hostID := createRouteTestUser(t, ctx, queries, "rev-edit-host")
	viewerID := createRouteTestUser(t, ctx, queries, "rev-edit-viewer")
	playID := createUserPlay(t, ctx, queries, hostID, 4, nil, nil)
	seedConfirmedParticipant(t, ctx, queries, playID, hostID)
	seedConfirmedParticipant(t, ctx, queries, playID, viewerID)
	markPlayEnded(t, ctx, sqlDB, playID)

	ts := setupReviewsTest(sessionWithProfile(viewerID, nil), queries)
	defer ts.Close()
	url := ts.URL + "/api/plays/" + playID + "/reviews/" + hostID

	first := doReviewRequest(t, http.MethodPut, url, `{"rating":5,"shoutout":"great game"}`, true)
	first.Body.Close()
	if first.StatusCode != http.StatusOK {
		t.Fatalf("first put status = %d, want 200", first.StatusCode)
	}

	// The edit drops the shoutout, changes the rating, and adds props
	second := doReviewRequest(t, http.MethodPut, url, `{"rating":3,"props":["well_organized","great_sport"]}`, true)
	second.Body.Close()
	if second.StatusCode != http.StatusOK {
		t.Fatalf("second put status = %d, want 200", second.StatusCode)
	}

	rows, err := queries.ListMyPlayReviews(ctx, db.ListMyPlayReviewsParams{
		PlayID:         playID,
		ReviewerUserID: viewerID,
	})
	if err != nil {
		t.Fatalf("ListMyPlayReviews: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("review rows = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.Rating == nil || *row.Rating != 3 {
		t.Fatalf("rating = %v, want 3", row.Rating)
	}
	if row.Shoutout != nil {
		t.Fatalf("shoutout = %v, want cleared", row.Shoutout)
	}
	if row.Props != `["well_organized","great_sport"]` {
		t.Fatalf("props = %q", row.Props)
	}

	// The edited review is echoed back on the sheet
	sheetResp := doReviewRequest(t, http.MethodGet, ts.URL+"/api/plays/"+playID+"/reviews", "", true)
	sheet := decodeReviewSheet(t, sheetResp)
	if len(sheet.Reviewees) != 1 || sheet.Reviewees[0].MyReview == nil {
		t.Fatalf("sheet missing my_review: %+v", sheet.Reviewees)
	}
	mine := sheet.Reviewees[0].MyReview
	if mine.Rating == nil || *mine.Rating != 3 || mine.Shoutout != nil || len(mine.Props) != 2 {
		t.Fatalf("my_review = %+v", mine)
	}
}
