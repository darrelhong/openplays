package main

import (
	"fmt"
	"log"
	"time"

	"openplays/server/internal/listener"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/ext"
	"github.com/celestix/gotgproto/sessionMaker"
	"github.com/celestix/gotgproto/types"
	"github.com/glebarez/sqlite"
	"github.com/gotd/td/tg"
)

func main() {
	cfg, err := listener.LoadConfig()

	if err != nil {
		log.Fatal(err)
	}

	client, err := gotgproto.NewClient(
		cfg.APIID,
		cfg.APIHash,
		gotgproto.ClientTypePhone(cfg.Phone),
		&gotgproto.ClientOpts{
			Session: sessionMaker.SqlSession(sqlite.Open("tele_session.db")),
		},
	)
	if err != nil {
		log.Fatal("failed to create client: %w", err)
	}

	handleMessage := func(ctx *ext.Context, update *ext.Update) error {
		if !filters.Supergroup(update) {
			return nil
		}

		channel, ok := update.EffectiveChat().(*types.Channel)
		if !ok || channel.Username != cfg.TargetTelegramGroupUsername {
			return nil
		}

		msg := update.EffectiveMessage
		if msg == nil {
			return nil
		}

		// Format timestamp

		// Extract sender and channel IDs
		var fromID, channelID int64
		if peer, ok := msg.FromID.(*tg.PeerUser); ok {
			fromID = peer.UserID
		}
		if peer, ok := msg.PeerID.(*tg.PeerChannel); ok {
			channelID = peer.ChannelID
		}

		ts := time.Unix(int64(msg.Date), 0).Local().Format("2006-01-02 15:04:05")
		fmt.Printf("\n[%s] Msg #%d\n", ts, msg.ID)
		fmt.Printf("From: User %d | Channel: %d\n", fromID, channelID)
		fmt.Printf("%s\n", msg.Message.Message)
		fmt.Println("───────────────────────────────")

		return nil
	}

	client.Dispatcher.AddHandler(handlers.NewMessage(filters.Message.Text,
		handleMessage))

	fmt.Println("Listening for messages...")
	client.Idle()
}
