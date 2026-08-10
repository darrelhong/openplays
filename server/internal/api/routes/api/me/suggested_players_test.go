package me_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/me"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
)

type fakeSuggestedPlayersStore struct {
	rows []db.ListSuggestedPlayersRow
	err  error
	args db.ListSuggestedPlayersParams
}

func (f *fakeSuggestedPlayersStore) ListSuggestedPlayers(_ context.Context, arg db.ListSuggestedPlayersParams) ([]db.ListSuggestedPlayersRow, error) {
	f.args = arg
	return f.rows, f.err
}

func setupSuggestedPlayers(authStore *fakeAuthStore, store *fakeSuggestedPlayersStore) *httptest.Server {
	svc := auth.NewService(authStore)
	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api/me")
	grp.UseMiddleware(authmw.RequireAuth(api, svc))
	me.RegisterSuggestedPlayers(grp, store)
	return httptest.NewServer(r)
}

func TestListSuggestedPlayers_ReturnsRatingAndFilters(t *testing.T) {
	sportProfile := `{"tennis":{"level":"4.0"}}`
	store := &fakeSuggestedPlayersStore{rows: []db.ListSuggestedPlayersRow{{
		ID: "suggested-player-1", DisplayName: "Suggested Player One", SportsProfile: &sportProfile,
	}}}
	ts := setupSuggestedPlayers(activeSession(), store)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/me/suggested-players?sport=tennis&exclude_play_id=play-1&limit=5", nil)
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
		Items []struct {
			ID         string  `json:"id"`
			RatingCode *string `json:"rating_code"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != "suggested-player-1" || body.Items[0].RatingCode == nil || *body.Items[0].RatingCode != "4.0" {
		t.Fatalf("items = %#v, want tennis rating 4.0", body.Items)
	}
	if store.args.UserID != "user-1" || store.args.ExcludePlayID != "play-1" || store.args.Limit != 5 {
		t.Fatalf("query args = %#v, want authenticated user, play exclusion, and limit", store.args)
	}
}

func TestListSuggestedPlayers_InvalidSport(t *testing.T) {
	store := &fakeSuggestedPlayersStore{}
	ts := setupSuggestedPlayers(activeSession(), store)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/me/suggested-players?sport=golf", nil)
	if err != nil {
		t.Fatal(err)
	}
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

func TestListSuggestedPlayers_StoreError(t *testing.T) {
	store := &fakeSuggestedPlayersStore{err: errors.New("database unavailable")}
	ts := setupSuggestedPlayers(activeSession(), store)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/me/suggested-players", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}
