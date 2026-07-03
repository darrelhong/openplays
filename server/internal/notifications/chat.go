package notifications

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	ChatMessageKind = "chat.message"
	chatPreviewMax  = 160
)

func NotifyDMChatMessage(ctx context.Context, sender Sender, conversationID, recipientUserID, senderUserID, senderName, messageBody string) error {
	if sender == nil || conversationID == "" || recipientUserID == "" {
		return nil
	}
	return sender.Notify(ctx, recipientUserID, Payload{
		Title: notificationName(senderName, "New message"),
		Body:  chatMessagePreview(messageBody),
		URL:   "/chat/" + conversationID,
		Tag:   chatNotificationTag(conversationID),
		Kind:  ChatMessageKind,
		Data:  chatNotificationData(conversationID, "dm", senderUserID),
	})
}

func NotifyPlayChatMessage(ctx context.Context, sender Sender, play PlaySnapshot, conversationID string, recipientUserIDs []string, senderUserID, senderName, messageBody string) error {
	if sender == nil || play.ID == "" || conversationID == "" {
		return nil
	}
	body := fmt.Sprintf("%s: %s", notificationName(senderName, "Someone"), chatMessagePreview(messageBody))
	payload := Payload{
		Title:  playNotificationTitle(play),
		Body:   body,
		URL:    "/chat/" + conversationID,
		Tag:    chatNotificationTag(conversationID),
		Kind:   ChatMessageKind,
		PlayID: play.ID,
		Data:   chatNotificationData(conversationID, "play", senderUserID),
	}

	var errs []error
	for _, userID := range recipientUserIDs {
		if userID == "" || userID == senderUserID {
			continue
		}
		if err := sender.Notify(ctx, userID, payload); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func chatNotificationTag(conversationID string) string {
	return "chat:" + conversationID
}

func chatNotificationData(conversationID, conversationKind, senderUserID string) map[string]string {
	return map[string]string{
		"conversation_id":   conversationID,
		"conversation_kind": conversationKind,
		"sender_user_id":    senderUserID,
	}
}

func chatMessagePreview(messageBody string) string {
	preview := strings.Join(strings.Fields(messageBody), " ")
	if preview == "" {
		return "New message"
	}
	runes := []rune(preview)
	if len(runes) <= chatPreviewMax {
		return preview
	}
	return string(runes[:chatPreviewMax]) + "..."
}
