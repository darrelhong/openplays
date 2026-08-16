package me_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/me"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/notifications"
	"openplays/server/internal/testdb"
)

type followRequestNotificationCall struct {
	userID  string
	payload notifications.Payload
}

type followRequestNotificationSender struct {
	calls []followRequestNotificationCall
}

func (f *followRequestNotificationSender) Notify(_ context.Context, userID string, payload notifications.Payload) error {
	f.calls = append(f.calls, followRequestNotificationCall{userID: userID, payload: payload})
	return nil
}

func setupFollowRequests(t *testing.T, queries *db.Queries, notifier notifications.Sender) *httptest.Server {
	t.Helper()
	svc := auth.NewService(queries)
	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/me")
	grp.UseMiddleware(authmw.RequireAuth(api, svc))
	me.RegisterGet(grp, queries)
	me.RegisterFollowRequests(grp, queries, notifier)
	return httptest.NewServer(r)
}

func TestListFollowRequests_PaginatesPendingFollowers(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	owner := createFollowRequestUser(t, ctx, queries, "follow-request-owner", "Owner", "owner")
	createSearchSessionForMe(t, ctx, queries, owner.ID, "tok")
	for _, id := range []string{"requester-a", "requester-b", "requester-c"} {
		requester := createFollowRequestUser(t, ctx, queries, id, "Requester "+id[len(id)-1:], id)
		if _, err := queries.CreateUserFollow(ctx, db.CreateUserFollowParams{
			FollowerID: requester.ID,
			FolloweeID: owner.ID,
			Status:     "pending",
		}); err != nil {
			t.Fatalf("CreateUserFollow(%s): %v", id, err)
		}
	}

	ts := setupFollowRequests(t, queries, &followRequestNotificationSender{})
	defer ts.Close()

	first := getFollowRequestPage(t, ts, "?limit=2")
	if len(first.Items) != 2 || first.Items[0].ID != "requester-c" || first.Items[1].ID != "requester-b" {
		t.Fatalf("first items = %#v, want requester-c, requester-b", first.Items)
	}
	if first.NextCursor == nil || !first.HasMore || first.Total != 3 {
		t.Fatalf("first page metadata = %#v, want total 3 and next page", first)
	}

	second := getFollowRequestPage(t, ts, "?limit=2&cursor="+url.QueryEscape(*first.NextCursor))
	if len(second.Items) != 1 || second.Items[0].ID != "requester-a" {
		t.Fatalf("second items = %#v, want requester-a", second.Items)
	}
	if second.NextCursor != nil || second.HasMore {
		t.Fatalf("second page metadata = %#v, want no next page", second)
	}
}

func TestAcceptFollowRequest_UpdatesStatusAndNotifiesRequester(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	owner := createFollowRequestUser(t, ctx, queries, "accept-owner", "Owner", "accept_owner")
	requester := createFollowRequestUser(t, ctx, queries, "accept-requester", "Requester", "accept_requester")
	createSearchSessionForMe(t, ctx, queries, owner.ID, "tok")
	if _, err := queries.CreateUserFollow(ctx, db.CreateUserFollowParams{
		FollowerID: requester.ID, FolloweeID: owner.ID, Status: "pending",
	}); err != nil {
		t.Fatalf("CreateUserFollow: %v", err)
	}
	notifier := &followRequestNotificationSender{}
	ts := setupFollowRequests(t, queries, notifier)
	defer ts.Close()

	resp := postFollowRequestAction(t, ts, requester.ID, "accept")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	follow, err := queries.GetUserFollow(ctx, db.GetUserFollowParams{
		FollowerID: requester.ID,
		FolloweeID: owner.ID,
	})
	if err != nil {
		t.Fatalf("GetUserFollow: %v", err)
	}
	if follow.Status != "accepted" || follow.AcceptedAt == nil {
		t.Fatalf("follow = %#v, want accepted with timestamp", follow)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.userID != requester.ID || call.payload.Kind != "follow.accepted" || call.payload.URL != "/accept_owner" {
		t.Fatalf("notification = %#v, want requester follow.accepted at owner profile", call)
	}

	missing := postFollowRequestAction(t, ts, "missing-requester", "accept")
	missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404", missing.StatusCode)
	}
}

func TestDeclineFollowRequest_DeletesWithoutNotification(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	owner := createFollowRequestUser(t, ctx, queries, "decline-owner", "Owner", "decline_owner")
	requester := createFollowRequestUser(t, ctx, queries, "decline-requester", "Requester", "decline_requester")
	createSearchSessionForMe(t, ctx, queries, owner.ID, "tok")
	if _, err := queries.CreateUserFollow(ctx, db.CreateUserFollowParams{
		FollowerID: requester.ID, FolloweeID: owner.ID, Status: "pending",
	}); err != nil {
		t.Fatalf("CreateUserFollow: %v", err)
	}
	notifier := &followRequestNotificationSender{}
	ts := setupFollowRequests(t, queries, notifier)
	defer ts.Close()

	resp := postFollowRequestAction(t, ts, requester.ID, "decline")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if _, err := queries.GetUserFollow(ctx, db.GetUserFollowParams{
		FollowerID: requester.ID,
		FolloweeID: owner.ID,
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetUserFollow error = %v, want sql.ErrNoRows", err)
	}
	if len(notifier.calls) != 0 {
		t.Fatalf("notification calls = %d, want 0", len(notifier.calls))
	}
}

func TestGetMe_IncludesPendingFollowRequestCount(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	owner := createFollowRequestUser(t, ctx, queries, "count-owner", "Owner", "count_owner")
	createSearchSessionForMe(t, ctx, queries, owner.ID, "tok")
	for _, id := range []string{"count-requester-a", "count-requester-b"} {
		requester := createFollowRequestUser(t, ctx, queries, id, "Requester", id)
		if _, err := queries.CreateUserFollow(ctx, db.CreateUserFollowParams{
			FollowerID: requester.ID, FolloweeID: owner.ID, Status: "pending",
		}); err != nil {
			t.Fatalf("CreateUserFollow: %v", err)
		}
	}

	ts := setupFollowRequests(t, queries, &followRequestNotificationSender{})
	defer ts.Close()
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/me/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		PendingCount int64 `json:"pending_follow_request_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.PendingCount != 2 {
		t.Fatalf("pending count = %d, want 2", body.PendingCount)
	}
}

func postFollowRequestAction(t *testing.T, ts *httptest.Server, userID, action string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/me/follow-requests/"+userID+"/"+action, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

type followRequestPage struct {
	Items []struct {
		ID string `json:"id"`
	} `json:"items"`
	Total      int64   `json:"total"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

func getFollowRequestPage(t *testing.T, ts *httptest.Server, query string) followRequestPage {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/me/follow-requests"+query, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var page followRequestPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	return page
}

func createFollowRequestUser(t *testing.T, ctx context.Context, queries *db.Queries, id, displayName, username string) db.User {
	t.Helper()
	googleID := "google-" + id
	user, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID: id, Email: id + "@example.com", DisplayName: displayName, GoogleID: &googleID,
	})
	if err != nil {
		t.Fatalf("create user %q: %v", id, err)
	}
	user, err = queries.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
		ID: user.ID, DisplayName: user.DisplayName, Username: &username,
	})
	if err != nil {
		t.Fatalf("update user %q: %v", id, err)
	}
	return user
}

func createSearchSessionForMe(t *testing.T, ctx context.Context, queries *db.Queries, userID, token string) {
	t.Helper()
	if err := queries.CreateSession(ctx, db.CreateSessionParams{
		Token: token, UserID: userID, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil && err != sql.ErrNoRows {
		t.Fatalf("create session: %v", err)
	}
}
