package notifications

import (
	"context"
	"fmt"
	"log/slog"

	"openplays/server/internal/db"
)

// FollowerFanOutSender stores one notification for every eligible follower and
// dispatches the corresponding push payload to each recipient.
type FollowerFanOutSender interface {
	FanOutFollowerNotifications(context.Context, db.FanOutFollowerNotificationsParams) error
	ListAcceptedFollowerIDs(context.Context, string) ([]string, error)
	DispatchFollowerPush(userID string, payload Payload)
}

// FanOutToFollowers writes feed rows for every accepted follower in one
// statement, then dispatches push per follower. It returns immediately; the
// work runs detached, because a play creator with many followers must not
// make play creation slow.
func FanOutToFollowers(sender FollowerFanOutSender, actorUserID string, payload Payload) {
	if sender == nil || actorUserID == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), webPushSendTimeout)
		defer cancel()

		if err := sender.FanOutFollowerNotifications(ctx, db.FanOutFollowerNotificationsParams{
			ActorUserID: actorUserID,
			Title:       payload.Title,
			Body:        optionalString(payload.Body),
			Url:         optionalString(payload.URL),
			Tag:         optionalString(payload.Tag),
			Kind:        optionalString(payload.Kind),
			PlayID:      optionalString(payload.PlayID),
		}); err != nil {
			slog.Warn("follower notification fan-out failed", "actor_user_id", actorUserID, "error", err)
			return
		}

		followerIDs, err := sender.ListAcceptedFollowerIDs(ctx, actorUserID)
		if err != nil {
			slog.Warn("list followers for push fan-out failed", "actor_user_id", actorUserID, "error", err)
			return
		}
		for _, followerID := range followerIDs {
			sender.DispatchFollowerPush(followerID, payload)
		}
	}()
}

// NotifyFollowStarted tells the followee that a new accepted follow began.
func NotifyFollowStarted(ctx context.Context, sender Sender, followeeID, followerID, followerName, followerUsername string) error {
	if sender == nil || followeeID == "" {
		return nil
	}
	return sender.Notify(ctx, followeeID, Payload{
		Title: notificationName(followerName, "Someone"),
		Body:  "started following you",
		URL:   "/" + followerUsername,
		Tag:   fmt.Sprintf("follow:started:%s", followerID),
		Kind:  "follow.started",
	})
}

// NotifyFollowRequested tells the followee that a new pending request arrived.
func NotifyFollowRequested(ctx context.Context, sender Sender, followeeID, followerID, followerName string) error {
	if sender == nil || followeeID == "" {
		return nil
	}
	return sender.Notify(ctx, followeeID, Payload{
		Title: notificationName(followerName, "Someone"),
		Body:  "requested to follow you",
		URL:   "/profile/follow-requests",
		Tag:   fmt.Sprintf("follow:requested:%s", followerID),
		Kind:  "follow.requested",
	})
}

// NotifyFollowAccepted tells the requester that their pending follow was accepted.
func NotifyFollowAccepted(ctx context.Context, sender Sender, followerID, followeeName, followeeUsername string) error {
	if sender == nil || followerID == "" {
		return nil
	}
	return sender.Notify(ctx, followerID, Payload{
		Title: notificationName(followeeName, "Someone"),
		Body:  "accepted your follow request",
		URL:   "/" + followeeUsername,
		Tag:   fmt.Sprintf("follow:accepted:%s", followerID),
		Kind:  "follow.accepted",
	})
}
