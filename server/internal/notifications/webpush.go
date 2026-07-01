package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"

	"openplays/server/internal/db"
)

const webPushSendTimeout = 10 * time.Second

type Payload struct {
	Title  string            `json:"title"`
	Body   string            `json:"body,omitempty"`
	URL    string            `json:"url,omitempty"`
	Tag    string            `json:"tag,omitempty"`
	Kind   string            `json:"kind,omitempty"`
	PlayID string            `json:"play_id,omitempty"`
	Data   map[string]string `json:"data,omitempty"`
}

type PushSubscription struct {
	Endpoint       string               `json:"endpoint"`
	ExpirationTime *int64               `json:"expirationTime,omitempty"`
	Keys           PushSubscriptionKeys `json:"keys"`
}

type PushSubscriptionKeys struct {
	Auth   string `json:"auth"`
	P256DH string `json:"p256dh"`
}

type Sender interface {
	Notify(ctx context.Context, userID string, payload Payload) error
}

type UserNotification struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Body      string  `json:"body,omitempty"`
	URL       string  `json:"url,omitempty"`
	Kind      string  `json:"kind,omitempty"`
	PlayID    string  `json:"play_id,omitempty"`
	ReadAt    *string `json:"read_at,omitempty"`
	CreatedAt string  `json:"created_at"`
}

type WebPushStore interface {
	GetWebPushVAPIDKeys(ctx context.Context) (db.WebPushVapidKey, error)
	CreateWebPushVAPIDKeys(ctx context.Context, arg db.CreateWebPushVAPIDKeysParams) (db.WebPushVapidKey, error)
	UpsertWebPushSubscription(ctx context.Context, arg db.UpsertWebPushSubscriptionParams) error
	ListWebPushSubscriptionsByUser(ctx context.Context, userID string) ([]db.WebPushSubscription, error)
	DeleteWebPushSubscription(ctx context.Context, arg db.DeleteWebPushSubscriptionParams) error
	CreateUserNotification(ctx context.Context, arg db.CreateUserNotificationParams) (db.UserNotification, error)
	UpsertChatUserNotificationByTag(ctx context.Context, arg db.UpsertChatUserNotificationByTagParams) (db.UserNotification, error)
	ListUserNotifications(ctx context.Context, arg db.ListUserNotificationsParams) ([]db.UserNotification, error)
	MarkAllUserNotificationsRead(ctx context.Context, userID string) error
	MarkUserNotificationsRead(ctx context.Context, arg db.MarkUserNotificationsReadParams) error
}

type WebPushService struct {
	publicKey  string
	privateKey string
	subscriber string
	store      WebPushStore
}

func NewSQLiteWebPushService(ctx context.Context, store WebPushStore, subscriber string) (*WebPushService, error) {
	if store == nil {
		return nil, errors.New("web push store is required")
	}
	keys, err := loadOrCreateVAPIDKeys(ctx, store)
	if err != nil {
		return nil, err
	}
	return &WebPushService{
		publicKey:  keys.PublicKey,
		privateKey: keys.PrivateKey,
		subscriber: subscriber,
		store:      store,
	}, nil
}

func MustNewSQLiteWebPushService(ctx context.Context, store WebPushStore, subscriber string) *WebPushService {
	service, err := NewSQLiteWebPushService(ctx, store, subscriber)
	if err != nil {
		panic(err)
	}
	return service
}

func loadOrCreateVAPIDKeys(ctx context.Context, store WebPushStore) (db.WebPushVapidKey, error) {
	keys, err := store.GetWebPushVAPIDKeys(ctx)
	if err == nil {
		return keys, nil
	}
	if err != sql.ErrNoRows {
		return db.WebPushVapidKey{}, fmt.Errorf("get VAPID keys: %w", err)
	}

	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return db.WebPushVapidKey{}, fmt.Errorf("generate VAPID keys: %w", err)
	}
	keys, err = store.CreateWebPushVAPIDKeys(ctx, db.CreateWebPushVAPIDKeysParams{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	})
	if err != nil {
		return db.WebPushVapidKey{}, fmt.Errorf("store VAPID keys: %w", err)
	}
	return keys, nil
}

func (s *WebPushService) PublicKey() string {
	if s == nil {
		return ""
	}
	return s.publicKey
}

func (s *WebPushService) Subscribe(ctx context.Context, userID string, subscription PushSubscription) error {
	if s == nil {
		return nil
	}
	if userID == "" {
		return errors.New("user ID is required")
	}
	if subscription.Endpoint == "" || subscription.Keys.Auth == "" || subscription.Keys.P256DH == "" {
		return errors.New("subscription endpoint and keys are required")
	}
	if err := validatePushEndpoint(subscription.Endpoint); err != nil {
		return err
	}

	return s.store.UpsertWebPushSubscription(ctx, db.UpsertWebPushSubscriptionParams{
		Endpoint:         subscription.Endpoint,
		UserID:           userID,
		Auth:             subscription.Keys.Auth,
		P256dh:           subscription.Keys.P256DH,
		ExpirationTimeMs: subscription.ExpirationTime,
	})
}

func validatePushEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return errors.New("subscription endpoint is invalid")
	}
	if parsed.Scheme != "https" || parsed.User != nil {
		return errors.New("subscription endpoint is not an allowed push service")
	}

	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	path := parsed.EscapedPath()
	if isAllowedPushEndpoint(host, path) {
		return nil
	}
	return errors.New("subscription endpoint is not an allowed push service")
}

func isAllowedPushEndpoint(host, path string) bool {
	switch {
	case host == "fcm.googleapis.com" && (strings.HasPrefix(path, "/fcm/") || strings.HasPrefix(path, "/wp/")):
		return true
	case host == "android.googleapis.com" && strings.HasPrefix(path, "/gcm/"):
		return true
	case hostMatches(host, "push.services.mozilla.com"):
		return true
	case hostMatches(host, "notify.windows.com"):
		return true
	case hostMatches(host, "push.apple.com"):
		return true
	default:
		return false
	}
}

func hostMatches(host, suffix string) bool {
	return host == suffix || strings.HasSuffix(host, "."+suffix)
}

func (s *WebPushService) Notify(ctx context.Context, userID string, payload Payload) error {
	if s == nil || userID == "" {
		return nil
	}

	policy := deliveryPolicyForKind(payload.Kind)
	if policy.Feed {
		if _, err := s.addNotification(ctx, userID, payload, policy); err != nil {
			return fmt.Errorf("store notification: %w", err)
		}
	}
	if policy.Push {
		message, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal push payload: %w", err)
		}
		s.dispatchPushAsync(userID, message)
	}

	return nil
}

func (s *WebPushService) dispatchPushAsync(userID string, message []byte) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), webPushSendTimeout)
		defer cancel()
		if err := s.sendPushNotifications(ctx, userID, message); err != nil {
			slog.Warn("web push delivery failed", "user_id", userID, "error", err)
		}
	}()
}

func (s *WebPushService) sendPushNotifications(ctx context.Context, userID string, message []byte) error {
	subscriptions, err := s.subscriptionsForUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list push subscriptions: %w", err)
	}
	var errs []error
	for _, subscription := range subscriptions {
		resp, err := webpush.SendNotificationWithContext(ctx, message, toWebPushSubscription(subscription), &webpush.Options{
			Subscriber:      s.subscriber,
			VAPIDPublicKey:  s.publicKey,
			VAPIDPrivateKey: s.privateKey,
			TTL:             300,
			Urgency:         webpush.UrgencyNormal,
		})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if resp == nil {
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
			if err := s.removeSubscription(ctx, userID, subscription.Endpoint); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if resp.StatusCode >= http.StatusBadRequest {
			errs = append(errs, fmt.Errorf("push endpoint returned %d", resp.StatusCode))
		}
	}
	return errors.Join(errs...)
}

func (s *WebPushService) ListNotifications(ctx context.Context, userID string, limit int) ([]UserNotification, error) {
	if s == nil || userID == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	rows, err := s.store.ListUserNotifications(ctx, db.ListUserNotificationsParams{
		UserID: userID,
		Limit:  int64(limit),
	})
	if err != nil {
		return nil, err
	}
	items := make([]UserNotification, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapUserNotification(row))
	}
	return items, nil
}

func (s *WebPushService) MarkNotificationsRead(ctx context.Context, userID string, ids []string) error {
	if s == nil || userID == "" {
		return nil
	}
	if len(ids) == 0 {
		return s.store.MarkAllUserNotificationsRead(ctx, userID)
	}
	return s.store.MarkUserNotificationsRead(ctx, db.MarkUserNotificationsReadParams{
		UserID: userID,
		Ids:    ids,
	})
}

func (s *WebPushService) addNotification(ctx context.Context, userID string, payload Payload, policy DeliveryPolicy) (db.UserNotification, error) {
	data, err := notificationData(payload.Data)
	if err != nil {
		return db.UserNotification{}, err
	}
	if policy.DebounceFeed && payload.Tag != "" {
		return s.store.UpsertChatUserNotificationByTag(ctx, db.UpsertChatUserNotificationByTagParams{
			ID:     uuid.NewString(),
			UserID: userID,
			Title:  payload.Title,
			Body:   optionalString(payload.Body),
			Url:    optionalString(payload.URL),
			Tag:    optionalString(payload.Tag),
			Kind:   optionalString(payload.Kind),
			PlayID: optionalString(payload.PlayID),
			Data:   data,
		})
	}
	return s.store.CreateUserNotification(ctx, db.CreateUserNotificationParams{
		ID:     uuid.NewString(),
		UserID: userID,
		Title:  payload.Title,
		Body:   optionalString(payload.Body),
		Url:    optionalString(payload.URL),
		Tag:    optionalString(payload.Tag),
		Kind:   optionalString(payload.Kind),
		PlayID: optionalString(payload.PlayID),
		Data:   data,
	})
}

func (s *WebPushService) subscriptionsForUser(ctx context.Context, userID string) ([]PushSubscription, error) {
	rows, err := s.store.ListWebPushSubscriptionsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	subscriptions := make([]PushSubscription, 0, len(rows))
	for _, row := range rows {
		subscriptions = append(subscriptions, PushSubscription{
			Endpoint:       row.Endpoint,
			ExpirationTime: row.ExpirationTimeMs,
			Keys: PushSubscriptionKeys{
				Auth:   row.Auth,
				P256DH: row.P256dh,
			},
		})
	}
	return subscriptions, nil
}

func (s *WebPushService) removeSubscription(ctx context.Context, userID, endpoint string) error {
	return s.store.DeleteWebPushSubscription(ctx, db.DeleteWebPushSubscriptionParams{
		UserID:   userID,
		Endpoint: endpoint,
	})
}

func mapUserNotification(row db.UserNotification) UserNotification {
	var readAt *string
	if row.ReadAt != nil {
		value := row.ReadAt.Format(time.RFC3339)
		readAt = &value
	}
	return UserNotification{
		ID:        row.ID,
		Title:     row.Title,
		Body:      stringValue(row.Body),
		URL:       stringValue(row.Url),
		Kind:      stringValue(row.Kind),
		PlayID:    stringValue(row.PlayID),
		ReadAt:    readAt,
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func notificationData(data map[string]string) (*string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal notification data: %w", err)
	}
	value := string(raw)
	return &value, nil
}

func toWebPushSubscription(subscription PushSubscription) *webpush.Subscription {
	return &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			Auth:   subscription.Keys.Auth,
			P256dh: subscription.Keys.P256DH,
		},
	}
}
