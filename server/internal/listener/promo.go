package listener

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"openplays/server/internal/db"
)

// PromoStore is the subset of db.Queries needed by the promo ticker.
type PromoStore interface {
	CountUpcomingPlays(ctx context.Context, arg db.CountUpcomingPlaysParams) (int64, error)
}

// PromoSender sends and deletes messages in a Telegram chat.
type PromoSender interface {
	SendMessage(ctx context.Context, chatUsername string, text string) (msgID int, err error)
	DeleteMessage(ctx context.Context, chatUsername string, msgID int) error
}

// Quiet hours — no messages posted during this window (in local timezone).
const (
	quietStart = 0 // midnight
	quietEnd   = 7 // 7am
)

// PromoTicker posts a promotional message to a Telegram group at fixed clock
// hours (cron-like), deleting the previous promo before sending a new one.
// Respects quiet hours (12am–7am local time).
//
// The ticker checks every minute whether it's time to post. A post is due
// when the current hour is a multiple of the interval and we haven't posted
// in this hour yet. This makes it deploy-independent — restarting the service
// doesn't shift the schedule.
type PromoTicker struct {
	store         PromoStore
	sender        PromoSender
	groupUsername string
	siteURL       string
	intervalHours int
	tz            *time.Location
	lastMsgID     int    // ID of the last promo message sent (0 = none)
	lastPostHour  string // "2026-04-16T14" — tracks the last hour we posted in
}

// NewPromoTicker creates a promo ticker.
// intervalHours is how many hours between posts (e.g. 3 = post at 7, 10, 13, 16, 19, 22).
// timezone is an IANA timezone string (e.g. "Asia/Singapore") used for scheduling.
func NewPromoTicker(store PromoStore, sender PromoSender, groupUsername, siteURL string, intervalHours int, timezone string) *PromoTicker {
	tz, err := time.LoadLocation(timezone)
	if err != nil {
		slog.Warn("promo: invalid timezone, using UTC", "timezone", timezone, "error", err)
		tz = time.UTC
	}
	return &PromoTicker{
		store:         store,
		sender:        sender,
		groupUsername: groupUsername,
		siteURL:       siteURL,
		intervalHours: intervalHours,
		tz:            tz,
	}
}

// Run starts the promo ticker. Blocks until ctx is cancelled.
func (t *PromoTicker) Run(ctx context.Context) {
	slog.Info("promo ticker started",
		"interval_hours", t.intervalHours,
		"group", t.groupUsername,
		"quiet_hours", fmt.Sprintf("%d:00–%d:00", quietStart, quietEnd),
		"timezone", t.tz.String(),
	)

	// Check every minute if it's time to post
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("promo ticker stopped")
			return
		case <-ticker.C:
			t.tick(ctx)
		}
	}
}

func (t *PromoTicker) tick(ctx context.Context) {
	now := time.Now().In(t.tz)
	hour := now.Hour()

	// Quiet hours
	if hour >= quietStart && hour < quietEnd {
		return
	}

	// Only post on hours that are multiples of the interval (starting from quietEnd)
	// e.g. intervalHours=3, quietEnd=7: post at 7, 10, 13, 16, 19, 22
	if (hour-quietEnd)%t.intervalHours != 0 {
		return
	}

	// Only post once per hour (dedup across the ~60 ticks per hour)
	hourKey := now.Format("2006-01-02T15")
	if t.lastPostHour == hourKey {
		return
	}

	t.post(ctx)
	t.lastPostHour = hourKey
}

func (t *PromoTicker) post(ctx context.Context) {
	count, err := t.store.CountUpcomingPlays(ctx, db.CountUpcomingPlaysParams{})
	if err != nil && err != sql.ErrNoRows {
		slog.Error("promo: failed to count plays", "error", err)
		return
	}

	if count == 0 {
		slog.Info("promo: no upcoming plays, skipping")
		return
	}

	// Delete previous promo message
	if t.lastMsgID != 0 {
		if err := t.sender.DeleteMessage(ctx, t.groupUsername, t.lastMsgID); err != nil {
			slog.Warn("promo: failed to delete previous message", "msg_id", t.lastMsgID, "error", err)
		}
		t.lastMsgID = 0
	}

	msg := fmt.Sprintf(
		"🏸 See all %d upcoming games on %s\n\nSort by distance or filter by date and level — find your next game!",
		count, t.siteURL,
	)

	msgID, err := t.sender.SendMessage(ctx, t.groupUsername, msg)
	if err != nil {
		slog.Error("promo: failed to send message", "error", err)
		return
	}

	t.lastMsgID = msgID
	slog.Info("promo: posted", "count", count, "msg_id", msgID, "group", t.groupUsername)
}
