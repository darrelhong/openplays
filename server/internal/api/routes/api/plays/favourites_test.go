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

func setupFavouriteTest(authStore *fakeAuthStore, store *db.Queries) *httptest.Server {
	svc := auth.NewService(authStore)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/plays")
	plays.RegisterFavourite(grp, store, authmw.RequireAuth(api, svc))
	plays.RegisterMyFavourites(api, store, authmw.RequireAuth(api, svc))

	return httptest.NewServer(r)
}

func TestFavouritePlay_ListsUpcomingFavourite(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	viewerID := createRouteTestUser(t, ctx, queries, "favourite-viewer")
	ownerID := createRouteTestUser(t, ctx, queries, "favourite-owner")
	playID := createUserPlay(t, ctx, queries, ownerID, 4, ptrString("MB"), ptrString("HI"))

	ts := setupFavouriteTest(sessionWithProfile(viewerID, nil), queries)
	defer ts.Close()

	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/plays/"+playID+"/favourite", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("PUT favourite status = %d, want 204", resp.StatusCode)
		}
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/me/favourites", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET favourites status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		Items []struct {
			ID           string  `json:"id"`
			IsFavourited *bool   `json:"is_favourited"`
			ViewerState  *string `json:"viewer_state"`
		} `json:"items"`
		Total int64 `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Total != 1 || len(out.Items) != 1 {
		t.Fatalf("favourites = total %d len %d, want one item", out.Total, len(out.Items))
	}
	if out.Items[0].ID != playID {
		t.Fatalf("favourite id = %s, want %s", out.Items[0].ID, playID)
	}
	if out.Items[0].IsFavourited == nil || !*out.Items[0].IsFavourited {
		t.Fatalf("is_favourited = %v, want true", out.Items[0].IsFavourited)
	}
	if out.Items[0].ViewerState == nil || *out.Items[0].ViewerState != "not_joined" {
		t.Fatalf("viewer_state = %v, want not_joined", out.Items[0].ViewerState)
	}
}

func TestUnfavouritePlay_IsIdempotent(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	viewerID := createRouteTestUser(t, ctx, queries, "unfavourite-viewer")
	ownerID := createRouteTestUser(t, ctx, queries, "unfavourite-owner")
	playID := createUserPlay(t, ctx, queries, ownerID, 4, ptrString("MB"), ptrString("HI"))

	ts := setupFavouriteTest(sessionWithProfile(viewerID, nil), queries)
	defer ts.Close()

	putReq, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/plays/"+playID+"/favourite", nil)
	putReq.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatal(err)
	}
	putResp.Body.Close()
	if putResp.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT favourite status = %d, want 204", putResp.StatusCode)
	}

	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/plays/"+playID+"/favourite", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("DELETE favourite status = %d, want 204", resp.StatusCode)
		}
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/me/favourites", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET favourites status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		Items []struct{} `json:"items"`
		Total int64      `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Total != 0 || len(out.Items) != 0 {
		t.Fatalf("favourites = total %d len %d, want empty", out.Total, len(out.Items))
	}
}

func TestFavouritePlay_AllowsSellBookingListings(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	viewerID := createRouteTestUser(t, ctx, queries, "favourite-sell-booking-viewer")
	playID := createFavouriteRouteListing(t, ctx, queries, "favourite-sell-booking", model.ListingSellBooking, 24)

	ts := setupFavouriteTest(sessionWithProfile(viewerID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/plays/"+playID+"/favourite", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT favourite sell_booking status = %d, want 204", resp.StatusCode)
	}
}

func TestFavouritePlay_RejectsExpiredListing(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	viewerID := createRouteTestUser(t, ctx, queries, "favourite-expired-viewer")
	playID := createFavouriteRouteListing(t, ctx, queries, "favourite-expired", model.ListingPlay, -48)

	ts := setupFavouriteTest(sessionWithProfile(viewerID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/plays/"+playID+"/favourite", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("PUT favourite expired status = %d, want 404", resp.StatusCode)
	}
}

func TestListPlaysIncludesFavouriteStateForAuthenticatedViewer(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	viewerID := createRouteTestUser(t, ctx, queries, "favourite-list-viewer")
	ownerID := createRouteTestUser(t, ctx, queries, "favourite-list-owner")
	favouritedID := createUserPlay(t, ctx, queries, ownerID, 4, ptrString("MB"), ptrString("HI"))
	otherID := createUserPlay(t, ctx, queries, ownerID, 4, ptrString("MB"), ptrString("HI"))
	if err := queries.FavouritePlay(ctx, db.FavouritePlayParams{UserID: viewerID, PlayID: favouritedID}); err != nil {
		t.Fatalf("FavouritePlay: %v", err)
	}

	svc := auth.NewService(sessionWithProfile(viewerID, nil))
	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/plays")
	plays.RegisterList(grp, queries, authmw.OptionalAuth(api, svc))
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET plays status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		Items []struct {
			ID           string `json:"id"`
			IsFavourited *bool  `json:"is_favourited"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	states := map[string]*bool{}
	for _, item := range out.Items {
		states[item.ID] = item.IsFavourited
	}
	if states[favouritedID] == nil || !*states[favouritedID] {
		t.Fatalf("is_favourited for saved play = %v, want true", states[favouritedID])
	}
	if states[otherID] == nil || *states[otherID] {
		t.Fatalf("is_favourited for unsaved play = %v, want false", states[otherID])
	}
}

func TestGetPlayDetailIncludesFavouriteStateForAuthenticatedViewer(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	viewerID := createRouteTestUser(t, ctx, queries, "favourite-detail-viewer")
	ownerID := createRouteTestUser(t, ctx, queries, "favourite-detail-owner")
	playID := createUserPlay(t, ctx, queries, ownerID, 4, ptrString("MB"), ptrString("HI"))
	if err := queries.FavouritePlay(ctx, db.FavouritePlayParams{UserID: viewerID, PlayID: playID}); err != nil {
		t.Fatalf("FavouritePlay: %v", err)
	}

	ts := setupGetDetailTest(sessionWithProfile(viewerID, nil), queries)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/plays/"+playID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET play detail status = %d, want 200", resp.StatusCode)
	}

	var out struct {
		IsFavourited *bool `json:"is_favourited"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.IsFavourited == nil || !*out.IsFavourited {
		t.Fatalf("is_favourited = %v, want true", out.IsFavourited)
	}
}

func createFavouriteRouteListing(t *testing.T, ctx context.Context, queries *db.Queries, id string, listingType model.ListingType, startsInHours time.Duration) string {
	t.Helper()

	startsAt := time.Now().UTC().Add(startsInHours * time.Hour)
	play, err := queries.CreatePlay(ctx, db.CreatePlayParams{
		ID:          id,
		ListingType: listingType,
		Sport:       model.SportBadminton,
		HostName:    "Favourite Host",
		StartsAt:    startsAt,
		EndsAt:      startsAt.Add(2 * time.Hour),
		Timezone:    "Asia/Singapore",
		Venue:       "SBH",
		Currency:    "SGD",
	})
	if err != nil {
		t.Fatalf("CreatePlay %s: %v", id, err)
	}
	return play.ID
}
