package users_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"openplays/server/internal/model"
)

type shoutoutPage struct {
	Items []struct {
		Shoutout            string `json:"shoutout"`
		ReviewerDisplayName string `json:"reviewer_display_name"`
		PlayID              string `json:"play_id"`
	} `json:"items"`
	Total      int64   `json:"total"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

func getShoutoutPage(t *testing.T, url string) (shoutoutPage, []byte) {
	t.Helper()

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var buf strings.Builder
	var page shoutoutPage
	if err := json.NewDecoder(io.TeeReader(resp.Body, &buf)).Decode(&page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	return page, []byte(buf.String())
}

func TestListUserShoutouts_PaginatesNewestFirst(t *testing.T) {
	ts, queries := setupSearchTest(t)
	defer ts.Close()
	ctx := context.Background()

	viewer := createSearchTestUser(t, ctx, queries, "sh-viewer", "Viewer", strPtr("sh_viewer"), nil)
	createSearchSession(t, ctx, queries, viewer.ID, "tok")
	username := "sh_target"
	target := createSearchTestUser(t, ctx, queries, "sh-target", "Shout Target", &username, nil)
	writerA := createSearchTestUser(t, ctx, queries, "sh-writer-a", "Writer A", strPtr("sh_writer_a"), nil)
	writerB := createSearchTestUser(t, ctx, queries, "sh-writer-b", "Writer B", strPtr("sh_writer_b"), nil)

	playID := createProfileTestPlayWithSport(t, ctx, queries, "sh-play", target.ID, model.SportBadminton)
	// Same-second created_at: ordering falls back to id DESC (newest insert first)
	seedProfileReview(t, ctx, queries, playID, viewer.ID, target.ID, nil, `[]`, strPtr("first shoutout"))
	seedProfileReview(t, ctx, queries, playID, writerA.ID, target.ID, int64Ptr(5), `[]`, strPtr("second shoutout"))
	seedProfileReview(t, ctx, queries, playID, writerB.ID, target.ID, int64Ptr(4), `[]`, strPtr("third shoutout"))

	pageOne, rawOne := getShoutoutPage(t, ts.URL+"/api/users/sh_target/shoutouts?limit=2")
	if pageOne.Total != 3 || len(pageOne.Items) != 2 || !pageOne.HasMore || pageOne.NextCursor == nil {
		t.Fatalf("page one = %+v", pageOne)
	}
	if pageOne.Items[0].Shoutout != "third shoutout" || pageOne.Items[1].Shoutout != "second shoutout" {
		t.Fatalf("page one order = %q, %q", pageOne.Items[0].Shoutout, pageOne.Items[1].Shoutout)
	}
	if pageOne.Items[0].ReviewerDisplayName != "Writer B" || pageOne.Items[0].PlayID != playID {
		t.Fatalf("attribution = %+v", pageOne.Items[0])
	}
	// The anonymity boundary holds here too: no rating values in the payload
	if strings.Contains(string(rawOne), `"rating"`) {
		t.Fatal("shoutout page leaked a rating")
	}

	pageTwo, _ := getShoutoutPage(t, ts.URL+"/api/users/sh_target/shoutouts?limit=2&cursor="+*pageOne.NextCursor)
	if len(pageTwo.Items) != 1 || pageTwo.HasMore || pageTwo.Items[0].Shoutout != "first shoutout" {
		t.Fatalf("page two = %+v", pageTwo)
	}
}

func TestListUserShoutouts_AuthAndMissingUser(t *testing.T) {
	ts, queries := setupSearchTest(t)
	defer ts.Close()
	ctx := context.Background()

	viewer := createSearchTestUser(t, ctx, queries, "sh-auth-viewer", "Viewer", strPtr("sh_auth_viewer"), nil)
	createSearchSession(t, ctx, queries, viewer.ID, "tok")

	// Unauthenticated requests never see review data
	unauth, err := http.Get(ts.URL + "/api/users/sh_auth_viewer/shoutouts")
	if err != nil {
		t.Fatal(err)
	}
	unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", unauth.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/users/no_such_user/shoutouts", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing user status = %d, want 404", resp.StatusCode)
	}
}
