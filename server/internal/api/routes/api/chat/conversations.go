package chat

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"openplays/server/internal/api/authmw"
	"openplays/server/internal/db"
)

type ListChatConversationsInput struct {
	Limit int64 `query:"limit" minimum:"1" maximum:"50" default:"50" doc:"Maximum conversations to return"`
}

type ListChatConversationsOutput struct {
	Body struct {
		Items []ChatConversationSummary `json:"items"`
	}
}

type CreateDMConversationInput struct {
	Body struct {
		RecipientUserID string `json:"recipient_user_id" required:"true"`
	}
}

type CreateDMConversationOutput struct {
	Body ChatConversationSummary
}

type CreatePlayConversationInput struct {
	Body struct {
		PlayID string `json:"play_id" required:"true"`
	}
}

type CreatePlayConversationOutput struct {
	Body ChatConversationSummary
}

func RegisterConversations(api huma.API, store Store) {
	huma.Register(api, huma.Operation{
		OperationID: "list-chat-conversations",
		Summary:     "List chat conversations",
		Description: "Returns the current user's latest visible conversations.",
		Method:      http.MethodGet,
		Path:        "/conversations",
		Tags:        []string{"Chat"},
	}, func(ctx context.Context, input *ListChatConversationsInput) (*ListChatConversationsOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		limit := input.Limit
		if limit <= 0 {
			limit = 50
		}
		dmRows, err := store.ListDMConversationsByUser(ctx, db.ListDMConversationsByUserParams{
			ViewerID: user.ID,
			Limit:    limit,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list conversations")
		}
		playRows, err := store.ListPlayConversationsByUser(ctx, db.ListPlayConversationsByUserParams{
			ViewerID: user.ID,
			Limit:    limit,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list conversations")
		}

		items := make([]chatConversationListItem, 0, len(dmRows)+len(playRows))
		for _, row := range dmRows {
			items = append(items, mapDMConversation(row))
		}
		for _, row := range playRows {
			items = append(items, mapPlayConversation(row))
		}
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].ActivityAt.Equal(items[j].ActivityAt) {
				return items[i].Summary.ID > items[j].Summary.ID
			}
			return items[i].ActivityAt.After(items[j].ActivityAt)
		})
		if int64(len(items)) > limit {
			items = items[:limit]
		}

		out := &ListChatConversationsOutput{}
		out.Body.Items = make([]ChatConversationSummary, 0, len(items))
		for _, item := range items {
			out.Body.Items = append(out.Body.Items, item.Summary)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-dm-conversation",
		Summary:     "Create or get a direct message conversation",
		Description: "Idempotently creates or returns a DM conversation with another active user.",
		Method:      http.MethodPost,
		Path:        "/dms",
		Tags:        []string{"Chat"},
	}, func(ctx context.Context, input *CreateDMConversationInput) (*CreateDMConversationOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		recipientID := strings.TrimSpace(input.Body.RecipientUserID)
		if recipientID == "" {
			return nil, huma.Error422UnprocessableEntity("recipient_user_id is required")
		}
		if recipientID == user.ID {
			return nil, huma.Error422UnprocessableEntity("cannot message yourself")
		}

		recipient, err := store.GetUserByID(ctx, recipientID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error404NotFound("recipient not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get recipient")
		}
		if recipient.Status != "active" {
			return nil, huma.Error422UnprocessableEntity("recipient is not active")
		}
		if blocked, err := isBlocked(ctx, store, user.ID, recipient.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to check block status")
		} else if blocked {
			return nil, huma.Error403Forbidden("direct message is blocked")
		}

		key := dmKey(user.ID, recipient.ID)
		conversation, err := store.CreateDMConversation(ctx, db.CreateDMConversationParams{
			ID:    uuid.NewString(),
			DmKey: &key,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create conversation")
		}
		for _, participantID := range []string{user.ID, recipient.ID} {
			if err := store.CreateDMParticipant(ctx, db.CreateDMParticipantParams{
				ConversationID: conversation.ID,
				UserID:         participantID,
			}); err != nil {
				return nil, huma.Error500InternalServerError("failed to create conversation participant")
			}
		}

		otherUser := mapUserSummary(recipient.ID, recipient.DisplayName, recipient.Username, recipient.PhotoUrl)
		out := &CreateDMConversationOutput{}
		out.Body = ChatConversationSummary{
			ID:          conversation.ID,
			Kind:        conversation.Kind,
			Title:       recipient.DisplayName,
			AvatarURL:   recipient.PhotoUrl,
			OtherUser:   &otherUser,
			UnreadCount: 0,
			UpdatedAt:   conversation.UpdatedAt.Format(time.RFC3339),
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-play-chat-conversation",
		Summary:     "Create or get a play chat conversation",
		Description: "Idempotently creates or returns a play chat conversation for current roster users.",
		Method:      http.MethodPost,
		Path:        "/play-conversations",
		Tags:        []string{"Chat"},
	}, func(ctx context.Context, input *CreatePlayConversationInput) (*CreatePlayConversationOutput, error) {
		user := authmw.UserFromContext(ctx)
		if user == nil {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		playID := strings.TrimSpace(input.Body.PlayID)
		if playID == "" {
			return nil, huma.Error422UnprocessableEntity("play_id is required")
		}
		play, err := store.GetPlayByID(ctx, playID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error404NotFound("play not found")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to get play")
		}
		if err := authorizePlayChatByPlayID(ctx, store, playID, user.ID); err != nil {
			return nil, err
		}

		conversation, err := store.CreatePlayConversation(ctx, db.CreatePlayConversationParams{
			ID:     uuid.NewString(),
			PlayID: &playID,
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create conversation")
		}

		out := &CreatePlayConversationOutput{}
		out.Body = ChatConversationSummary{
			ID:          conversation.ID,
			Kind:        conversation.Kind,
			Title:       playConversationTitle(play),
			PlayID:      conversation.PlayID,
			UnreadCount: 0,
			UpdatedAt:   conversation.UpdatedAt.Format(time.RFC3339),
		}
		return out, nil
	})
}

func dmKey(userA, userB string) string {
	ids := []string{userA, userB}
	sort.Strings(ids)
	return ids[0] + ":" + ids[1]
}

func isBlocked(ctx context.Context, store Store, userA, userB string) (bool, error) {
	return store.IsBlocked(ctx, db.IsBlockedParams{
		BlockerID:   userA,
		BlockedID:   userB,
		BlockerID_2: userB,
		BlockedID_2: userA,
	})
}

func playConversationTitle(play db.GetPlayByIDRow) string {
	if play.Name != nil && *play.Name != "" {
		return *play.Name
	}
	return play.VenueName
}
