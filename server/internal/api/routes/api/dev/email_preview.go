package dev

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/auth"
)

const emailPreviewCode = "482731"

type EmailPreviewOutput struct {
	Body struct {
		HTML string `json:"html" doc:"Rendered HTML for the email OTP preview"`
	}
}

// RegisterEmailPreview exposes the exact OTP email HTML for local visual checks.
// It is only reachable when the enclosing dev router is enabled.
func RegisterEmailPreview(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "dev-email-otp-preview",
		Summary:     "Preview email sign-in code",
		Description: "Render the email OTP HTML with a fixed sample code. Only registered when DEV_AUTH_ENABLED=true.",
		Method:      http.MethodGet,
		Path:        "/email-otp-preview",
		Tags:        []string{"Dev"},
	}, func(context.Context, *struct{}) (*EmailPreviewOutput, error) {
		out := &EmailPreviewOutput{}
		out.Body.HTML = auth.RenderEmailOTPHTML(emailPreviewCode)
		return out, nil
	})
}
