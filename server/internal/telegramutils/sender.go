// Package telegramutils provides helpers for working with Telegram user data.
package telegramutils

import "fmt"

// UserInfo holds the subset of Telegram user fields needed for sender resolution.
// Extracted from *tg.User by the caller so this package has no gotgproto dependency.
type UserInfo struct {
	Username  string // Telegram @username (may be empty)
	FirstName string
	LastName  string
}

// ResolveSender returns a Telegram @username and a display name from the given
// user info and numeric user ID.
//
// Username is the Telegram @username, empty if the user doesn't have one.
// Display name is first+last name when available, falling back to username,
// then "User_{id}", then "Unknown".
func ResolveSender(user *UserInfo, userID int64) (username, displayName string) {
	if user != nil {
		username = user.Username

		name := user.FirstName
		if user.LastName != "" {
			if name != "" {
				name += " "
			}
			name += user.LastName
		}
		if name != "" {
			displayName = name
		}
	}

	if displayName == "" {
		if username != "" {
			displayName = username
		} else if userID != 0 {
			displayName = fmt.Sprintf("User_%d", userID)
		} else {
			displayName = "Unknown"
		}
	}

	return username, displayName
}
