package chat

import (
	"context"

	"openplays/server/internal/db"
)

type Store interface {
	GetUserByID(ctx context.Context, id string) (db.User, error)
	IsBlocked(ctx context.Context, arg db.IsBlockedParams) (bool, error)
	GetPlayByID(ctx context.Context, id string) (db.GetPlayByIDRow, error)
	GetChatConversation(ctx context.Context, id string) (db.ChatConversation, error)
	CreateDMConversation(ctx context.Context, arg db.CreateDMConversationParams) (db.ChatConversation, error)
	CreateDMParticipant(ctx context.Context, arg db.CreateDMParticipantParams) error
	CreatePlayConversation(ctx context.Context, arg db.CreatePlayConversationParams) (db.ChatConversation, error)
	ListDMConversationsByUser(ctx context.Context, arg db.ListDMConversationsByUserParams) ([]db.ListDMConversationsByUserRow, error)
	ListPlayConversationsByUser(ctx context.Context, arg db.ListPlayConversationsByUserParams) ([]db.ListPlayConversationsByUserRow, error)
	GetDMConversationPeer(ctx context.Context, arg db.GetDMConversationPeerParams) (db.GetDMConversationPeerRow, error)
	IsPlayChatMember(ctx context.Context, arg db.IsPlayChatMemberParams) (bool, error)
	CreateChatMessage(ctx context.Context, arg db.CreateChatMessageParams) (db.ChatMessage, error)
	GetChatMessageWithSender(ctx context.Context, arg db.GetChatMessageWithSenderParams) (db.GetChatMessageWithSenderRow, error)
	ListChatMessages(ctx context.Context, arg db.ListChatMessagesParams) ([]db.ListChatMessagesRow, error)
	SoftDeleteChatMessageBySender(ctx context.Context, arg db.SoftDeleteChatMessageBySenderParams) (db.ChatMessage, error)
	UpsertChatReadState(ctx context.Context, arg db.UpsertChatReadStateParams) error
}
