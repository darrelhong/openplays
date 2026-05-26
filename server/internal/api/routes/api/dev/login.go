package dev

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/auth"
	"openplays/server/internal/db"
)

type LoginInput struct {
	Body struct {
		UserID string `json:"user_id" required:"true" doc:"Seed user ID to sign in as"`
	}
}

type LoginOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      struct {
		User         auth.User `json:"user"`
		SessionToken string    `json:"session_token"`
	}
}

type LoginStore interface {
	GetUserByID(ctx context.Context, id string) (db.User, error)
	CreateSession(ctx context.Context, arg db.CreateSessionParams) error
}

func RegisterLogin(api huma.API, store LoginStore, cfg Config) {
	huma.Register(api, huma.Operation{
		OperationID: "dev-login",
		Summary:     "Dev login",
		Description: "Create a local development session for an existing seed user. Only registered when DEV_AUTH_ENABLED=true.",
		Method:      http.MethodPost,
		Path:        "/login",
		Tags:        []string{"Dev"},
	}, func(ctx context.Context, input *LoginInput) (*LoginOutput, error) {
		user, err := store.GetUserByID(ctx, input.Body.UserID)
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("seed user not found")
		}
		if err != nil {
			slog.Error("dev auth: get user failed", "error", err)
			return nil, huma.Error500InternalServerError("failed to get user")
		}
		if user.Status != "active" {
			return nil, huma.Error403Forbidden("user is not active")
		}

		token, err := sessionToken()
		if err != nil {
			slog.Error("dev auth: token generation failed", "error", err)
			return nil, huma.Error500InternalServerError("failed to create session")
		}

		if err := store.CreateSession(ctx, db.CreateSessionParams{
			Token:     token,
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(auth.SessionDuration),
		}); err != nil {
			slog.Error("dev auth: create session failed", "error", err)
			return nil, huma.Error500InternalServerError("failed to create session")
		}

		out := &LoginOutput{}
		out.SetCookie = http.Cookie{
			Name:     "session",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   cfg.CookieSecure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(auth.SessionDuration.Seconds()),
		}
		out.Body.User = mapUser(user)
		out.Body.SessionToken = token
		return out, nil
	})
}

func sessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
