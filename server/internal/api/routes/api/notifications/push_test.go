package notifications_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/authmw"
	notificationsRouter "openplays/server/internal/api/routes/api/notifications"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/notifications"
	"openplays/server/internal/testdb"
)

type fakeAuthStore struct {
	sessionRow db.GetSessionWithUserRow
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
	return f.sessionRow, nil
}
func (f *fakeAuthStore) CreateSession(_ context.Context, _ db.CreateSessionParams) error {
	return nil
}
func (f *fakeAuthStore) DeleteSession(_ context.Context, _ string) error {
	return nil
}
func (f *fakeAuthStore) RefreshSession(_ context.Context, _ db.RefreshSessionParams) error {
	return nil
}

func TestSubscribeWebPushAcceptsBrowserSubscriptionShape(t *testing.T) {
	service, ts := setupNotificationTest(t)
	defer ts.Close()
	_ = service

	body := `{
		"endpoint":"https://updates.push.services.mozilla.com/wpush/v2/example",
		"expirationTime":null,
		"keys":{"auth":"auth-token","p256dh":"public-key"}
	}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/notifications/push/subscriptions", strings.NewReader(body))
	req.Header.Set("content-type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
}

func TestSubscribeWebPushRejectsArbitraryEndpoint(t *testing.T) {
	service, ts := setupNotificationTest(t)
	defer ts.Close()
	_ = service

	body := `{
		"endpoint":"https://169.254.169.254/latest/meta-data",
		"keys":{"auth":"auth-token","p256dh":"public-key"}
	}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/notifications/push/subscriptions", strings.NewReader(body))
	req.Header.Set("content-type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSubscribeWebPushRejectsGoogleapisNonPushEndpoint(t *testing.T) {
	service, ts := setupNotificationTest(t)
	defer ts.Close()
	_ = service

	body := `{
		"endpoint":"https://storage.googleapis.com/private-bucket/object",
		"keys":{"auth":"auth-token","p256dh":"public-key"}
	}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/notifications/push/subscriptions", strings.NewReader(body))
	req.Header.Set("content-type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestListNotificationsAndMarkRead(t *testing.T) {
	service, ts := setupNotificationTest(t)
	defer ts.Close()

	if err := service.Notify(context.Background(), "user-1", notifications.Payload{
		Title: "Friday Friendly",
		Body:  "You were added to the game",
		URL:   "/play/play-1",
		Kind:  "play.player_added",
	}); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/notifications/?limit=50", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Notifications []notifications.UserNotification `json:"notifications"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(out.Notifications) != 1 {
		t.Fatalf("notifications len = %d, want 1", len(out.Notifications))
	}
	if out.Notifications[0].ReadAt != nil {
		t.Fatalf("read_at = %v, want unread", *out.Notifications[0].ReadAt)
	}

	readReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/notifications/read", strings.NewReader(`{"ids":[]}`))
	readReq.Header.Set("content-type", "application/json")
	readReq.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	readResp, err := http.DefaultClient.Do(readReq)
	if err != nil {
		t.Fatal(err)
	}
	defer readResp.Body.Close()
	if readResp.StatusCode != http.StatusNoContent {
		t.Fatalf("mark read status = %d, want 204", readResp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/notifications/?limit=50", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "tok"})
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode reread list: %v", err)
	}
	if out.Notifications[0].ReadAt == nil {
		t.Fatalf("read_at is nil, want marked read")
	}
}

func TestSQLiteWebPushServicePersistsNotificationsAndVAPIDKeys(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	createNotificationTestUser(t, queries)

	service, err := notifications.NewSQLiteWebPushService(context.Background(), queries, "mailto:dev@openplays.app")
	if err != nil {
		t.Fatalf("NewSQLiteWebPushService: %v", err)
	}
	publicKey := service.PublicKey()
	if err := service.Notify(context.Background(), "user-1", notifications.Payload{
		Title: "Friday Friendly",
		Body:  "Test notification",
		Kind:  "test.notification",
	}); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	reloaded, err := notifications.NewSQLiteWebPushService(context.Background(), queries, "mailto:dev@openplays.app")
	if err != nil {
		t.Fatalf("reload NewSQLiteWebPushService: %v", err)
	}
	if reloaded.PublicKey() != publicKey {
		t.Fatalf("public key changed after reload")
	}
	items, err := reloaded.ListNotifications(context.Background(), "user-1", 50)
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("notifications len = %d, want 1", len(items))
	}
	if items[0].Title != "Friday Friendly" {
		t.Fatalf("notification title = %q, want Friday Friendly", items[0].Title)
	}
}

func TestSQLiteWebPushServiceDebouncesChatNotificationsByTag(t *testing.T) {
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	createNotificationTestUser(t, queries)

	service, err := notifications.NewSQLiteWebPushService(context.Background(), queries, "mailto:dev@openplays.app")
	if err != nil {
		t.Fatalf("NewSQLiteWebPushService: %v", err)
	}
	payload := notifications.Payload{
		Title: "Alice Tan",
		Body:  "first message",
		Kind:  notifications.ChatMessageKind,
		Tag:   "chat:conversation-1",
	}
	if err := service.Notify(context.Background(), "user-1", payload); err != nil {
		t.Fatalf("Notify first: %v", err)
	}
	items, err := service.ListNotifications(context.Background(), "user-1", 50)
	if err != nil {
		t.Fatalf("ListNotifications first: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("notifications len after first = %d, want 1", len(items))
	}
	if err := service.MarkNotificationsRead(context.Background(), "user-1", nil); err != nil {
		t.Fatalf("MarkNotificationsRead: %v", err)
	}

	payload.Body = "second message"
	if err := service.Notify(context.Background(), "user-1", payload); err != nil {
		t.Fatalf("Notify second: %v", err)
	}
	items, err = service.ListNotifications(context.Background(), "user-1", 50)
	if err != nil {
		t.Fatalf("ListNotifications second: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("notifications len after second = %d, want 1", len(items))
	}
	if items[0].Body != "second message" {
		t.Fatalf("notification body = %q, want latest message", items[0].Body)
	}
	if items[0].ReadAt != nil {
		t.Fatalf("read_at = %v, want unread after new message", *items[0].ReadAt)
	}
}

func setupNotificationTest(t *testing.T) (*notifications.WebPushService, *httptest.Server) {
	t.Helper()

	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	createNotificationTestUser(t, queries)

	now := time.Now()
	authStore := &fakeAuthStore{
		sessionRow: db.GetSessionWithUserRow{
			Token:       "tok",
			UserID:      "user-1",
			ExpiresAt:   now.Add(time.Hour),
			UserID2:     "user-1",
			Email:       "user-1@example.com",
			DisplayName: "Test User",
			Status:      "active",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
	service, err := notifications.NewSQLiteWebPushService(context.Background(), queries, "mailto:dev@openplays.app")
	if err != nil {
		t.Fatalf("NewSQLiteWebPushService: %v", err)
	}

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api")
	notificationsRouter.Register(grp, service, authmw.RequireAuth(api, auth.NewService(authStore)))
	return service, httptest.NewServer(r)
}

func createNotificationTestUser(t *testing.T, queries *db.Queries) {
	t.Helper()

	googleID := "google-user-1"
	if _, err := queries.UpsertUserByGoogleID(context.Background(), db.UpsertUserByGoogleIDParams{
		ID:          "user-1",
		Email:       "user-1@example.com",
		DisplayName: "Test User",
		GoogleID:    &googleID,
	}); err != nil {
		t.Fatalf("UpsertUserByGoogleID: %v", err)
	}
}
