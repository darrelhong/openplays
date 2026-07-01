package notifications

import (
	"context"
	"testing"
	"time"

	"openplays/server/internal/db"
)

func TestNotifyDoesNotBlockOnPushDelivery(t *testing.T) {
	store := &blockingWebPushStore{
		listStarted: make(chan struct{}),
		unblockList: make(chan struct{}),
	}
	service := &WebPushService{
		publicKey:  "public-key",
		privateKey: "private-key",
		subscriber: "mailto:dev@openplays.app",
		store:      store,
	}

	err := service.Notify(context.Background(), "user-1", Payload{Title: "Friday Friendly"})
	if err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if !store.created {
		t.Fatal("notification was not stored")
	}

	select {
	case <-store.listStarted:
	case <-time.After(time.Second):
		t.Fatal("push delivery did not start")
	}
	close(store.unblockList)
}

func TestNotifyCanStoreFeedWithoutPushDelivery(t *testing.T) {
	// No real kind is feed-only today; register a synthetic one so the
	// Push-gating branch in Notify stays covered.
	const feedOnlyKind = "test.feed_only"
	deliveryPoliciesByKind[feedOnlyKind] = DeliveryPolicy{Feed: true, Push: false}
	t.Cleanup(func() { delete(deliveryPoliciesByKind, feedOnlyKind) })

	store := &blockingWebPushStore{
		listStarted: make(chan struct{}),
		unblockList: make(chan struct{}),
	}
	service := &WebPushService{
		publicKey:  "public-key",
		privateKey: "private-key",
		subscriber: "mailto:dev@openplays.app",
		store:      store,
	}

	err := service.Notify(context.Background(), "user-1", Payload{
		Title: "Friday Friendly",
		Kind:  feedOnlyKind,
	})
	if err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if !store.created {
		t.Fatal("notification was not stored")
	}

	select {
	case <-store.listStarted:
		close(store.unblockList)
		t.Fatal("push delivery started")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestNotifyDebouncesChatFeedByTag(t *testing.T) {
	store := &blockingWebPushStore{
		listStarted: make(chan struct{}),
		unblockList: make(chan struct{}),
	}
	service := &WebPushService{
		publicKey:  "public-key",
		privateKey: "private-key",
		subscriber: "mailto:dev@openplays.app",
		store:      store,
	}

	err := service.Notify(context.Background(), "user-1", Payload{
		Title: "Alice Tan",
		Body:  "hello",
		Kind:  "chat.message",
		Tag:   "chat:conversation-1",
	})
	if err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if store.created {
		t.Fatal("chat notification used non-debounced create")
	}
	if !store.upserted {
		t.Fatal("chat notification did not use debounced upsert")
	}

	select {
	case <-store.listStarted:
	case <-time.After(time.Second):
		t.Fatal("push delivery did not start")
	}
	close(store.unblockList)
}

type blockingWebPushStore struct {
	created     bool
	upserted    bool
	listStarted chan struct{}
	unblockList chan struct{}
}

func (s *blockingWebPushStore) GetWebPushVAPIDKeys(context.Context) (db.WebPushVapidKey, error) {
	panic("unexpected GetWebPushVAPIDKeys call")
}

func (s *blockingWebPushStore) CreateWebPushVAPIDKeys(context.Context, db.CreateWebPushVAPIDKeysParams) (db.WebPushVapidKey, error) {
	panic("unexpected CreateWebPushVAPIDKeys call")
}

func (s *blockingWebPushStore) UpsertWebPushSubscription(context.Context, db.UpsertWebPushSubscriptionParams) error {
	panic("unexpected UpsertWebPushSubscription call")
}

func (s *blockingWebPushStore) ListWebPushSubscriptionsByUser(context.Context, string) ([]db.WebPushSubscription, error) {
	close(s.listStarted)
	<-s.unblockList
	return nil, nil
}

func (s *blockingWebPushStore) DeleteWebPushSubscription(context.Context, db.DeleteWebPushSubscriptionParams) error {
	panic("unexpected DeleteWebPushSubscription call")
}

func (s *blockingWebPushStore) CreateUserNotification(_ context.Context, arg db.CreateUserNotificationParams) (db.UserNotification, error) {
	s.created = true
	return db.UserNotification{
		ID:        arg.ID,
		UserID:    arg.UserID,
		Title:     arg.Title,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (s *blockingWebPushStore) UpsertChatUserNotificationByTag(_ context.Context, arg db.UpsertChatUserNotificationByTagParams) (db.UserNotification, error) {
	s.upserted = true
	return db.UserNotification{
		ID:        arg.ID,
		UserID:    arg.UserID,
		Title:     arg.Title,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (s *blockingWebPushStore) ListUserNotifications(context.Context, db.ListUserNotificationsParams) ([]db.UserNotification, error) {
	panic("unexpected ListUserNotifications call")
}

func (s *blockingWebPushStore) MarkAllUserNotificationsRead(context.Context, string) error {
	panic("unexpected MarkAllUserNotificationsRead call")
}

func (s *blockingWebPushStore) MarkUserNotificationsRead(context.Context, db.MarkUserNotificationsReadParams) error {
	panic("unexpected MarkUserNotificationsRead call")
}
