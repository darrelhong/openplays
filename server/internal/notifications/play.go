package notifications

import (
	"context"
	"errors"
	"fmt"

	"openplays/server/internal/db"
)

type PlaySnapshot struct {
	ID        string
	Name      *string
	VenueName string
}

func PlaySnapshotFromDB(play db.GetPlayByIDRow) PlaySnapshot {
	return PlaySnapshot{
		ID:        play.ID,
		Name:      play.Name,
		VenueName: play.VenueName,
	}
}

// NotifyHostsJoinRequested fires for every join that lands in the pending
// queue, on classic and require-waitlist plays alike.
func NotifyHostsJoinRequested(ctx context.Context, sender Sender, play PlaySnapshot, hostUserIDs []string, playerUserID, playerName string) error {
	body := fmt.Sprintf("%s requested to join", notificationName(playerName, "Someone"))
	payload := Payload{
		Title:  playNotificationTitle(play),
		Body:   body,
		URL:    "/play/" + play.ID,
		Tag:    "play:" + play.ID + ":request:" + playerUserID,
		Kind:   "play.join_requested",
		PlayID: play.ID,
	}
	return notifyHosts(ctx, sender, payload, hostUserIDs, playerUserID)
}

func NotifyPlayerMovedToWaitlist(ctx context.Context, sender Sender, play PlaySnapshot, playerUserID string) error {
	if sender == nil || playerUserID == "" {
		return nil
	}
	return sender.Notify(ctx, playerUserID, Payload{
		Title:  playNotificationTitle(play),
		Body:   "You were added to the waitlist",
		URL:    "/play/" + play.ID,
		Tag:    "play:" + play.ID + ":waitlisted:" + playerUserID,
		Kind:   "play.moved_to_waitlist",
		PlayID: play.ID,
	})
}

// NotifyReviewPrompt nudges a participant to review their co-players once a
// play has ended. Sent at most once per (play, user) by the reviews.Prompter.
func NotifyReviewPrompt(ctx context.Context, sender Sender, play PlaySnapshot, userID string) error {
	if sender == nil || userID == "" {
		return nil
	}
	return sender.Notify(ctx, userID, Payload{
		Title:  playNotificationTitle(play),
		Body:   "How was your game? Give props and shoutouts to show your appreciation!",
		URL:    "/play/" + play.ID,
		Tag:    "play:" + play.ID + ":review_prompt",
		Kind:   "play.review_prompt",
		PlayID: play.ID,
	})
}

func NotifyPlayerAdded(ctx context.Context, sender Sender, play PlaySnapshot, playerUserID string) error {
	if sender == nil || playerUserID == "" {
		return nil
	}
	return sender.Notify(ctx, playerUserID, Payload{
		Title:  playNotificationTitle(play),
		Body:   "You were added to the game",
		URL:    "/play/" + play.ID,
		Tag:    "play:" + play.ID + ":added:" + playerUserID,
		Kind:   "play.player_added",
		PlayID: play.ID,
	})
}

func NotifyPlayerInvited(ctx context.Context, sender Sender, play PlaySnapshot, playerUserID, hostName string) error {
	if sender == nil || playerUserID == "" {
		return nil
	}
	return sender.Notify(ctx, playerUserID, Payload{
		Title:  playNotificationTitle(play),
		Body:   fmt.Sprintf("%s invited you to the game", notificationName(hostName, "Someone")),
		URL:    "/play/" + play.ID,
		Tag:    "play:" + play.ID + ":invited:" + playerUserID,
		Kind:   "play.invited",
		PlayID: play.ID,
	})
}

func NotifyPlayCancelled(ctx context.Context, sender Sender, play PlaySnapshot, recipientUserIDs []string) error {
	if sender == nil {
		return nil
	}
	payload := Payload{
		Title:  playNotificationTitle(play),
		Body:   "This game was cancelled",
		URL:    "/play/" + play.ID,
		Tag:    "play:" + play.ID + ":cancelled",
		Kind:   "play.cancelled",
		PlayID: play.ID,
	}
	return notifyUsers(ctx, sender, payload, recipientUserIDs)
}

// NotifyFollowersPlayHosted tells accepted followers when a user hosts a
// public play. Unlisted plays intentionally never call this helper.
func NotifyFollowersPlayHosted(ctx context.Context, sender FollowerNotifier, actorUserID, actorName string, play PlaySnapshot) error {
	if sender == nil || actorUserID == "" || play.ID == "" {
		return nil
	}
	return sender.NotifyFollowers(ctx, actorUserID, Payload{
		Title:  playNotificationTitle(play),
		Body:   fmt.Sprintf("%s is hosting a game", notificationName(actorName, "Someone")),
		URL:    "/play/" + play.ID,
		Tag:    "follow:hosted:" + play.ID,
		Kind:   "follow.play_hosted",
		PlayID: play.ID,
	})
}

// NotifyFollowersPlayerJoined tells accepted followers when a player enters
// the public roster. Added and confirmed are the same public-roster event; a
// later added-to-confirmed transition must not emit a second notification.
func NotifyFollowersPlayerJoined(ctx context.Context, sender FollowerNotifier, play PlaySnapshot, playerUserID, playerName string) error {
	if sender == nil || playerUserID == "" || play.ID == "" {
		return nil
	}
	return sender.NotifyFollowers(ctx, playerUserID, Payload{
		Title:  playNotificationTitle(play),
		Body:   fmt.Sprintf("%s joined the game", notificationName(playerName, "Someone")),
		URL:    "/play/" + play.ID,
		Tag:    fmt.Sprintf("follow:joined:%s:%s", playerUserID, play.ID),
		Kind:   "follow.play_joined",
		PlayID: play.ID,
	})
}

func NotifyHostsInviteAccepted(ctx context.Context, sender Sender, play PlaySnapshot, hostUserIDs []string, playerUserID, playerName string) error {
	body := fmt.Sprintf("%s accepted your invitation", notificationName(playerName, "Someone"))
	payload := Payload{
		Title:  playNotificationTitle(play),
		Body:   body,
		URL:    "/play/" + play.ID,
		Tag:    "play:" + play.ID + ":invite_accepted:" + playerUserID,
		Kind:   "play.invite_accepted",
		PlayID: play.ID,
	}
	return notifyHosts(ctx, sender, payload, hostUserIDs, playerUserID)
}

func NotifyHostsInviteDeclined(ctx context.Context, sender Sender, play PlaySnapshot, hostUserIDs []string, playerUserID, playerName string) error {
	body := fmt.Sprintf("%s declined your invitation", notificationName(playerName, "Someone"))
	payload := Payload{
		Title:  playNotificationTitle(play),
		Body:   body,
		URL:    "/play/" + play.ID,
		Tag:    "play:" + play.ID + ":invite_declined:" + playerUserID,
		Kind:   "play.invite_declined",
		PlayID: play.ID,
	}
	return notifyHosts(ctx, sender, payload, hostUserIDs, playerUserID)
}

func NotifyHostsPlayerJoined(ctx context.Context, sender Sender, play PlaySnapshot, hostUserIDs []string, playerUserID, playerName string) error {
	body := fmt.Sprintf("%s joined the game", notificationName(playerName, "Someone"))
	payload := Payload{
		Title:  playNotificationTitle(play),
		Body:   body,
		URL:    "/play/" + play.ID,
		Tag:    "play:" + play.ID + ":joined:" + playerUserID,
		Kind:   "play.player_joined",
		PlayID: play.ID,
	}
	return notifyHosts(ctx, sender, payload, hostUserIDs, playerUserID)
}

func NotifyHostsPlayerConfirmed(ctx context.Context, sender Sender, play PlaySnapshot, hostUserIDs []string, playerUserID, playerName string) error {
	body := fmt.Sprintf("%s confirmed their spot", notificationName(playerName, "Someone"))
	payload := Payload{
		Title:  playNotificationTitle(play),
		Body:   body,
		URL:    "/play/" + play.ID,
		Tag:    "play:" + play.ID + ":confirmed:" + playerUserID,
		Kind:   "play.player_confirmed",
		PlayID: play.ID,
	}
	return notifyHosts(ctx, sender, payload, hostUserIDs, playerUserID)
}

func NotifyHostsPlayerLeft(ctx context.Context, sender Sender, play PlaySnapshot, hostUserIDs []string, playerUserID, playerName string) error {
	body := fmt.Sprintf("%s left the game", notificationName(playerName, "Someone"))
	payload := Payload{
		Title:  playNotificationTitle(play),
		Body:   body,
		URL:    "/play/" + play.ID,
		Tag:    "play:" + play.ID + ":left:" + playerUserID,
		Kind:   "play.player_left",
		PlayID: play.ID,
	}
	return notifyHosts(ctx, sender, payload, hostUserIDs, playerUserID)
}

func notifyHosts(ctx context.Context, sender Sender, payload Payload, hostUserIDs []string, excludeUserID string) error {
	if sender == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(hostUserIDs))
	var errs []error
	for _, userID := range hostUserIDs {
		if userID == "" || userID == excludeUserID {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		if err := sender.Notify(ctx, userID, payload); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func notifyUsers(ctx context.Context, sender Sender, payload Payload, userIDs []string) error {
	if sender == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(userIDs))
	var errs []error
	for _, userID := range userIDs {
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		if err := sender.Notify(ctx, userID, payload); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func playNotificationTitle(play PlaySnapshot) string {
	if play.Name != nil && *play.Name != "" {
		return *play.Name
	}
	if play.VenueName != "" {
		return play.VenueName
	}
	return "OpenPlays game"
}

func notificationName(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
