package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/geo"
	"openplays/server/internal/google"
	"openplays/server/internal/listener"
	"openplays/server/internal/listener/parser"
	"openplays/server/internal/telegramutils"
	"openplays/server/internal/venue"

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

	sqlDb, err := sql.Open("sqlite", cfg.DBURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer sqlDb.Close()

	queries := db.New(sqlDb)
	pipeline := parser.NewPipeline(cfg.LLM)

	// --- Geocoder: uncomment ONE provider or leave both commented to disable ---

	var geocoder geo.Coder

	// Option A: Google Places (5,000 free requests/month, requires API key)
	if cfg.Google.APIKey != "" {
		geocoder = google.NewClient(cfg.Google)
		log.Println("Geocoder: Google Places enabled")
	}

	// Option B: OneMap (Singapore government API, free, requires email/password)
	// if cfg.OneMap.Email != "" && cfg.OneMap.Password != "" {
	// 	geocoder = onemap.NewClient(cfg.OneMap)
	// 	log.Println("Geocoder: OneMap enabled")
	// }

	if geocoder == nil {
		log.Println("Geocoder: disabled (no credentials configured)")
	}

	// Suppress unused import when OneMap is commented out.
	_ = google.Config{}

	resolver := venue.NewResolver(queries, geocoder)
	worker := listener.NewWorker(queries, pipeline, resolver, cfg.TelegramGroupTimezone)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Run(ctx)

	client, err := gotgproto.NewClient(
		cfg.TelegramAPIID,
		cfg.TelegramAPIHash,
		gotgproto.ClientTypePhone(cfg.TelegramUserPhone),
		&gotgproto.ClientOpts{
			Session: sessionMaker.SqlSession(sqlite.Open(cfg.TelegramSessionDB)),
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
		if !ok || channel.Username != cfg.TelegramGroupUsername {
			return nil
		}

		msg := update.EffectiveMessage
		if msg == nil {
			return nil
		}

		var fromID int64
		if peer, ok := msg.FromID.(*tg.PeerUser); ok {
			fromID = peer.UserID
		}

		// Telegram only provides the @username in the update entities if the user
		// has one set AND the update includes a full (non-min) user object.
		// For supergroup messages, Telegram typically sends min user objects which
		// lack the username. Resolving via API requires access hashes that are
		// session-scoped and not linkable — so source_sender_username will be null
		// for most users. This is a Telegram platform limitation.
		var userInfo *telegramutils.UserInfo
		if user := update.EffectiveUser(); user != nil {
			ui := telegramutils.UserInfo{}
			if u, ok := user.GetUsername(); ok {
				ui.Username = u
			}
			ui.FirstName, _ = user.GetFirstName()
			ui.LastName, _ = user.GetLastName()
			userInfo = &ui
		}

		senderUsername, senderName := telegramutils.ResolveSender(userInfo, fromID)
		msgTime := time.Unix(int64(msg.Date), 0).UTC()

		msgID := fmt.Sprintf("%d", msg.ID)
		group := channel.Username

		result, err := listener.HandleMessage(ctx, queries, "telegram", senderUsername, senderName, msg.Message.Message, msgTime, &msgID, &group)
		if err != nil {
			log.Printf("failed to handle message: %v", err)
			return nil
		}
		if result == listener.HandleSkipped {
			return nil
		}

		worker.Notify()

		return nil
	}

	client.Dispatcher.AddHandler(handlers.NewMessage(filters.Message.Text,
		handleMessage))

	fmt.Println("Listening for messages...")
	fmt.Printf("LLM: %s (model: %s)\n", cfg.LLM.BaseURL, cfg.LLM.Model)
	fmt.Printf("Group: %s (%s)\n", cfg.TelegramGroupUsername, cfg.TelegramGroupTimezone)
	client.Idle()
}
