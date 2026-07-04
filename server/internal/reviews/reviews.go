// Package reviews holds the domain rules for post-game player reviews:
// the review window, the prop vocabulary, and (later) the prompt scheduler.
package reviews

import "time"

const (
	// Window is how long after a play ends its reviews stay editable.
	Window = 14 * 24 * time.Hour
	// PromptDelay is how long after a play ends the review prompt fires.
	PromptDelay = time.Hour
)

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
	switch {
	case now.Before(endsAt):
		return WindowNotOpen, closesAt
	case now.After(closesAt):
		return WindowClosed, closesAt
	default:
		return WindowOpen, closesAt
	}
}
