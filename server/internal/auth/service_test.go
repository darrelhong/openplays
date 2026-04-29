package auth_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"openplays/server/internal/auth"
	"openplays/server/internal/db"
)

// --- Fake store at DB boundary ---

type fakeStore struct {
	upsertUser db.User
	upsertErr  error

	createSessionCalled bool
	createSessionParams db.CreateSessionParams
	createSessionErr    error

	sessionRow db.GetSessionWithUserRow
	sessionErr error

	deleteSessionCalled bool
	deleteSessionToken  string
	deleteSessionErr    error

	refreshCalled bool
	refreshErr    error
}

func (f *fakeStore) UpsertUserByGoogleID(_ context.Context, arg db.UpsertUserByGoogleIDParams) (db.User, error) {
	if f.upsertErr != nil {
		return db.User{}, f.upsertErr
	}
	u := f.upsertUser
	if u.ID == "" {
		u.ID = arg.ID // simulate new user getting assigned ID
	}
	u.Email = arg.Email
	u.DisplayName = arg.DisplayName
	u.PhotoUrl = arg.PhotoUrl
	if arg.GoogleID != nil {
		u.GoogleID = arg.GoogleID
	}
	return u, nil
}

func (f *fakeStore) CreateSession(_ context.Context, arg db.CreateSessionParams) error {
	f.createSessionCalled = true
	f.createSessionParams = arg
	return f.createSessionErr
}

func (f *fakeStore) GetSessionWithUser(_ context.Context, _ string) (db.GetSessionWithUserRow, error) {
	return f.sessionRow, f.sessionErr
}

func (f *fakeStore) DeleteSession(_ context.Context, token string) error {
	f.deleteSessionCalled = true
	f.deleteSessionToken = token
	return f.deleteSessionErr
}

func (f *fakeStore) RefreshSession(_ context.Context, _ db.RefreshSessionParams) error {
	f.refreshCalled = true
	return f.refreshErr
}

// --- Helpers ---

func googleIdentity(email, name string) auth.Identity {
	return auth.Identity{
		Provider:    auth.ProviderGoogle,
		ProviderID:  "google-123",
		Email:       email,
		DisplayName: name,
		PhotoURL:    "https://photo.url/pic.jpg",
	}
}

func activeUser() db.User {
	return db.User{Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

// --- Login tests ---

func TestLogin_ValidIdentity_CreatesUserAndSession(t *testing.T) {
	store := &fakeStore{upsertUser: activeUser()}
	svc := auth.NewService(store)

	result, err := svc.Login(context.Background(), googleIdentity("test@gmail.com", "Test User"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.User.Email != "test@gmail.com" {
		t.Errorf("email = %q, want test@gmail.com", result.User.Email)
	}
	if result.User.DisplayName != "Test User" {
		t.Errorf("name = %q, want Test User", result.User.DisplayName)
	}
	if result.SessionToken == "" {
		t.Error("session token empty")
	}
	if !store.createSessionCalled {
		t.Error("CreateSession not called")
	}
	if store.createSessionParams.UserID == "" {
		t.Error("session user_id empty")
	}
}

func TestLogin_BannedUser_Rejected(t *testing.T) {
	store := &fakeStore{
		upsertUser: db.User{Status: "banned", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	svc := auth.NewService(store)

	_, err := svc.Login(context.Background(), googleIdentity("banned@gmail.com", "Banned"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, auth.ErrAccountBanned) {
		t.Errorf("error = %v, want ErrAccountBanned", err)
	}
	if store.createSessionCalled {
		t.Error("session should not be created for banned user")
	}
}

func TestLogin_UpsertFails_ReturnsError(t *testing.T) {
	store := &fakeStore{upsertErr: errors.New("db down")}
	svc := auth.NewService(store)

	_, err := svc.Login(context.Background(), googleIdentity("x@x.com", "X"))
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- GetSession tests ---

func TestGetSession_ValidToken_ReturnsUser(t *testing.T) {
	now := time.Now()
	store := &fakeStore{
		sessionRow: db.GetSessionWithUserRow{
			Token: "abc", UserID: "user-1", ExpiresAt: now.Add(time.Hour),
			UserID2: "user-1", Email: "test@gmail.com", DisplayName: "Test", Status: "active",
			CreatedAt: now, UpdatedAt: now,
		},
	}
	svc := auth.NewService(store)

	user, err := svc.GetSession(context.Background(), "abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "user-1" {
		t.Errorf("id = %q, want user-1", user.ID)
	}
	if !store.refreshCalled {
		t.Error("session should be refreshed (rolling expiry)")
	}
}

func TestGetSession_ExpiredToken_ReturnsError(t *testing.T) {
	store := &fakeStore{sessionErr: sql.ErrNoRows}
	svc := auth.NewService(store)

	_, err := svc.GetSession(context.Background(), "expired-token")
	if !errors.Is(err, auth.ErrNoSession) {
		t.Errorf("error = %v, want ErrNoSession", err)
	}
}

func TestGetSession_EmptyToken_ReturnsError(t *testing.T) {
	svc := auth.NewService(&fakeStore{})

	_, err := svc.GetSession(context.Background(), "")
	if !errors.Is(err, auth.ErrNoSession) {
		t.Errorf("error = %v, want ErrNoSession", err)
	}
}

func TestGetSession_SuspendedUser_Rejected(t *testing.T) {
	now := time.Now()
	store := &fakeStore{
		sessionRow: db.GetSessionWithUserRow{
			Token: "abc", UserID: "u1", ExpiresAt: now.Add(time.Hour),
			UserID2: "u1", Email: "x@x.com", DisplayName: "X", Status: "suspended",
			CreatedAt: now, UpdatedAt: now,
		},
	}
	svc := auth.NewService(store)

	_, err := svc.GetSession(context.Background(), "abc")
	if !errors.Is(err, auth.ErrAccountBanned) {
		t.Errorf("error = %v, want ErrAccountBanned", err)
	}
}

// --- Logout tests ---

func TestLogout_DeletesSession(t *testing.T) {
	store := &fakeStore{}
	svc := auth.NewService(store)

	err := svc.Logout(context.Background(), "my-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.deleteSessionCalled {
		t.Error("DeleteSession not called")
	}
	if store.deleteSessionToken != "my-token" {
		t.Errorf("deleted token = %q, want my-token", store.deleteSessionToken)
	}
}

func TestLogout_EmptyToken_NoOp(t *testing.T) {
	store := &fakeStore{}
	svc := auth.NewService(store)

	err := svc.Logout(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.deleteSessionCalled {
		t.Error("should not call DeleteSession for empty token")
	}
}
