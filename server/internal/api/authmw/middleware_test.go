package authmw_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
)

// fakeStore implements auth.Store for testing.
type fakeStore struct {
	sessionRow db.GetSessionWithUserRow
	sessionErr error
}

func (f *fakeStore) UpsertUserByGoogleID(_ context.Context, _ db.UpsertUserByGoogleIDParams) (db.User, error) {
	return db.User{}, nil
}
func (f *fakeStore) UpsertUserByFacebookID(_ context.Context, _ db.UpsertUserByFacebookIDParams) (db.User, error) {
	return db.User{}, nil
}
func (f *fakeStore) LinkGoogleID(_ context.Context, _ db.LinkGoogleIDParams) (db.User, error) {
	return db.User{}, nil
}
func (f *fakeStore) LinkFacebookID(_ context.Context, _ db.LinkFacebookIDParams) (db.User, error) {
	return db.User{}, nil
}
func (f *fakeStore) GetSessionWithUser(_ context.Context, _ string) (db.GetSessionWithUserRow, error) {
	return f.sessionRow, f.sessionErr
}
func (f *fakeStore) CreateSession(_ context.Context, _ db.CreateSessionParams) error  { return nil }
func (f *fakeStore) DeleteSession(_ context.Context, _ string) error                  { return nil }
func (f *fakeStore) RefreshSession(_ context.Context, _ db.RefreshSessionParams) error { return nil }

// setupTestAPI creates a test server with auth middleware on a /test endpoint.
func setupTestAPI(store *fakeStore) *httptest.Server {
	svc := auth.NewService(store)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	api.UseMiddleware(authmw.RequireAuth(api, svc))

	huma.Register(api, huma.Operation{
		OperationID: "test-authed",
		Method:      http.MethodGet,
		Path:        "/test",
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body struct{ UserID string `json:"user_id"` } }, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("no user")
		}
		out := &struct{ Body struct{ UserID string `json:"user_id"` } }{}
		out.Body.UserID = user.ID
		return out, nil
	})

	return httptest.NewServer(r)
}

func TestMiddleware_ValidCookie_SetsUserInContext(t *testing.T) {
	now := time.Now()
	store := &fakeStore{
		sessionRow: db.GetSessionWithUserRow{
			Token: "valid-token", UserID: "user-1", ExpiresAt: now.Add(time.Hour),
			UserID2: "user-1", Email: "test@test.com", DisplayName: "Test",
			Status: "active", CreatedAt: now, UpdatedAt: now,
		},
	}
	ts := setupTestAPI(store)
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/test", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "valid-token"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestMiddleware_MissingCookie_Returns401(t *testing.T) {
	store := &fakeStore{sessionErr: sql.ErrNoRows}
	ts := setupTestAPI(store)
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/test", nil)
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

func TestMiddleware_InvalidSession_Returns401(t *testing.T) {
	store := &fakeStore{sessionErr: sql.ErrNoRows}
	ts := setupTestAPI(store)
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/test", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "expired-token"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}
