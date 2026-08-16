package notifications

import (
	"context"
	"fmt"
)

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
