package chat_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/routes/api/chat"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/testdb"
)

func setupChatTest(t *testing.T) (*httptest.Server, *db.Queries) {
	t.Helper()
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	svc := auth.NewService(queries)

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api")
	chat.Register(grp, queries, svc)

	return httptest.NewServer(r), queries
}

func TestDMConversationFlow(t *testing.T) {
	ts, queries := setupChatTest(t)
	defer ts.Close()
	ctx := context.Background()

	alice := createChatTestUser(t, ctx, queries, "chat-alice", "Alice Tan", "alice_tan", "active")
	bob := createChatTestUser(t, ctx, queries, "chat-bob", "Bob Lee", "bob_lee", "active")
	createChatSession(t, ctx, queries, alice.ID, "alice-token")
	createChatSession(t, ctx, queries, bob.ID, "bob-token")

	createResp := doJSON[conversationResponse](t, http.MethodPost, ts.URL+"/api/chat/dms", "alice-token", map[string]string{
		"recipient_user_id": bob.ID,
	}, http.StatusOK)
	if createResp.ID == "" {
		t.Fatal("conversation id is empty")
	}
	if createResp.OtherUser == nil || createResp.OtherUser.ID != bob.ID {
		t.Fatalf("other_user = %#v, want Bob", createResp.OtherUser)
	}

	createAgain := doJSON[conversationResponse](t, http.MethodPost, ts.URL+"/api/chat/dms", "alice-token", map[string]string{
		"recipient_user_id": bob.ID,
	}, http.StatusOK)
	if createAgain.ID != createResp.ID {
		t.Fatalf("second conversation id = %q, want %q", createAgain.ID, createResp.ID)
	}

	message := doJSON[messageResponse](t, http.MethodPost, ts.URL+"/api/chat/conversations/"+createResp.ID+"/messages", "alice-token", map[string]string{
		"body": "  hello bob  ",
	}, http.StatusOK)
	if message.Body == nil || *message.Body != "hello bob" {
		t.Fatalf("message body = %v, want trimmed body", message.Body)
	}
	if !message.CanDelete {
		t.Fatal("sender should be able to delete their message")
	}

	bobMessages := doJSON[listMessagesResponse](t, http.MethodGet, ts.URL+"/api/chat/conversations/"+createResp.ID+"/messages", "bob-token", nil, http.StatusOK)
	if len(bobMessages.Items) != 1 {
		t.Fatalf("messages len = %d, want 1", len(bobMessages.Items))
	}
	if bobMessages.Items[0].CanDelete {
		t.Fatal("recipient should not be able to delete sender's message")
	}

	bobConversations := doJSON[listConversationsResponse](t, http.MethodGet, ts.URL+"/api/chat/conversations", "bob-token", nil, http.StatusOK)
	if len(bobConversations.Items) != 1 {
		t.Fatalf("conversations len = %d, want 1", len(bobConversations.Items))
	}
	if bobConversations.Items[0].UnreadCount != 1 {
		t.Fatalf("unread_count = %d, want 1", bobConversations.Items[0].UnreadCount)
	}
	if bobConversations.Items[0].LastMessage == nil || bobConversations.Items[0].LastMessage.Body == nil {
		t.Fatalf("last_message missing body: %#v", bobConversations.Items[0].LastMessage)
	}

	doJSON[map[string]any](t, http.MethodPost, ts.URL+"/api/chat/conversations/"+createResp.ID+"/read", "bob-token", map[string]int64{
		"last_read_message_id": message.ID,
	}, http.StatusNoContent)
	bobConversations = doJSON[listConversationsResponse](t, http.MethodGet, ts.URL+"/api/chat/conversations", "bob-token", nil, http.StatusOK)
	if bobConversations.Items[0].UnreadCount != 0 {
		t.Fatalf("unread_count after read = %d, want 0", bobConversations.Items[0].UnreadCount)
	}

	messageID := strconv.FormatInt(message.ID, 10)
	doJSON[map[string]any](t, http.MethodDelete, ts.URL+"/api/chat/conversations/"+createResp.ID+"/messages/"+messageID, "bob-token", nil, http.StatusForbidden)
	doJSON[map[string]any](t, http.MethodDelete, ts.URL+"/api/chat/conversations/"+createResp.ID+"/messages/"+messageID, "alice-token", nil, http.StatusNoContent)

	bobMessages = doJSON[listMessagesResponse](t, http.MethodGet, ts.URL+"/api/chat/conversations/"+createResp.ID+"/messages", "bob-token", nil, http.StatusOK)
	if bobMessages.Items[0].Body != nil {
		t.Fatalf("deleted message body = %q, want nil", *bobMessages.Items[0].Body)
	}
	if bobMessages.Items[0].DeletedAt == nil {
		t.Fatal("deleted message missing deleted_at")
	}
}

func TestDMAccessGuards(t *testing.T) {
	ts, queries := setupChatTest(t)
	defer ts.Close()
	ctx := context.Background()

	alice := createChatTestUser(t, ctx, queries, "guard-alice", "Alice Tan", "guard_alice", "active")
	suspended := createChatTestUser(t, ctx, queries, "guard-suspended", "Suspended", "guard_suspended", "suspended")
	createChatSession(t, ctx, queries, alice.ID, "alice-token")

	doJSON[map[string]any](t, http.MethodPost, ts.URL+"/api/chat/dms", "alice-token", map[string]string{
		"recipient_user_id": alice.ID,
	}, http.StatusUnprocessableEntity)
	doJSON[map[string]any](t, http.MethodPost, ts.URL+"/api/chat/dms", "alice-token", map[string]string{
		"recipient_user_id": suspended.ID,
	}, http.StatusUnprocessableEntity)
}

func TestBlockedDMIsHiddenAndForbiddenUntilUnblocked(t *testing.T) {
	ts, queries := setupChatTest(t)
	defer ts.Close()
	ctx := context.Background()

	alice := createChatTestUser(t, ctx, queries, "block-alice", "Alice Tan", "block_alice", "active")
	bob := createChatTestUser(t, ctx, queries, "block-bob", "Bob Lee", "block_bob", "active")
	createChatSession(t, ctx, queries, alice.ID, "alice-token")

	conversation := doJSON[conversationResponse](t, http.MethodPost, ts.URL+"/api/chat/dms", "alice-token", map[string]string{
		"recipient_user_id": bob.ID,
	}, http.StatusOK)
	if err := queries.CreateBlock(ctx, db.CreateBlockParams{BlockerID: alice.ID, BlockedID: bob.ID}); err != nil {
		t.Fatalf("CreateBlock: %v", err)
	}

	conversations := doJSON[listConversationsResponse](t, http.MethodGet, ts.URL+"/api/chat/conversations", "alice-token", nil, http.StatusOK)
	if len(conversations.Items) != 0 {
		t.Fatalf("blocked conversations len = %d, want 0", len(conversations.Items))
	}
	doJSON[map[string]any](t, http.MethodGet, ts.URL+"/api/chat/conversations/"+conversation.ID+"/messages", "alice-token", nil, http.StatusForbidden)
	doJSON[map[string]any](t, http.MethodPost, ts.URL+"/api/chat/conversations/"+conversation.ID+"/messages", "alice-token", map[string]string{
		"body": "still there?",
	}, http.StatusForbidden)

	if err := queries.DeleteBlock(ctx, db.DeleteBlockParams{BlockerID: alice.ID, BlockedID: bob.ID}); err != nil {
		t.Fatalf("DeleteBlock: %v", err)
	}
	createAgain := doJSON[conversationResponse](t, http.MethodPost, ts.URL+"/api/chat/dms", "alice-token", map[string]string{
		"recipient_user_id": bob.ID,
	}, http.StatusOK)
	if createAgain.ID != conversation.ID {
		t.Fatalf("conversation after unblock = %q, want existing %q", createAgain.ID, conversation.ID)
	}
}

type conversationResponse struct {
	ID          string           `json:"id"`
	OtherUser   *userSummary     `json:"other_user"`
	LastMessage *messageResponse `json:"last_message"`
	UnreadCount int64            `json:"unread_count"`
}

type listConversationsResponse struct {
	Items []conversationResponse `json:"items"`
}

type messageResponse struct {
	ID        int64       `json:"id"`
	Sender    userSummary `json:"sender"`
	Body      *string     `json:"body"`
	DeletedAt *string     `json:"deleted_at"`
	CanDelete bool        `json:"can_delete"`
}

type listMessagesResponse struct {
	Items []messageResponse `json:"items"`
}

type userSummary struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Username    *string `json:"username"`
}

func createChatTestUser(t *testing.T, ctx context.Context, queries *db.Queries, id, displayName, username, status string) db.User {
	t.Helper()
	googleID := "google-" + id
	user, err := queries.UpsertUserByGoogleID(ctx, db.UpsertUserByGoogleIDParams{
		ID:          id,
		Email:       id + "@example.test",
		Username:    &username,
		DisplayName: displayName,
		GoogleID:    &googleID,
	})
	if err != nil {
		t.Fatalf("UpsertUserByGoogleID: %v", err)
	}
	if status != "active" {
		if err := queries.UpdateUserStatus(ctx, db.UpdateUserStatusParams{ID: id, Status: status}); err != nil {
			t.Fatalf("UpdateUserStatus: %v", err)
		}
		user.Status = status
	}
	return user
}

func createChatSession(t *testing.T, ctx context.Context, queries *db.Queries, userID, token string) {
	t.Helper()
	if err := queries.CreateSession(ctx, db.CreateSessionParams{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
}

func doJSON[T any](t *testing.T, method, url, token string, body any, wantStatus int) T {
	t.Helper()
	var reqBody *bytes.Reader
	if body == nil {
		reqBody = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reqBody = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "session", Value: token})
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s status = %d, want %d", method, url, resp.StatusCode, wantStatus)
	}
	var out T
	if wantStatus == http.StatusNoContent {
		return out
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}
