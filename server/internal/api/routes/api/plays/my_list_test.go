package plays_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/plays"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/testdb"
)

func setupMyListTest(authStore *fakeAuthStore, store *db.Queries) *httptest.Server {
	svc := auth.NewService(authStore)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api")
	plays.RegisterMyList(grp, store, authmw.RequireAuth(api, svc))

	return httptest.NewServer(r)
}

func TestListMyUpcomingPlays_IncludesInvitedPlay(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	hostID := createRouteTestUser(t, ctx, queries, "my-list-invited-host")
	viewerID := createRouteTestUser(t, ctx, queries, "my-list-invited-viewer")
	playID := createUserPlay(t, ctx, queries, hostID, 4, nil, nil)
	seedInvitedParticipant(t, ctx, queries, playID, viewerID)

	ts := setupMyListTest(sessionWithProfile(viewerID, nil), queries)
	defer ts.Close()
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/me/plays", nil)
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

	var page struct {
		Items []struct {
			ID          string  `json:"id"`
			ViewerState *string `json:"viewer_state"`
		} `json:"items"`
		Total int64 `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("page total/items = %d/%d, want 1/1", page.Total, len(page.Items))
	}
	if page.Items[0].ID != playID {
		t.Fatalf("play id = %q, want %q", page.Items[0].ID, playID)
	}
	if page.Items[0].ViewerState == nil || *page.Items[0].ViewerState != "invited" {
		t.Fatalf("viewer_state = %v, want invited", page.Items[0].ViewerState)
	}
}
