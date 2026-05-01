package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	authpkg "openplays/server/internal/auth"
)

type FacebookInput struct {
	Body struct {
		Code        string `json:"code" required:"true" doc:"OAuth authorization code from Facebook callback"`
		RedirectURI string `json:"redirect_uri" required:"true" doc:"The redirect_uri used in the OAuth flow (must match)"`
	}
}

type FacebookOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      struct {
		User         authpkg.User `json:"user"`
		SessionToken string       `json:"session_token"`
	}
}

// RegisterFacebook registers POST /auth/facebook.
// Exchanges Facebook OAuth code for user info, then logs in.
func RegisterFacebook(api huma.API, svc *authpkg.Service, verifier *authpkg.FacebookVerifier, cookieCfg CookieConfig) {
	huma.Register(api, huma.Operation{
		OperationID: "auth-facebook",
		Summary:     "Authenticate with Facebook",
		Description: "Exchange a Facebook OAuth code for a session.",
		Method:      http.MethodPost,
		Path:        "/facebook",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *FacebookInput) (*FacebookOutput, error) {
		// Override redirect_uri for this exchange (SvelteKit passes its own callback URL)
		claims, err := verifier.Exchange(input.Body.Code, input.Body.RedirectURI)
		if err != nil {
			slog.Warn("auth: facebook verification failed", "error", err)
			return nil, huma.Error401Unauthorized("invalid Facebook credentials")
		}

		identity := authpkg.Identity{
			Provider:    authpkg.ProviderFacebook,
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
			slog.Error("auth: facebook login failed", "error", err)
			return nil, huma.Error500InternalServerError("login failed")
		}

		out := &FacebookOutput{}
		out.SetCookie = sessionCookie(result.SessionToken, cookieCfg)
		out.Body.User = result.User
		out.Body.SessionToken = result.SessionToken
		return out, nil
	})
}
