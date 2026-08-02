package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	authpkg "openplays/server/internal/auth"
)

const emailOTPAcceptedMessage = "If the address is eligible, we sent a sign-in code."

type EmailOTPRequestInput struct {
	Body struct {
		Email string `json:"email" required:"true" maxLength:"254" doc:"Email address to receive the sign-in code"`
	}
}

type EmailOTPRequestOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type EmailOTPVerifyInput struct {
	Body struct {
		Email string `json:"email" required:"true" maxLength:"254" doc:"Email address used to request the sign-in code"`
		Code  string `json:"code" required:"true" minLength:"6" maxLength:"6" pattern:"^[0-9]{6}$" doc:"Six-digit sign-in code"`
	}
}

type EmailOTPVerifyOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
	Body      struct {
		User         authpkg.User `json:"user"`
		SessionToken string       `json:"session_token"`
	}
}

func RegisterEmailOTP(api huma.API, svc *authpkg.EmailOTPService, cookieCfg CookieConfig) {
	huma.Register(api, huma.Operation{
		OperationID:   "auth-email-otp-request",
		Summary:       "Request an email sign-in code",
		Description:   "Send a short-lived, single-use sign-in code to an eligible email address.",
		Method:        http.MethodPost,
		Path:          "/email-otp/request",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, input *EmailOTPRequestInput) (*EmailOTPRequestOutput, error) {
		if svc == nil {
			return nil, huma.Error503ServiceUnavailable("email login is unavailable")
		}
		err := svc.Request(ctx, input.Body.Email)
		switch {
		case err == nil, errors.Is(err, authpkg.ErrEmailOTPLimited):
			out := &EmailOTPRequestOutput{}
			out.Body.Message = emailOTPAcceptedMessage
			return out, nil
		case errors.Is(err, authpkg.ErrInvalidEmail):
			return nil, huma.Error422UnprocessableEntity("enter a valid email address")
		case errors.Is(err, authpkg.ErrEmailDomainDenied):
			return nil, huma.Error422UnprocessableEntity("email provider is not supported")
		default:
			slog.Error("auth: email otp request failed", "error", err)
			return nil, huma.Error503ServiceUnavailable("email delivery is temporarily unavailable")
		}
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-email-otp-verify",
		Summary:     "Verify an email sign-in code",
		Description: "Exchange a valid, short-lived email sign-in code for a session.",
		Method:      http.MethodPost,
		Path:        "/email-otp/verify",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *EmailOTPVerifyInput) (*EmailOTPVerifyOutput, error) {
		if svc == nil {
			return nil, huma.Error503ServiceUnavailable("email login is unavailable")
		}
		result, err := svc.Verify(ctx, input.Body.Email, input.Body.Code)
		if err != nil {
			if errors.Is(err, authpkg.ErrInvalidEmailOTP) {
				return nil, huma.Error401Unauthorized("that code is invalid or expired")
			}
			if errors.Is(err, authpkg.ErrAccountBanned) {
				return nil, huma.Error403Forbidden(err.Error())
			}
			slog.Error("auth: email otp login failed", "error", err)
			return nil, huma.Error500InternalServerError("login failed")
		}

		out := &EmailOTPVerifyOutput{}
		out.SetCookie = sessionCookie(result.SessionToken, cookieCfg)
		out.Body.User = result.User
		out.Body.SessionToken = result.SessionToken
		return out, nil
	})
}
