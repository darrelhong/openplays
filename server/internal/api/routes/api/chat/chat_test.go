package chat_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"openplays/server/internal/api/routes/api/chat"
	"openplays/server/internal/auth"
	"openplays/server/internal/db"
	"openplays/server/internal/model"
	"openplays/server/internal/notifications"
	"openplays/server/internal/testdb"
)

func setupChatTest(t *testing.T, notifiers ...notifications.Sender) (*httptest.Server, *db.Queries) {
	t.Helper()
	sqlDB := testdb.New(t)
	queries := db.New(sqlDB)
	svc := auth.NewService(queries)
	var notifier notifications.Sender
	if len(notifiers) > 0 {
		notifier = notifiers[0]
	}

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "0.0.1"))
	grp := huma.NewGroup(api, "/api")
	chat.Register(grp, queries, svc, notifier)

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

func TestListConversationsPagination(t *testing.T) {
	ts, queries := setupChatTest(t)
	defer ts.Close()
	ctx := context.Background()

	alice := createChatTestUser(t, ctx, queries, "page-alice", "Alice Tan", "alice_page", "active")
	createChatSession(t, ctx, queries, alice.ID, "alice-token")

	conversationIDs := make([]string, 0, 3)
	for i, peer := range []struct{ id, name, username string }{
		{"page-bob", "Bob Lee", "bob_page"},
		{"page-carol", "Carol Ng", "carol_page"},
		{"page-dave", "Dave Koh", "dave_page"},
	} {
		user := createChatTestUser(t, ctx, queries, peer.id, peer.name, peer.username, "active")
		conversation := doJSON[conversationResponse](t, http.MethodPost, ts.URL+"/api/chat/dms", "alice-token", map[string]string{
			"recipient_user_id": user.ID,
		}, http.StatusOK)
		doJSON[messageResponse](t, http.MethodPost, ts.URL+"/api/chat/conversations/"+conversation.ID+"/messages", "alice-token", map[string]string{
			"body": "hello " + strconv.Itoa(i),
		}, http.StatusOK)
		conversationIDs = append(conversationIDs, conversation.ID)
	}

	firstPage := doJSON[listConversationsResponse](t, http.MethodGet, ts.URL+"/api/chat/conversations?limit=2", "alice-token", nil, http.StatusOK)
	if len(firstPage.Items) != 2 {
		t.Fatalf("first page len = %d, want 2", len(firstPage.Items))
	}
	if firstPage.NextCursor == nil {
		t.Fatal("first page next_cursor is nil, want cursor")
	}

	secondPage := doJSON[listConversationsResponse](t, http.MethodGet, ts.URL+"/api/chat/conversations?limit=2&cursor="+url.QueryEscape(*firstPage.NextCursor), "alice-token", nil, http.StatusOK)
	if len(secondPage.Items) != 1 {
		t.Fatalf("second page len = %d, want 1", len(secondPage.Items))
	}
	if secondPage.NextCursor != nil {
		t.Fatalf("second page next_cursor = %q, want none", *secondPage.NextCursor)
	}

	seen := map[string]bool{}
	for _, item := range append(firstPage.Items, secondPage.Items...) {
		if seen[item.ID] {
			t.Fatalf("conversation %s returned twice", item.ID)
		}
		seen[item.ID] = true
	}
	for _, id := range conversationIDs {
		if !seen[id] {
			t.Fatalf("conversation %s missing from paginated results", id)
		}
	}

	doJSON[map[string]any](t, http.MethodGet, ts.URL+"/api/chat/conversations?cursor=not-a-cursor", "alice-token", nil, http.StatusUnprocessableEntity)
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

func TestDMMessageNotifiesPeer(t *testing.T) {
	notifier := &fakeChatNotificationSender{}
	ts, queries := setupChatTest(t, notifier)
	defer ts.Close()
	ctx := context.Background()

	alice := createChatTestUser(t, ctx, queries, "notify-dm-alice", "Alice Tan", "notify_dm_alice", "active")
	bob := createChatTestUser(t, ctx, queries, "notify-dm-bob", "Bob Lee", "notify_dm_bob", "active")
	createChatSession(t, ctx, queries, alice.ID, "alice-token")
	createChatSession(t, ctx, queries, bob.ID, "bob-token")

	conversation := doJSON[conversationResponse](t, http.MethodPost, ts.URL+"/api/chat/dms", "alice-token", map[string]string{
		"recipient_user_id": bob.ID,
	}, http.StatusOK)
	doJSON[messageResponse](t, http.MethodPost, ts.URL+"/api/chat/conversations/"+conversation.ID+"/messages", "alice-token", map[string]string{
		"body": "hello bob",
	}, http.StatusOK)

	if len(notifier.calls) != 1 {
		t.Fatalf("notification calls = %d, want 1", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.userID != bob.ID {
		t.Fatalf("notification user = %q, want Bob", call.userID)
	}
	if call.payload.Kind != notifications.ChatMessageKind {
		t.Fatalf("notification kind = %q, want chat.message", call.payload.Kind)
	}
	if call.payload.Tag != "chat:"+conversation.ID {
		t.Fatalf("notification tag = %q, want chat:%s", call.payload.Tag, conversation.ID)
	}
	if call.payload.Title != "Alice Tan" {
		t.Fatalf("notification title = %q, want sender name", call.payload.Title)
	}
	if call.payload.Body != "hello bob" {
		t.Fatalf("notification body = %q, want message preview", call.payload.Body)
	}
	if call.payload.Data["conversation_id"] != conversation.ID || call.payload.Data["conversation_kind"] != "dm" {
		t.Fatalf("notification data = %#v, want conversation metadata", call.payload.Data)
	}
}

func TestPlayConversationFlow(t *testing.T) {
	ts, queries := setupChatTest(t)
	defer ts.Close()
	ctx := context.Background()

	host := createChatTestUser(t, ctx, queries, "play-chat-host", "Host Tan", "play_chat_host", "active")
	confirmed := createChatTestUser(t, ctx, queries, "play-chat-confirmed", "Confirmed Lee", "play_chat_confirmed", "active")
	added := createChatTestUser(t, ctx, queries, "play-chat-added", "Added Goh", "play_chat_added", "active")
	waitlisted := createChatTestUser(t, ctx, queries, "play-chat-waitlisted", "Waitlisted Ng", "play_chat_waitlisted", "active")
	outsider := createChatTestUser(t, ctx, queries, "play-chat-outsider", "Outsider Lim", "play_chat_outsider", "active")
	createChatSession(t, ctx, queries, host.ID, "host-token")
	createChatSession(t, ctx, queries, confirmed.ID, "confirmed-token")
	createChatSession(t, ctx, queries, added.ID, "added-token")
	createChatSession(t, ctx, queries, waitlisted.ID, "waitlisted-token")
	createChatSession(t, ctx, queries, outsider.ID, "outsider-token")

	play := createChatTestPlay(t, ctx, queries, "play-chat-game", host.ID, "Wednesday Doubles")
	createChatPlayParticipant(t, ctx, queries, play.ID, confirmed.ID, model.ParticipantConfirmed)
	createChatPlayParticipant(t, ctx, queries, play.ID, added.ID, model.ParticipantAdded)
	createChatPlayParticipant(t, ctx, queries, play.ID, waitlisted.ID, model.ParticipantWaitlisted)

	waitlistedCreate := map[string]string{"play_id": play.ID}
	doJSON[map[string]any](t, http.MethodPost, ts.URL+"/api/chat/play-conversations", "waitlisted-token", waitlistedCreate, http.StatusForbidden)
	doJSON[map[string]any](t, http.MethodPost, ts.URL+"/api/chat/play-conversations", "outsider-token", waitlistedCreate, http.StatusForbidden)

	conversation := doJSON[conversationResponse](t, http.MethodPost, ts.URL+"/api/chat/play-conversations", "host-token", map[string]string{
		"play_id": play.ID,
	}, http.StatusOK)
	if conversation.ID == "" {
		t.Fatal("conversation id is empty")
	}
	if conversation.Kind != "play" {
		t.Fatalf("kind = %q, want play", conversation.Kind)
	}
	if conversation.PlayID == nil || *conversation.PlayID != play.ID {
		t.Fatalf("play_id = %v, want %s", conversation.PlayID, play.ID)
	}
	if conversation.Title != "Wednesday Doubles" {
		t.Fatalf("title = %q, want custom play name", conversation.Title)
	}

	again := doJSON[conversationResponse](t, http.MethodPost, ts.URL+"/api/chat/play-conversations", "confirmed-token", map[string]string{
		"play_id": play.ID,
	}, http.StatusOK)
	if again.ID != conversation.ID {
		t.Fatalf("second conversation id = %q, want %q", again.ID, conversation.ID)
	}

	message := doJSON[messageResponse](t, http.MethodPost, ts.URL+"/api/chat/conversations/"+conversation.ID+"/messages", "confirmed-token", map[string]string{
		"body": "see everyone there",
	}, http.StatusOK)
	if message.Body == nil || *message.Body != "see everyone there" {
		t.Fatalf("message body = %v, want play chat body", message.Body)
	}

	hostMessages := doJSON[listMessagesResponse](t, http.MethodGet, ts.URL+"/api/chat/conversations/"+conversation.ID+"/messages", "host-token", nil, http.StatusOK)
	if len(hostMessages.Items) != 1 {
		t.Fatalf("host messages len = %d, want 1", len(hostMessages.Items))
	}
	addedMessages := doJSON[listMessagesResponse](t, http.MethodGet, ts.URL+"/api/chat/conversations/"+conversation.ID+"/messages", "added-token", nil, http.StatusOK)
	if len(addedMessages.Items) != 1 {
		t.Fatalf("added messages len = %d, want 1", len(addedMessages.Items))
	}
	doJSON[map[string]any](t, http.MethodGet, ts.URL+"/api/chat/conversations/"+conversation.ID+"/messages", "waitlisted-token", nil, http.StatusForbidden)
	doJSON[map[string]any](t, http.MethodPost, ts.URL+"/api/chat/conversations/"+conversation.ID+"/messages", "outsider-token", map[string]string{
		"body": "can I join?",
	}, http.StatusForbidden)

	hostConversations := doJSON[listConversationsResponse](t, http.MethodGet, ts.URL+"/api/chat/conversations", "host-token", nil, http.StatusOK)
	if len(hostConversations.Items) != 1 {
		t.Fatalf("host conversations len = %d, want 1", len(hostConversations.Items))
	}
	if hostConversations.Items[0].ID != conversation.ID {
		t.Fatalf("listed conversation = %q, want %q", hostConversations.Items[0].ID, conversation.ID)
	}
	if hostConversations.Items[0].UnreadCount != 1 {
		t.Fatalf("host unread_count = %d, want 1", hostConversations.Items[0].UnreadCount)
	}
}

func TestPlayMessageNotifiesCurrentRosterExceptSender(t *testing.T) {
	notifier := &fakeChatNotificationSender{}
	ts, queries := setupChatTest(t, notifier)
	defer ts.Close()
	ctx := context.Background()

	host := createChatTestUser(t, ctx, queries, "notify-play-host", "Host Tan", "notify_play_host", "active")
	confirmed := createChatTestUser(t, ctx, queries, "notify-play-confirmed", "Confirmed Lee", "notify_play_confirmed", "active")
	added := createChatTestUser(t, ctx, queries, "notify-play-added", "Added Goh", "notify_play_added", "active")
	waitlisted := createChatTestUser(t, ctx, queries, "notify-play-waitlisted", "Waitlisted Ng", "notify_play_waitlisted", "active")
	createChatSession(t, ctx, queries, host.ID, "host-token")
	createChatSession(t, ctx, queries, confirmed.ID, "confirmed-token")

	play := createChatTestPlay(t, ctx, queries, "notify-play-game", host.ID, "Saturday Smash")
	createChatPlayParticipant(t, ctx, queries, play.ID, confirmed.ID, model.ParticipantConfirmed)
	createChatPlayParticipant(t, ctx, queries, play.ID, added.ID, model.ParticipantAdded)
	createChatPlayParticipant(t, ctx, queries, play.ID, waitlisted.ID, model.ParticipantWaitlisted)

	conversation := doJSON[conversationResponse](t, http.MethodPost, ts.URL+"/api/chat/play-conversations", "host-token", map[string]string{
		"play_id": play.ID,
	}, http.StatusOK)
	doJSON[messageResponse](t, http.MethodPost, ts.URL+"/api/chat/conversations/"+conversation.ID+"/messages", "confirmed-token", map[string]string{
		"body": "bring shuttles",
	}, http.StatusOK)

	if len(notifier.calls) != 2 {
		t.Fatalf("notification calls = %d, want host and added player", len(notifier.calls))
	}
	got := map[string]notifications.Payload{}
	for _, call := range notifier.calls {
		got[call.userID] = call.payload
	}
	for _, userID := range []string{host.ID, added.ID} {
		payload, ok := got[userID]
		if !ok {
			t.Fatalf("missing notification for %s", userID)
		}
		if payload.Kind != notifications.ChatMessageKind {
			t.Fatalf("notification kind for %s = %q, want chat.message", userID, payload.Kind)
		}
		if payload.PlayID != play.ID || payload.URL != "/chat/"+conversation.ID {
			t.Fatalf("notification play context for %s = play_id %q url %q", userID, payload.PlayID, payload.URL)
		}
		if payload.Tag != "chat:"+conversation.ID {
			t.Fatalf("notification tag for %s = %q, want chat:%s", userID, payload.Tag, conversation.ID)
		}
		if payload.Body != "Confirmed Lee: bring shuttles" {
			t.Fatalf("notification body for %s = %q, want sender preview", userID, payload.Body)
		}
	}
	if _, ok := got[confirmed.ID]; ok {
		t.Fatal("sender should not receive a play chat notification")
	}
	if _, ok := got[waitlisted.ID]; ok {
		t.Fatal("waitlisted user should not receive a play chat notification")
	}
}

func TestPlayConversationAccessFollowsCurrentRoster(t *testing.T) {
	ts, queries := setupChatTest(t)
	defer ts.Close()
	ctx := context.Background()

	host := createChatTestUser(t, ctx, queries, "play-chat-current-host", "Host Tan", "play_chat_current_host", "active")
	player := createChatTestUser(t, ctx, queries, "play-chat-current-player", "Player Lee", "play_chat_current_player", "active")
	createChatSession(t, ctx, queries, host.ID, "host-token")
	createChatSession(t, ctx, queries, player.ID, "player-token")

	play := createChatTestPlay(t, ctx, queries, "play-chat-current-game", host.ID, "Current Roster Game")
	participant := createChatPlayParticipant(t, ctx, queries, play.ID, player.ID, model.ParticipantConfirmed)
	conversation := doJSON[conversationResponse](t, http.MethodPost, ts.URL+"/api/chat/play-conversations", "player-token", map[string]string{
		"play_id": play.ID,
	}, http.StatusOK)
	doJSON[messageResponse](t, http.MethodPost, ts.URL+"/api/chat/conversations/"+conversation.ID+"/messages", "player-token", map[string]string{
		"body": "I am in",
	}, http.StatusOK)

	if err := queries.DeletePlayParticipant(ctx, participant.ID); err != nil {
		t.Fatalf("DeletePlayParticipant: %v", err)
	}

	doJSON[map[string]any](t, http.MethodGet, ts.URL+"/api/chat/conversations/"+conversation.ID+"/messages", "player-token", nil, http.StatusForbidden)
	playerConversations := doJSON[listConversationsResponse](t, http.MethodGet, ts.URL+"/api/chat/conversations", "player-token", nil, http.StatusOK)
	if len(playerConversations.Items) != 0 {
		t.Fatalf("player conversations len after leaving roster = %d, want 0", len(playerConversations.Items))
	}
}

type chatNotificationCall struct {
	userID  string
	payload notifications.Payload
}

type fakeChatNotificationSender struct {
	calls []chatNotificationCall
}

func (f *fakeChatNotificationSender) Notify(_ context.Context, userID string, payload notifications.Payload) error {
	f.calls = append(f.calls, chatNotificationCall{
		userID:  userID,
		payload: payload,
	})
	return nil
}

type conversationResponse struct {
	ID          string           `json:"id"`
	Kind        string           `json:"kind"`
	Title       string           `json:"title"`
	PlayID      *string          `json:"play_id"`
	OtherUser   *userSummary     `json:"other_user"`
	LastMessage *messageResponse `json:"last_message"`
	UnreadCount int64            `json:"unread_count"`
}

type listConversationsResponse struct {
	Items      []conversationResponse `json:"items"`
	NextCursor *string                `json:"next_cursor"`
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

func createChatTestPlay(t *testing.T, ctx context.Context, queries *db.Queries, id, creatorID, name string) db.Play {
	t.Helper()
	gameType := model.GameDoubles
	maxPlayers := int64(4)
	slotsLeft := int64(3)
	playName := name
	play, err := queries.CreatePlay(ctx, db.CreatePlayParams{
		ID:          id,
		ListingType: model.ListingPlay,
		Sport:       model.SportBadminton,
		GameType:    &gameType,
		HostName:    "Host",
		Name:        &playName,
		StartsAt:    time.Now().Add(time.Hour),
		EndsAt:      time.Now().Add(2 * time.Hour),
		Timezone:    "Asia/Singapore",
		Venue:       "Test Sports Hall",
		Currency:    "SGD",
		MaxPlayers:  &maxPlayers,
		SlotsLeft:   &slotsLeft,
		Contacts:    model.Contacts{},
		Meta:        model.Meta{},
		CreatedBy:   &creatorID,
		Visibility:  "public",
	})
	if err != nil {
		t.Fatalf("CreatePlay: %v", err)
	}
	if _, err := queries.CreatePlayHost(ctx, db.CreatePlayHostParams{PlayID: play.ID, UserID: creatorID}); err != nil {
		t.Fatalf("CreatePlayHost: %v", err)
	}
	return play
}

func createChatPlayParticipant(t *testing.T, ctx context.Context, queries *db.Queries, playID, userID string, status model.PlayParticipantStatus) db.PlayParticipant {
	t.Helper()
	participant, err := queries.CreatePlayParticipant(ctx, db.CreatePlayParticipantParams{
		PlayID: playID,
		UserID: &userID,
		Status: status,
	})
	if err != nil {
		t.Fatalf("CreatePlayParticipant %s: %v", status, err)
	}
	return participant
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
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s %s status = %d, want %d: %s", method, url, resp.StatusCode, wantStatus, string(raw))
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
