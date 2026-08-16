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
