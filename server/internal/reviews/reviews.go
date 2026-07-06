// Package reviews holds the domain rules for post-game player reviews:
// the review window, the prop vocabulary, and (later) the prompt scheduler.
package reviews

import (
	"os"
	"time"
)

// Window is how long after a play ends its reviews stay editable.
const Window = 14 * 24 * time.Hour

// Window states as exposed to clients.
const (
	WindowNotOpen = "not_open"
	WindowOpen    = "open"
	WindowClosed  = "closed"
)

// WindowState reports where now falls in a play's review window. The Go time
// comparison is authoritative; SQL time filters are only used for list scans.
func WindowState(endsAt, now time.Time) (state string, closesAt time.Time) {
	closesAt = endsAt.Add(Window)
	if alwaysOpen() {
		return WindowOpen, closesAt
	}
	switch {
	case now.Before(endsAt):
		return WindowNotOpen, closesAt
	case now.After(closesAt):
		return WindowClosed, closesAt
	default:
		return WindowOpen, closesAt
	}
}

// alwaysOpen forces the review window open regardless of the play's times, a
// local-dev convenience (set DEV_REVIEWS_ALWAYS_OPEN=true) so review flows can
// be tested without waiting for a play to end. Never set in production.
func alwaysOpen() bool {
	return os.Getenv("DEV_REVIEWS_ALWAYS_OPEN") == "true"
}
