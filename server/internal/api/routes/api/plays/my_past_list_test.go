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
	"openplays/server/internal/testdb"
)

func setupMyPastListTest(authStore *fakeAuthStore, store *db.Queries) *httptest.Server {
	svc := auth.NewService(authStore)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api")
	plays.RegisterMyPastList(grp, store, authmw.RequireAuth(api, svc))

	return httptest.NewServer(r)
}

type myPastListPage struct {
	Items []struct {
		ID          string  `json:"id"`
		StartsAt    string  `json:"starts_at"`
		CancelledAt *string `json:"cancelled_at"`
		ViewerState *string `json:"viewer_state"`
	} `json:"items"`
	Total      int64   `json:"total"`
	NextCursor *string `json:"next_cursor"`
}

func getMyPastPlays(t *testing.T, ts *httptest.Server, query string) myPastListPage {
	t.Helper()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/me/plays/past"+query, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var page myPastListPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	return page
}

func TestListMyPastPlays(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	userID := createRouteTestUser(t, ctx, queries, "past-list-user")
	otherID := createRouteTestUser(t, ctx, queries, "past-list-other")

	// Two ended plays, one cancelled upcoming play, one live upcoming play,
	// and someone else's ended play
	oldPlayID := createUserPlay(t, ctx, queries, userID, 4, nil, nil)
	setPlayEndsAt(t, ctx, sqlDB, oldPlayID, time.Now().UTC().Add(-30*24*time.Hour))
	recentPlayID := createUserPlay(t, ctx, queries, userID, 4, nil, nil)
	setPlayEndsAt(t, ctx, sqlDB, recentPlayID, time.Now().UTC().Add(-3*24*time.Hour))
	cancelledPlayID := createUserPlay(t, ctx, queries, userID, 4, nil, nil)
	if _, err := queries.CancelUserCreatedPlay(ctx, db.CancelUserCreatedPlayParams{
		ID:          cancelledPlayID,
		CancelledBy: &userID,
	}); err != nil {
		t.Fatalf("cancel play: %v", err)
	}
	upcomingPlayID := createUserPlay(t, ctx, queries, userID, 4, nil, nil)
	otherPastPlayID := createUserPlay(t, ctx, queries, otherID, 4, nil, nil)
	setPlayEndsAt(t, ctx, sqlDB, otherPastPlayID, time.Now().UTC().Add(-10*24*time.Hour))

	ts := setupMyPastListTest(sessionWithProfile(userID, nil), queries)
	defer ts.Close()

	page := getMyPastPlays(t, ts, "")
	if page.Total != 3 {
		t.Fatalf("total = %d, want 3 (two ended + one cancelled)", page.Total)
	}
	if len(page.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(page.Items))
	}
	// Newest first: the cancelled play still has its future starts_at
	if page.Items[0].ID != cancelledPlayID || page.Items[0].CancelledAt == nil {
		t.Fatalf("first item = %+v, want the cancelled play with cancelled_at set", page.Items[0])
	}
	if page.Items[1].ID != recentPlayID || page.Items[2].ID != oldPlayID {
		t.Fatalf("order = %s, %s; want recent then old", page.Items[1].ID, page.Items[2].ID)
	}
	for _, item := range page.Items {
		if item.ID == upcomingPlayID {
			t.Fatal("upcoming play leaked into the past list")
		}
		if item.ID == otherPastPlayID {
			t.Fatal("another user's play leaked into the past list")
		}
		if item.ViewerState == nil || *item.ViewerState != "creator" {
			t.Fatalf("viewer_state = %v, want creator", item.ViewerState)
		}
	}

	// Cursor pagination walks the same order
	firstPage := getMyPastPlays(t, ts, "?limit=2")
	if len(firstPage.Items) != 2 || firstPage.NextCursor == nil {
		t.Fatalf("first page = %d items, cursor %v", len(firstPage.Items), firstPage.NextCursor)
	}
	secondPage := getMyPastPlays(t, ts, "?limit=2&cursor="+*firstPage.NextCursor)
	if len(secondPage.Items) != 1 || secondPage.Items[0].ID != oldPlayID {
		t.Fatalf("second page = %+v, want just the oldest play", secondPage.Items)
	}
}
