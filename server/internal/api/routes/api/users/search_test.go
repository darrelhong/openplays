package users_test

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

	"openplays/server/internal/api/routes/api/users"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/testdb"
)

func setupSearchTest(t *testing.T) (*httptest.Server, *db.Queries) {
	t.Helper()

	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	svc := auth.NewService(queries)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api")
	users.Register(grp, queries, svc)

	return httptest.NewServer(r), queries
}

func TestSearchUsers_ReturnsActiveMatchesWithSportRating(t *testing.T) {
	ts, queries := setupSearchTest(t)
	defer ts.Close()
	ctx := context.Background()

	searcher := createSearchTestUser(t, ctx, queries, "searcher", "Searcher", nil, nil)
	username := "alice"
	badminton := "HB"
	profileRaw := mustSportsProfileRaw(t, &model.SportsProfile{
		Badminton: &model.SportLevelProfile{Level: &badminton},
	})
	createSearchTestUser(t, ctx, queries, "alice", "Alice Tan", &username, profileRaw)
	createSearchTestUser(t, ctx, queries, "bob", "Bob Lee", nil, nil)
	inactive := createSearchTestUser(t, ctx, queries, "alicia", "Alicia Suspended", nil, nil)
	if err := queries.UpdateUserStatus(ctx, db.UpdateUserStatusParams{ID: inactive.ID, Status: "suspended"}); err != nil {
		t.Fatalf("suspend user: %v", err)
	}
	createSearchSession(t, ctx, queries, searcher.ID, "tok")

	req, _ := http.NewRequest("GET", ts.URL+"/api/users/search?q=ali&sport=badminton", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var page users.SearchPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %d, want 1: %#v", len(page.Items), page.Items)
	}
	item := page.Items[0]
	if item.ID != "alice" || item.DisplayName != "Alice Tan" {
		t.Fatalf("item = %#v, want Alice Tan", item)
	}
	if item.RatingCode == nil || *item.RatingCode != "HB" {
		t.Fatalf("rating_code = %v, want HB", item.RatingCode)
	}
}

func TestSearchUsers_NoAuthReturns401(t *testing.T) {
	ts, _ := setupSearchTest(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/users/search?q=ali")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestSearchUsers_InvalidSportReturns422(t *testing.T) {
	ts, queries := setupSearchTest(t)
	defer ts.Close()
	ctx := context.Background()

	searcher := createSearchTestUser(t, ctx, queries, "searcher", "Searcher", nil, nil)
	createSearchSession(t, ctx, queries, searcher.ID, "tok")

	req, _ := http.NewRequest("GET", ts.URL+"/api/users/search?q=ali&sport=chess", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func createSearchTestUser(t *testing.T, ctx context.Context, queries *db.Queries, id, displayName string, username *string, sportsProfile *string) db.User {
	t.Helper()

	googleID := "google-" + id
	user, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID:          id,
		Email:       id + "@example.com",
		DisplayName: displayName,
		GoogleID:    &googleID,
	})
	if err != nil {
		t.Fatalf("create user %q: %v", id, err)
	}
	if username != nil || sportsProfile != nil {
		user, err = queries.UpdateUserProfile(ctx, db.UpdateUserProfileParams{
			ID:            user.ID,
			DisplayName:   user.DisplayName,
			Username:      username,
			SportsProfile: sportsProfile,
		})
		if err != nil {
			t.Fatalf("update user %q profile: %v", id, err)
		}
	}
	return user
}

func createSearchSession(t *testing.T, ctx context.Context, queries *db.Queries, userID, token string) {
	t.Helper()
	if err := queries.CreateSession(ctx, db.CreateSessionParams{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}
}

func mustSportsProfileRaw(t *testing.T, profile *model.SportsProfile) *string {
	t.Helper()
	raw, err := model.SportsProfileString(profile)
	if err != nil {
		t.Fatalf("SportsProfileString: %v", err)
	}
	return raw
}
