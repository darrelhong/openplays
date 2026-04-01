package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"openplays/server/internal/listener"
	"openplays/server/internal/listener/parser"

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

	pipeline := parser.NewPipeline(cfg.LLM)

	// Serialize LLM calls — local models can only handle one request at a time.
	var llmMu sync.Mutex

	client, err := gotgproto.NewClient(
		cfg.APIID,
		cfg.APIHash,
		gotgproto.ClientTypePhone(cfg.Phone),
		&gotgproto.ClientOpts{
			Session: sessionMaker.SqlSession(sqlite.Open("tele_session.db")),
		},
	)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
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

		// Extract sender info
		var fromID int64
		if peer, ok := msg.FromID.(*tg.PeerUser); ok {
			fromID = peer.UserID
		}

		senderName := resolveSenderName(update, fromID)

		msgTime := time.Unix(int64(msg.Date), 0).Local()
		ts := msgTime.Format("2006-01-02 15:04:05")

		// Print raw message
		fmt.Printf("\n╔══════════════════════════════════════════\n")
		fmt.Printf("║ [%s] Msg #%d | From: %s (%d)\n", ts, msg.ID, senderName, fromID)
		fmt.Printf("╠══════════════════════════════════════════\n")
		fmt.Printf("║ %s\n", msg.Message.Message)
		fmt.Printf("╠══════════════════════════════════════════\n")

		// Parse via LLM pipeline — serialized to avoid overwhelming the local model
		input := parser.MessageInput{
			Text:       msg.Message.Message,
			SenderName: senderName,
			Timestamp:  msgTime,
			Timezone:   cfg.TargetTelegramGroupTimezone,
		}

		llmMu.Lock()
		parseCtx, cancel := context.WithTimeout(context.Background(), cfg.LLM.Timeout)
		candidates, err := pipeline.Parse(parseCtx, input)
		cancel()
		llmMu.Unlock()

		if err != nil {
			fmt.Printf("║ ❌ Parse error: %v\n", err)
			fmt.Printf("╚══════════════════════════════════════════\n")
			return nil
		}

		if len(candidates) == 0 {
			fmt.Printf("║ ⚪ No plays extracted\n")
			fmt.Printf("╚══════════════════════════════════════════\n")
			return nil
		}

		fmt.Printf("║ ✅ %d play(s) extracted:\n", len(candidates))

		for i, c := range candidates {
			play := parser.ToPlay(&c, input)

			playJSON, _ := json.MarshalIndent(play, "║   ", "  ")
			fmt.Printf("║\n")
			fmt.Printf("║ 🏸 Play %d/%d:\n", i+1, len(candidates))
			fmt.Printf("║   %s\n", string(playJSON))
		}

		fmt.Printf("╚══════════════════════════════════════════\n")

		return nil
	}

	client.Dispatcher.AddHandler(handlers.NewMessage(filters.Message.Text,
		handleMessage))

	fmt.Println("Listening for messages... (LLM:", cfg.LLM.BaseURL, "model:", cfg.LLM.Model, ")")
	client.Idle()
}

// resolveSenderName extracts the user's display name from the update entities.
// Prefers username (linkable on Telegram), then full name, then fallback.
func resolveSenderName(update *ext.Update, userID int64) string {
	if user := update.EffectiveUser(); user != nil {
		if username, ok := user.GetUsername(); ok && username != "" {
			return username
		}
		first, _ := user.GetFirstName()
		last, _ := user.GetLastName()
		name := strings.TrimSpace(first + " " + last)
		if name != "" {
			return name
		}
	}
	if userID != 0 {
		return fmt.Sprintf("User_%d", userID)
	}
	return "Unknown"
}
