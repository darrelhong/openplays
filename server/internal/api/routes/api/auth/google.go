package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	authpkg "openplays/server/internal/auth"
)

type GoogleInput struct {
	Body struct {
		Credential string `json:"credential" doc:"Google ID token from GIS Sign-In" required:"true"`
	}
}

type GoogleOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      struct {
		User         authpkg.User `json:"user"`
		SessionToken string       `json:"session_token"`
	}
}

// RegisterGoogle registers POST /auth/google.
func RegisterGoogle(api huma.API, svc *authpkg.Service, verifier *authpkg.GoogleVerifier, cookieCfg CookieConfig) {
	huma.Register(api, huma.Operation{
		OperationID: "auth-google",
		Summary:     "Authenticate with Google",
		Description: "Verify a Google ID token, create/update user, return session.",
		Method:      http.MethodPost,
		Path:        "/google",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *GoogleInput) (*GoogleOutput, error) {
		claims, err := verifier.Verify(input.Body.Credential)
		if err != nil {
			slog.Warn("auth: google token verification failed", "error", err)
			return nil, huma.Error401Unauthorized("invalid Google token")
		}

		identity := authpkg.Identity{
			Provider:    authpkg.ProviderGoogle,
			ProviderID:  claims.Subject,
			Email:       claims.Email,
			DisplayName: claims.Name,
			PhotoURL:    claims.Picture,
		}

		result, err := svc.Login(ctx, identity)
		if err != nil {
			if errors.Is(err, authpkg.ErrAccountBanned) {
				return nil, huma.Error403Forbidden(err.Error())
			}
			slog.Error("auth: login failed", "error", err)
			return nil, huma.Error500InternalServerError("login failed")
		}

		out := &GoogleOutput{}
		out.SetCookie = sessionCookie(result.SessionToken, cookieCfg)
		out.Body.User = result.User
		out.Body.SessionToken = result.SessionToken
		return out, nil
	})
}
