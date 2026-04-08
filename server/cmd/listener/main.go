package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"openplays/server/internal/db"
	"openplays/server/internal/geo"
	"openplays/server/internal/google"
	"openplays/server/internal/listener"
	"openplays/server/internal/listener/parser"
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
	worker := listener.NewWorker(queries, pipeline, resolver, cfg.TargetTelegramGroupTimezone)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Run(ctx)

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

		var fromID int64
		if peer, ok := msg.FromID.(*tg.PeerUser); ok {
			fromID = peer.UserID
		}

		senderName := resolveSenderName(update, fromID)
		msgTime := time.Unix(int64(msg.Date), 0).UTC()

		msgID := fmt.Sprintf("%d", msg.ID)
		group := channel.Username

		result, err := listener.HandleMessage(ctx, queries, "telegram", senderName, msg.Message.Message, msgTime, &msgID, &group)
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
	fmt.Printf("Group: %s (%s)\n", cfg.TargetTelegramGroupUsername, cfg.TargetTelegramGroupTimezone)
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
