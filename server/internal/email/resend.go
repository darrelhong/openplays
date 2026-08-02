package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const resendAPIURL = "https://api.resend.com/emails"

// ResendSender delivers email through Resend's HTTP API.
type ResendSender struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

// NewResendSender creates a Resend-backed Sender.
func NewResendSender(apiKey string, client *http.Client) (*ResendSender, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("resend API key is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &ResendSender{apiKey: apiKey, endpoint: resendAPIURL, client: client}, nil
}

func (s *ResendSender) Send(ctx context.Context, message Message) error {
	payload := struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		HTML    string   `json:"html,omitempty"`
		Text    string   `json:"text,omitempty"`
	}{
		From: message.From, To: []string{message.To}, Subject: message.Subject,
		HTML: message.HTML, Text: message.Text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode resend request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "openplays-server/1.0")
	if message.IdempotencyKey != "" {
		req.Header.Set("Idempotency-Key", message.IdempotencyKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send email with resend: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("resend returned HTTP %d", resp.StatusCode)
	}
	return nil
}
