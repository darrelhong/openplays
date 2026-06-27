package plays_test

import (
	"context"

	"openplays/server/internal/notifications"
)

type notificationCall struct {
	userID  string
	payload notifications.Payload
}

type fakeNotificationSender struct {
	calls []notificationCall
}

func (f *fakeNotificationSender) Notify(_ context.Context, userID string, payload notifications.Payload) error {
	f.calls = append(f.calls, notificationCall{
		userID:  userID,
		payload: payload,
	})
	return nil
}
