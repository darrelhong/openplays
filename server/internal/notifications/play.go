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

func NotifyHostWaitlistJoined(ctx context.Context, sender Sender, play PlaySnapshot, hostUserIDs []string, playerName string) error {
	if sender == nil {
		return nil
	}
	body := fmt.Sprintf("%s joined the waitlist", notificationName(playerName, "Someone"))
	payload := Payload{
		Title:  playNotificationTitle(play),
		Body:   body,
		URL:    "/play/" + play.ID,
		Tag:    "play:" + play.ID + ":waitlist",
		Kind:   "play.waitlist_joined",
		PlayID: play.ID,
	}

	seen := make(map[string]struct{}, len(hostUserIDs))
	var errs []error
	for _, userID := range hostUserIDs {
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
