package users_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"openplays/server/internal/db"
)

func TestFollowLists_OnlyOwnerOrAcceptedFollowerCanView(t *testing.T) {
	ts, queries := setupSearchTest(t)
	defer ts.Close()
	ctx := context.Background()

	owner := createSearchTestUser(t, ctx, queries, "follow-list-owner", "List Owner", strPtr("list_owner"), nil)
	acceptedFollower := createSearchTestUser(t, ctx, queries, "follow-list-follower", "Accepted Follower", strPtr("list_follower"), nil)
	pendingRequester := createSearchTestUser(t, ctx, queries, "follow-list-pending", "Pending Requester", strPtr("list_pending"), nil)
	stranger := createSearchTestUser(t, ctx, queries, "follow-list-stranger", "Stranger", strPtr("list_stranger"), nil)
	followingTarget := createSearchTestUser(t, ctx, queries, "follow-list-following", "Following Target", strPtr("list_following"), nil)
	if _, err := queries.CreateUserFollow(ctx, db.CreateUserFollowParams{
		FollowerID: acceptedFollower.ID, FolloweeID: owner.ID, Status: "accepted",
	}); err != nil {
		t.Fatalf("create accepted follower: %v", err)
	}
	if _, err := queries.CreateUserFollow(ctx, db.CreateUserFollowParams{
		FollowerID: pendingRequester.ID, FolloweeID: owner.ID, Status: "pending",
	}); err != nil {
		t.Fatalf("create pending requester: %v", err)
	}
	if _, err := queries.CreateUserFollow(ctx, db.CreateUserFollowParams{
		FollowerID: owner.ID, FolloweeID: followingTarget.ID, Status: "accepted",
	}); err != nil {
		t.Fatalf("create following target: %v", err)
	}
	createSearchSession(t, ctx, queries, owner.ID, "owner-token")
	createSearchSession(t, ctx, queries, acceptedFollower.ID, "follower-token")
	createSearchSession(t, ctx, queries, pendingRequester.ID, "pending-token")
	createSearchSession(t, ctx, queries, stranger.ID, "stranger-token")

	for _, token := range []string{"owner-token", "follower-token"} {
		followers := getFollowList(t, ts, token, "list_owner", "followers", http.StatusOK)
		if len(followers.Items) != 1 || followers.Items[0].ID != acceptedFollower.ID {
			t.Fatalf("followers for %s = %#v, want accepted follower", token, followers.Items)
		}
		following := getFollowList(t, ts, token, "list_owner", "following", http.StatusOK)
		if len(following.Items) != 1 || following.Items[0].ID != followingTarget.ID {
			t.Fatalf("following for %s = %#v, want following target", token, following.Items)
		}
	}

	for _, token := range []string{"pending-token", "stranger-token"} {
		getFollowList(t, ts, token, "list_owner", "followers", http.StatusForbidden)
		getFollowList(t, ts, token, "list_owner", "following", http.StatusForbidden)
	}

	profileReq, err := http.NewRequest(http.MethodGet, ts.URL+"/api/users/list_owner", nil)
	if err != nil {
		t.Fatal(err)
	}
	profileReq.AddCookie(&http.Cookie{Name: "session", Value: "stranger-token"})
	profileResp, err := http.DefaultClient.Do(profileReq)
	if err != nil {
		t.Fatal(err)
	}
	defer profileResp.Body.Close()
	if profileResp.StatusCode != http.StatusOK {
		t.Fatalf("profile status = %d, want 200", profileResp.StatusCode)
	}
	var profile struct {
		FollowerCount  int64 `json:"follower_count"`
		FollowingCount int64 `json:"following_count"`
	}
	if err := json.NewDecoder(profileResp.Body).Decode(&profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	if profile.FollowerCount != 1 || profile.FollowingCount != 1 {
		t.Fatalf("profile counts = %#v, want follower/following 1/1", profile)
	}
}

func TestFollowLists_PaginateWithTextUserCursor(t *testing.T) {
	ts, queries := setupSearchTest(t)
	defer ts.Close()
	ctx := context.Background()
	owner := createSearchTestUser(t, ctx, queries, "follow-cursor-owner", "Cursor Owner", strPtr("cursor_owner"), nil)
	createSearchSession(t, ctx, queries, owner.ID, "cursor-token")
	for _, id := range []string{"cursor-follower-a", "cursor-follower-b", "cursor-follower-c"} {
		follower := createSearchTestUser(t, ctx, queries, id, id, strPtr(id), nil)
		if _, err := queries.CreateUserFollow(ctx, db.CreateUserFollowParams{
			FollowerID: follower.ID, FolloweeID: owner.ID, Status: "accepted",
		}); err != nil {
			t.Fatalf("create follow %s: %v", id, err)
		}
	}

	first := getFollowList(t, ts, "cursor-token", "cursor_owner", "followers", http.StatusOK, "?limit=2")
	if len(first.Items) != 2 || first.Items[0].ID != "cursor-follower-c" || first.Items[1].ID != "cursor-follower-b" {
		t.Fatalf("first page = %#v, want c,b", first.Items)
	}
	if first.NextCursor == nil || !first.HasMore {
		t.Fatalf("first page metadata = %#v, want next cursor", first)
	}
	second := getFollowList(t, ts, "cursor-token", "cursor_owner", "followers", http.StatusOK, "?limit=2&cursor="+url.QueryEscape(*first.NextCursor))
	if len(second.Items) != 1 || second.Items[0].ID != "cursor-follower-a" || second.NextCursor != nil || second.HasMore {
		t.Fatalf("second page = %#v, want only a and no next page", second)
	}
}

type followListPage struct {
	Items []struct {
		ID string `json:"id"`
	} `json:"items"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

func getFollowList(t *testing.T, ts *httptest.Server, token, username, list string, wantStatus int, query ...string) followListPage {
	t.Helper()
	path := ts.URL + "/api/users/" + username + "/" + list
	if len(query) > 0 {
		path += query[0]
	}
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s status = %d, want %d", list, resp.StatusCode, wantStatus)
	}
	if wantStatus == http.StatusForbidden {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read %s error response: %v", list, err)
		}
		if strings.Contains(string(body), "only followers can see this") {
			t.Fatalf("%s error reveals follow-list visibility rule: %s", list, body)
		}
		return followListPage{}
	}
	var page followListPage
	if wantStatus == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			t.Fatalf("decode %s: %v", list, err)
		}
	}
	return page
}
