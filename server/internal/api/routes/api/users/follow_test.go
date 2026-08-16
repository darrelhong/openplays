package users_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/routes/api/users"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/notifications"
	"openplays/server/internal/testdb"
)

type followNotificationCall struct {
	userID  string
	payload notifications.Payload
}

type followNotificationSender struct {
	calls []followNotificationCall
}

func (f *followNotificationSender) Notify(_ context.Context, userID string, payload notifications.Payload) error {
	f.calls = append(f.calls, followNotificationCall{userID: userID, payload: payload})
	return nil
}

func setupFollowTest(t *testing.T, queries *db.Queries, notifier notifications.Sender) *httptest.Server {
	t.Helper()
	svc := auth.NewService(queries)
	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api")
	users.Register(grp, queries, svc, notifier)
	return httptest.NewServer(r)
}

func TestFollowUser_AcceptsImmediatelyAndNotifiesFollowee(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	follower := createSearchTestUser(t, ctx, queries, "follow-follower", "Follower", strPtr("follower"), nil)
	followee := createSearchTestUser(t, ctx, queries, "follow-followee", "Followee", strPtr("followee"), nil)
	createSearchSession(t, ctx, queries, follower.ID, "tok")
	notifier := &followNotificationSender{}
	ts := setupFollowTest(t, queries, notifier)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/users/followee/follow", nil)
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
	var out struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Status != "accepted" {
		t.Fatalf("status = %q, want accepted", out.Status)
	}

	follow, err := queries.GetUserFollow(ctx, db.GetUserFollowParams{
		FollowerID: follower.ID,
		FolloweeID: followee.ID,
	})
	if err != nil {
		t.Fatalf("GetUserFollow: %v", err)
	}
	if follow.Status != "accepted" {
		t.Fatalf("stored status = %q, want accepted", follow.Status)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.userID != followee.ID {
		t.Fatalf("notification recipient = %q, want %q", call.userID, followee.ID)
	}
	if call.payload.Kind != "follow.started" {
		t.Fatalf("notification kind = %q, want follow.started", call.payload.Kind)
	}
	if call.payload.URL != "/follower" {
		t.Fatalf("notification url = %q, want /follower", call.payload.URL)
	}
}

func TestFollowUser_RestrictedProfileCreatesPendingRequest(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	follower := createSearchTestUser(t, ctx, queries, "follow-pending-follower", "Pending Follower", strPtr("pending_follower"), nil)
	followee := createSearchTestUser(t, ctx, queries, "follow-pending-followee", "Pending Followee", strPtr("pending_followee"), nil)
	if _, err := sqlDB.ExecContext(ctx, "UPDATE users SET restrict_follow = 1 WHERE id = ?", followee.ID); err != nil {
		t.Fatalf("restrict followee: %v", err)
	}
	createSearchSession(t, ctx, queries, follower.ID, "tok")
	notifier := &followNotificationSender{}
	ts := setupFollowTest(t, queries, notifier)
	defer ts.Close()

	resp := requestFollow(t, ts, http.MethodPut, "pending_followee")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Status != "pending" {
		t.Fatalf("status = %q, want pending", out.Status)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	if notifier.calls[0].userID != followee.ID || notifier.calls[0].payload.Kind != "follow.requested" {
		t.Fatalf("notification = %#v, want followee follow.requested", notifier.calls[0])
	}
	if notifier.calls[0].payload.URL != "/profile/follow-requests" {
		t.Fatalf("notification url = %q, want follow request inbox", notifier.calls[0].payload.URL)
	}
}

func TestFollowUser_RejectsSelfFollow(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	user := createSearchTestUser(t, ctx, queries, "follow-self", "Self Follower", strPtr("self_follower"), nil)
	createSearchSession(t, ctx, queries, user.ID, "tok")
	ts := setupFollowTest(t, queries, nil)
	defer ts.Close()

	resp := requestFollow(t, ts, http.MethodPut, "self_follower")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestFollowUser_RejectsMutuallyBlockedUser(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	follower := createSearchTestUser(t, ctx, queries, "follow-blocked-follower", "Blocked Follower", strPtr("blocked_follower"), nil)
	followee := createSearchTestUser(t, ctx, queries, "follow-blocked-followee", "Blocked Followee", strPtr("blocked_followee"), nil)
	if err := queries.CreateBlock(ctx, db.CreateBlockParams{BlockerID: follower.ID, BlockedID: followee.ID}); err != nil {
		t.Fatalf("CreateBlock: %v", err)
	}
	createSearchSession(t, ctx, queries, follower.ID, "tok")
	ts := setupFollowTest(t, queries, nil)
	defer ts.Close()

	resp := requestFollow(t, ts, http.MethodPut, "blocked_followee")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestFollowUser_RepeatedPutIsNoOp(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	follower := createSearchTestUser(t, ctx, queries, "follow-repeat-follower", "Repeat Follower", strPtr("repeat_follower"), nil)
	createSearchTestUser(t, ctx, queries, "follow-repeat-followee", "Repeat Followee", strPtr("repeat_followee"), nil)
	createSearchSession(t, ctx, queries, follower.ID, "tok")
	notifier := &followNotificationSender{}
	ts := setupFollowTest(t, queries, notifier)
	defer ts.Close()

	first := requestFollow(t, ts, http.MethodPut, "repeat_followee")
	first.Body.Close()
	if first.StatusCode != http.StatusOK {
		t.Fatalf("first status = %d, want 200", first.StatusCode)
	}
	second := requestFollow(t, ts, http.MethodPut, "repeat_followee")
	defer second.Body.Close()
	if second.StatusCode != http.StatusOK {
		t.Fatalf("second status = %d, want 200", second.StatusCode)
	}
	var out struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(second.Body).Decode(&out); err != nil {
		t.Fatalf("decode repeated response: %v", err)
	}
	if out.Status != "accepted" {
		t.Fatalf("repeated status = %q, want accepted", out.Status)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
}

func TestUnfollowUser_NonexistentFollowSucceeds(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()
	follower := createSearchTestUser(t, ctx, queries, "unfollow-missing-follower", "Missing Follower", strPtr("missing_follower"), nil)
	createSearchTestUser(t, ctx, queries, "unfollow-missing-followee", "Missing Followee", strPtr("missing_followee"), nil)
	createSearchSession(t, ctx, queries, follower.ID, "tok")
	ts := setupFollowTest(t, queries, nil)
	defer ts.Close()

	resp := requestFollow(t, ts, http.MethodDelete, "missing_followee")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func requestFollow(t *testing.T, ts *httptest.Server, method, username string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, ts.URL+"/api/users/"+username+"/follow", nil)
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
