package chat

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
)

const maxMessageBodyLength = 2000

type ListChatMessagesInput struct {
	ID       string `path:"id" doc:"Conversation ID"`
	Limit    int64  `query:"limit" minimum:"1" maximum:"50" default:"50" doc:"Maximum messages to return"`
	BeforeID int64  `query:"before_id" minimum:"0" doc:"Fetch messages older than this message ID"`
}

type ListChatMessagesOutput struct {
	Body struct {
		Items []ChatMessagePublic `json:"items"`
	}
}

type SendChatMessageInput struct {
	ID   string `path:"id" doc:"Conversation ID"`
	Body struct {
		Body string `json:"body" required:"true" maxLength:"2000"`
	}
}

type SendChatMessageOutput struct {
	Body ChatMessagePublic
}

type MarkChatConversationReadInput struct {
	ID   string `path:"id" doc:"Conversation ID"`
	Body struct {
		LastReadMessageID int64 `json:"last_read_message_id" required:"true" minimum:"1"`
	}
}

type DeleteChatMessageInput struct {
	ID        string `path:"id" doc:"Conversation ID"`
	MessageID int64  `path:"messageID" doc:"Message ID"`
}

func RegisterMessages(api huma.API, store Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-chat-messages",
		Summary:     "List chat messages",
		Description: "Returns the latest visible messages for a conversation.",
		Method:      http.MethodGet,
		Path:        "/conversations/{id}/messages",
		Tags:        []string{"Chat"},
	}, func(ctx context.Context, input *ListChatMessagesInput) (*ListChatMessagesOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if _, err := authorizeConversation(ctx, store, input.ID, user.ID); err != nil {
			return nil, err
		}
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		rows, err := store.ListChatMessages(ctx, db.ListChatMessagesParams{
			ConversationID: input.ID,
			BeforeID:       input.BeforeID,
			Limit:          limit,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list messages")
		}
		slices.Reverse(rows)
		out := &ListChatMessagesOutput{}
		out.Body.Items = make([]ChatMessagePublic, 0, len(rows))
		for _, row := range rows {
			out.Body.Items = append(out.Body.Items, mapListMessage(row, user.ID))
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "send-chat-message",
		Summary:     "Send chat message",
		Description: "Sends a plain text message to a conversation.",
		Method:      http.MethodPost,
		Path:        "/conversations/{id}/messages",
		Tags:        []string{"Chat"},
	}, func(ctx context.Context, input *SendChatMessageInput) (*SendChatMessageOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if _, err := authorizeConversation(ctx, store, input.ID, user.ID); err != nil {
			return nil, err
		}
		body := strings.TrimSpace(input.Body.Body)
		if body == "" {
			return nil, huma.Error422UnprocessableEntity("message body cannot be empty")
		}
		if len(body) > maxMessageBodyLength {
			return nil, huma.Error422UnprocessableEntity("message body is too long")
		}
		message, err := store.CreateChatMessage(ctx, db.CreateChatMessageParams{
			ConversationID: input.ID,
			SenderUserID:   user.ID,
			Body:           &body,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to send message")
		}
		row, err := store.GetChatMessageWithSender(ctx, db.GetChatMessageWithSenderParams{
			ID:             message.ID,
			ConversationID: input.ID,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get message")
		}
		return &SendChatMessageOutput{Body: mapMessage(row, user.ID)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "mark-chat-conversation-read",
		Summary:     "Mark chat conversation read",
		Description: "Stores the current user's read cursor for a conversation.",
		Method:      http.MethodPost,
		Path:        "/conversations/{id}/read",
		Tags:        []string{"Chat"},
	}, func(ctx context.Context, input *MarkChatConversationReadInput) (*struct{}, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if _, err := authorizeConversation(ctx, store, input.ID, user.ID); err != nil {
			return nil, err
		}
		if input.Body.LastReadMessageID <= 0 {
			return nil, huma.Error422UnprocessableEntity("last_read_message_id must be positive")
		}
		if err := store.UpsertChatReadState(ctx, db.UpsertChatReadStateParams{
			ConversationID:    input.ID,
			UserID:            user.ID,
			LastReadMessageID: input.Body.LastReadMessageID,
		}); err != nil {
			return nil, huma.Error500InternalServerError("failed to mark conversation read")
		}
		return &struct{}{}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-chat-message",
		Summary:     "Delete chat message",
		Description: "Soft-deletes a message sent by the current user.",
		Method:      http.MethodDelete,
		Path:        "/conversations/{id}/messages/{messageID}",
		Tags:        []string{"Chat"},
	}, func(ctx context.Context, input *DeleteChatMessageInput) (*struct{}, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		if _, err := authorizeConversation(ctx, store, input.ID, user.ID); err != nil {
			return nil, err
		}
		if _, err := store.SoftDeleteChatMessageBySender(ctx, db.SoftDeleteChatMessageBySenderParams{
			ID:             input.MessageID,
			ConversationID: input.ID,
			SenderUserID:   user.ID,
		}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				row, getErr := store.GetChatMessageWithSender(ctx, db.GetChatMessageWithSenderParams{
					ID:             input.MessageID,
					ConversationID: input.ID,
				})
				if errors.Is(getErr, sql.ErrNoRows) {
					return nil, huma.Error404NotFound("message not found")
				}
				if getErr != nil {
					return nil, huma.Error500InternalServerError("failed to get message")
				}
				if row.SenderUserID != user.ID {
					return nil, huma.Error403Forbidden("cannot delete another user's message")
				}
				return &struct{}{}, nil
			}
			return nil, huma.Error500InternalServerError("failed to delete message")
		}
		return &struct{}{}, nil
	})
}

func authorizeConversation(ctx context.Context, store Store, conversationID, viewerID string) (db.ChatConversation, error) {
	conversation, err := store.GetChatConversation(ctx, conversationID)
	if errors.Is(err, sql.ErrNoRows) {
		return db.ChatConversation{}, huma.Error404NotFound("conversation not found")
	}
	if err != nil {
		return db.ChatConversation{}, huma.Error500InternalServerError("failed to get conversation")
	}

	switch conversation.Kind {
	case conversationKindDM:
		if _, err := authorizeDM(ctx, store, conversationID, viewerID); err != nil {
			return db.ChatConversation{}, err
		}
	case conversationKindPlay:
		if conversation.PlayID == nil {
			return db.ChatConversation{}, huma.Error500InternalServerError("play conversation is missing play_id")
		}
		if err := authorizePlayChatByPlayID(ctx, store, *conversation.PlayID, viewerID); err != nil {
			return db.ChatConversation{}, err
		}
	default:
		return db.ChatConversation{}, huma.Error500InternalServerError("unsupported conversation kind")
	}
	return conversation, nil
}

func authorizeDM(ctx context.Context, store Store, conversationID, viewerID string) (db.GetDMConversationPeerRow, error) {
	peer, err := store.GetDMConversationPeer(ctx, db.GetDMConversationPeerParams{
		ConversationID: conversationID,
		ViewerID:       viewerID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return db.GetDMConversationPeerRow{}, huma.Error404NotFound("conversation not found")
	}
	if err != nil {
		return db.GetDMConversationPeerRow{}, huma.Error500InternalServerError("failed to get conversation")
	}
	if peer.Status != "active" {
		return db.GetDMConversationPeerRow{}, huma.Error403Forbidden("conversation is not available")
	}
	if blocked, err := isBlocked(ctx, store, viewerID, peer.ID); err != nil {
		return db.GetDMConversationPeerRow{}, huma.Error500InternalServerError("failed to check block status")
	} else if blocked {
		return db.GetDMConversationPeerRow{}, huma.Error403Forbidden("direct message is blocked")
	}
	return peer, nil
}

func authorizePlayChatByPlayID(ctx context.Context, store Store, playID, viewerID string) error {
	allowed, err := store.IsPlayChatMember(ctx, db.IsPlayChatMemberParams{
		PlayID: playID,
		UserID: viewerID,
	})
	if err != nil {
		return huma.Error500InternalServerError("failed to check play chat access")
	}
	if !allowed {
		return huma.Error403Forbidden("play chat is only available to current roster users")
	}
	return nil
}
