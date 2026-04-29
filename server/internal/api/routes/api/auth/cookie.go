package auth

import (
	"net/http"

	authpkg "openplays/server/internal/auth"
)

// CookieConfig controls session cookie behavior.
type CookieConfig struct {
	Secure bool // false for local dev (HTTP), true for prod (HTTPS)
}

// sessionCookie builds a session cookie with the given token and config.
func sessionCookie(token string, cfg CookieConfig) http.Cookie {
	return http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(authpkg.SessionDuration.Seconds()),
	}
}

// clearSessionCookie builds a cookie that deletes the session.
func clearSessionCookie(cfg CookieConfig) http.Cookie {
	return http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
}
