package me_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/api/routes/api/me"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
)

// fakeProfileStore implements me.ProfileStore at the DB seam.
type fakeProfileStore struct {
	updated db.User
	err     error
}

func (f *fakeProfileStore) UpdateUserProfile(_ context.Context, arg db.UpdateUserProfileParams) (db.User, error) {
	if f.err != nil {
		return db.User{}, f.err
	}
	u := f.updated
	u.DisplayName = arg.DisplayName
	u.Username = arg.Username
	return u, nil
}

// fakeAuthStore satisfies auth.Store.
type fakeAuthStore struct {
	sessionRow db.GetSessionWithUserRow
	sessionErr error
}

func (f *fakeAuthStore) UpsertUserByGoogleID(_ context.Context, _ db.UpsertUserByGoogleIDParams) (db.User, error) {
	return db.User{}, nil
}
func (f *fakeAuthStore) UpsertUserByFacebookID(_ context.Context, _ db.UpsertUserByFacebookIDParams) (db.User, error) {
	return db.User{}, nil
}
func (f *fakeAuthStore) LinkGoogleID(_ context.Context, _ db.LinkGoogleIDParams) (db.User, error) {
	return db.User{}, nil
}
func (f *fakeAuthStore) LinkFacebookID(_ context.Context, _ db.LinkFacebookIDParams) (db.User, error) {
	return db.User{}, nil
}
func (f *fakeAuthStore) GetSessionWithUser(_ context.Context, _ string) (db.GetSessionWithUserRow, error) {
	return f.sessionRow, f.sessionErr
}
func (f *fakeAuthStore) CreateSession(_ context.Context, _ db.CreateSessionParams) error { return nil }
func (f *fakeAuthStore) DeleteSession(_ context.Context, _ string) error                 { return nil }
func (f *fakeAuthStore) RefreshSession(_ context.Context, _ db.RefreshSessionParams) error {
	return nil
}

func setup(authStore *fakeAuthStore, profileStore *fakeProfileStore) *httptest.Server {
	svc := auth.NewService(authStore)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))

	// Mirror production setup: group + middleware
	grp := huma.NewGroup(api, "/api/me")
	grp.UseMiddleware(authmw.RequireAuth(api, svc))
	me.RegisterGet(grp)
	me.RegisterUpdate(grp, profileStore)

	return httptest.NewServer(r)
}

func activeSession() *fakeAuthStore {
	now := time.Now()
	return &fakeAuthStore{
		sessionRow: db.GetSessionWithUserRow{
			Token: "tok", UserID: "user-1", ExpiresAt: now.Add(time.Hour),
			UserID2: "user-1", Email: "test@test.com", DisplayName: "Test User",
			Status: "active", CreatedAt: now, UpdatedAt: now,
		},
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	now := time.Now()
	profileStore := &fakeProfileStore{
		updated: db.User{
			ID: "user-1", Email: "test@test.com", Status: "active",
			CreatedAt: now, UpdatedAt: now,
		},
	}
	ts := setup(activeSession(), profileStore)
	defer ts.Close()

	body := `{"display_name":"New Name","username":"newuser"}`
	req, _ := http.NewRequest("PATCH", ts.URL+"/api/me/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestUpdateProfile_UsernameTaken_Returns409(t *testing.T) {
	profileStore := &fakeProfileStore{
		err: errors.New("UNIQUE constraint failed: users.username"),
	}
	ts := setup(activeSession(), profileStore)
	defer ts.Close()

	body := `{"display_name":"Test","username":"taken"}`
	req, _ := http.NewRequest("PATCH", ts.URL+"/api/me/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}

func TestUpdateProfile_NoAuth_Returns401(t *testing.T) {
	profileStore := &fakeProfileStore{}
	authStore := &fakeAuthStore{sessionErr: errors.New("no session")}
	ts := setup(authStore, profileStore)
	defer ts.Close()

	body := `{"display_name":"Test"}`
	req, _ := http.NewRequest("PATCH", ts.URL+"/api/me/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No cookie

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

// Compile-time check: *db.Queries satisfies ProfileStore
var _ me.ProfileStore = (*db.Queries)(nil)

func TestUpdateProfile_EmptyDisplayName_Returns422(t *testing.T) {
	ts := setup(activeSession(), &fakeProfileStore{})
	defer ts.Close()

	body := `{"display_name":"","username":"valid"}`
	req, _ := http.NewRequest("PATCH", ts.URL+"/api/me/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

func TestUpdateProfile_WhitespaceDisplayName_Returns422(t *testing.T) {
	ts := setup(activeSession(), &fakeProfileStore{})
	defer ts.Close()

	body := `{"display_name":"   ","username":"valid"}`
	req, _ := http.NewRequest("PATCH", ts.URL+"/api/me/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

func TestUpdateProfile_EmptyUsername_Returns422(t *testing.T) {
	ts := setup(activeSession(), &fakeProfileStore{})
	defer ts.Close()

	body := `{"display_name":"Valid Name","username":""}`
	req, _ := http.NewRequest("PATCH", ts.URL+"/api/me/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

func TestUpdateProfile_NullUsername_PreservesExisting(t *testing.T) {
	now := time.Now()
	existingUsername := "existing"
	authStore := &fakeAuthStore{
		sessionRow: db.GetSessionWithUserRow{
			Token: "tok", UserID: "user-1", ExpiresAt: now.Add(time.Hour),
			UserID2: "user-1", Email: "test@test.com", DisplayName: "Test",
			Username: &existingUsername, Status: "active",
			CreatedAt: now, UpdatedAt: now,
		},
	}
	profileStore := &fakeProfileStore{
		updated: db.User{
			ID: "user-1", Email: "test@test.com", Username: &existingUsername,
			Status: "active", CreatedAt: now, UpdatedAt: now,
		},
	}
	ts := setup(authStore, profileStore)
	defer ts.Close()

	// No username field in body — should keep existing
	body := `{"display_name":"Updated Name"}`
	req, _ := http.NewRequest("PATCH", ts.URL+"/api/me/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
