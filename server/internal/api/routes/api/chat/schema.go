package chat

import (
	"time"

	"openplays/server/internal/db"
)

const (
	conversationKindDM   = "dm"
	conversationKindPlay = "play"
)

type ChatUserSummary struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Username    *string `json:"username,omitempty"`
	PhotoURL    *string `json:"photo_url,omitempty"`
}

type ChatConversationSummary struct {
	ID          string              `json:"id"`
	Kind        string              `json:"kind" enum:"dm,play"`
	Title       string              `json:"title"`
	AvatarURL   *string             `json:"avatar_url,omitempty"`
	PlayID      *string             `json:"play_id,omitempty"`
	OtherUser   *ChatUserSummary    `json:"other_user,omitempty"`
	LastMessage *ChatMessagePreview `json:"last_message,omitempty"`
	UnreadCount int64               `json:"unread_count"`
	UpdatedAt   string              `json:"updated_at"`
}

type ChatMessagePreview struct {
	ID        int64           `json:"id"`
	Sender    ChatUserSummary `json:"sender"`
	Body      *string         `json:"body,omitempty"`
	DeletedAt *string         `json:"deleted_at,omitempty"`
	CreatedAt string          `json:"created_at"`
}

type ChatMessagePublic struct {
	ID        int64           `json:"id"`
	Sender    ChatUserSummary `json:"sender"`
	Body      *string         `json:"body,omitempty"`
	DeletedAt *string         `json:"deleted_at,omitempty"`
	CreatedAt string          `json:"created_at"`
	CanDelete bool            `json:"can_delete"`
}

type chatConversationListItem struct {
	Summary    ChatConversationSummary
	ActivityAt time.Time
}

func mapUserSummary(id, displayName string, username, photoURL *string) ChatUserSummary {
	return ChatUserSummary{
		ID:          id,
		DisplayName: displayName,
		Username:    username,
		PhotoURL:    photoURL,
	}
}

func mapDMConversation(row db.ListDMConversationsByUserRow) chatConversationListItem {
	otherUser := mapUserSummary(row.OtherUserID, row.OtherDisplayName, row.OtherUsername, row.OtherPhotoUrl)
	summary := ChatConversationSummary{
		ID:          row.ID,
		Kind:        row.Kind,
		Title:       otherUser.DisplayName,
		AvatarURL:   otherUser.PhotoURL,
		PlayID:      row.PlayID,
		OtherUser:   &otherUser,
		UnreadCount: row.UnreadCount,
		UpdatedAt:   row.UpdatedAt.Format(time.RFC3339),
	}
	activityAt := row.UpdatedAt
	if row.LastMessageID != nil && row.LastMessageSenderUserID != nil && row.LastMessageCreatedAt != nil {
		summary.LastMessage = mapMessagePreview(
			*row.LastMessageID,
			*row.LastMessageSenderUserID,
			row.LastMessageSenderDisplayName,
			row.LastMessageSenderUsername,
			row.LastMessageSenderPhotoUrl,
			row.LastMessageBody,
			row.LastMessageDeletedAt,
			*row.LastMessageCreatedAt,
		)
		activityAt = *row.LastMessageCreatedAt
	}
	return chatConversationListItem{Summary: summary, ActivityAt: activityAt}
}

func mapPlayConversation(row db.ListPlayConversationsByUserRow) chatConversationListItem {
	summary := ChatConversationSummary{
		ID:          row.ID,
		Kind:        row.Kind,
		Title:       row.Title,
		PlayID:      row.PlayID,
		UnreadCount: row.UnreadCount,
		UpdatedAt:   row.UpdatedAt.Format(time.RFC3339),
	}
	activityAt := row.UpdatedAt
	if row.LastMessageID != nil && row.LastMessageSenderUserID != nil && row.LastMessageCreatedAt != nil {
		summary.LastMessage = mapMessagePreview(
			*row.LastMessageID,
			*row.LastMessageSenderUserID,
			row.LastMessageSenderDisplayName,
			row.LastMessageSenderUsername,
			row.LastMessageSenderPhotoUrl,
			row.LastMessageBody,
			row.LastMessageDeletedAt,
			*row.LastMessageCreatedAt,
		)
		activityAt = *row.LastMessageCreatedAt
	}
	return chatConversationListItem{Summary: summary, ActivityAt: activityAt}
}

func mapMessagePreview(
	id int64,
	senderUserID string,
	senderDisplayName *string,
	senderUsername *string,
	senderPhotoURL *string,
	body *string,
	deletedAt *time.Time,
	createdAt time.Time,
) *ChatMessagePreview {
	return &ChatMessagePreview{
		ID:        id,
		Sender:    mapUserSummary(senderUserID, stringValue(senderDisplayName), senderUsername, senderPhotoURL),
		Body:      visibleMessageBody(body, deletedAt),
		DeletedAt: optionalTime(deletedAt),
		CreatedAt: createdAt.Format(time.RFC3339),
	}
}

func mapMessage(row db.GetChatMessageWithSenderRow, viewerID string) ChatMessagePublic {
	return ChatMessagePublic{
		ID: row.ID,
		Sender: mapUserSummary(
			row.SenderUserID,
			row.SenderDisplayName,
			row.SenderUsername,
			row.SenderPhotoUrl,
		),
		Body:      visibleMessageBody(row.Body, row.DeletedAt),
		DeletedAt: optionalTime(row.DeletedAt),
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
		CanDelete: row.DeletedAt == nil && row.SenderUserID == viewerID,
	}
}

func mapListMessage(row db.ListChatMessagesRow, viewerID string) ChatMessagePublic {
	return ChatMessagePublic{
		ID: row.ID,
		Sender: mapUserSummary(
			row.SenderUserID,
			row.SenderDisplayName,
			row.SenderUsername,
			row.SenderPhotoUrl,
		),
		Body:      visibleMessageBody(row.Body, row.DeletedAt),
		DeletedAt: optionalTime(row.DeletedAt),
		CreatedAt: row.CreatedAt.Format(time.RFC3339),
		CanDelete: row.DeletedAt == nil && row.SenderUserID == viewerID,
	}
}

func visibleMessageBody(body *string, deletedAt *time.Time) *string {
	if deletedAt != nil {
		return nil
	}
	return body
}

func optionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.RFC3339)
	return &formatted
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
