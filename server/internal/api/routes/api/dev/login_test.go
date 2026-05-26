package dev_test

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

	"openplays/server/internal/api/routes/api/dev"
	"openplays/server/internal/db"
)

type fakeLoginStore struct {
	user           db.User
	getErr         error
	createdSession db.CreateSessionParams
}

func (f *fakeLoginStore) GetUserByID(_ context.Context, _ string) (db.User, error) {
	if f.getErr != nil {
		return db.User{}, f.getErr
	}
	return f.user, nil
}

func (f *fakeLoginStore) CreateSession(_ context.Context, arg db.CreateSessionParams) error {
	f.createdSession = arg
	return nil
}

func TestRegister_DisabledDoesNotRegisterLogin(t *testing.T) {
	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	dev.Register(api, db.New(&fakeDB{}), dev.Config{Enabled: false})

	req := httptest.NewRequest(http.MethodPost, "/dev/login", strings.NewReader(`{"user_id":"seed-host"}`))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestLogin_CreatesSessionCookie(t *testing.T) {
	now := time.Now()
	store := &fakeLoginStore{user: db.User{
		ID:          "seed-host",
		Email:       "seed-host@example.test",
		DisplayName: "Seed Host",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}}

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/dev")
	dev.RegisterLogin(grp, store, dev.Config{Enabled: true, CookieSecure: false})

	req := httptest.NewRequest(http.MethodPost, "/dev/login", strings.NewReader(`{"user_id":"seed-host"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}
	if store.createdSession.UserID != "seed-host" || store.createdSession.Token == "" {
		t.Fatalf("created session = %#v", store.createdSession)
	}
	cookie := rec.Result().Cookies()[0]
	if cookie.Name != "session" || cookie.Value == "" || cookie.Secure {
		t.Fatalf("cookie = %#v", cookie)
	}

	var out struct {
		SessionToken string `json:"session_token"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.SessionToken == "" || out.User.ID != "seed-host" {
		t.Fatalf("response = %#v", out)
	}
}

func TestLogin_UnknownUserReturns404(t *testing.T) {
	store := &fakeLoginStore{getErr: sql.ErrNoRows}
	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/dev")
	dev.RegisterLogin(grp, store, dev.Config{Enabled: true})

	req := httptest.NewRequest(http.MethodPost, "/dev/login", strings.NewReader(`{"user_id":"missing"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

type fakeDB struct{}

func (f *fakeDB) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (f *fakeDB) PrepareContext(context.Context, string) (*sql.Stmt, error) { return nil, nil }
func (f *fakeDB) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, nil
}
func (f *fakeDB) QueryRowContext(context.Context, string, ...interface{}) *sql.Row { return nil }
