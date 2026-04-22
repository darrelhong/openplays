package telegramutils

import (
	"context"
	"fmt"

	"github.com/celestix/gotgproto"
	"github.com/gotd/td/tg"
)

// MessageSender implements listener.PromoSender using gotgproto.
type MessageSender struct {
	client *gotgproto.Client
}

// NewMessageSender creates a sender that can send/delete messages via the Telegram API.
func NewMessageSender(client *gotgproto.Client) *MessageSender {
	return &MessageSender{client: client}
}

func (s *MessageSender) SendMessage(ctx context.Context, chatUsername string, text string) (int, error) {
	resolved, err := s.client.CreateContext().ResolveUsername(chatUsername)
	if err != nil {
		return 0, fmt.Errorf("resolve username %q: %w", chatUsername, err)
	}

	msg, err := s.client.CreateContext().SendMessage(resolved.GetID(), &tg.MessagesSendMessageRequest{
		Message: text,
	})
	if err != nil {
		return 0, fmt.Errorf("send message: %w", err)
	}

	return msg.ID, nil
}

func (s *MessageSender) DeleteMessage(ctx context.Context, chatUsername string, msgID int) error {
	extCtx := s.client.CreateContext()
	resolved, err := extCtx.ResolveUsername(chatUsername)
	if err != nil {
		return fmt.Errorf("resolve username %q: %w", chatUsername, err)
	}

	inputChannel := resolved.GetInputChannel()
	if inputChannel == nil {
		return fmt.Errorf("resolved %q is not a channel", chatUsername)
	}

	_, err = extCtx.Raw.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
		Channel: inputChannel.(*tg.InputChannel),
		ID:      []int{msgID},
	})
	if err != nil {
		return fmt.Errorf("delete message %d: %w", msgID, err)
	}

	return nil
}
