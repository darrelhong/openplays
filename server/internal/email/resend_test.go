package email

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestResendSenderSend(t *testing.T) {
	var got struct {
		From string   `json:"from"`
		To   []string `json:"to"`
		Text string   `json:"text"`
	}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization header = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("User-Agent is empty")
		}
		if r.Header.Get("Idempotency-Key") != "email-otp/abc" {
			t.Errorf("idempotency key = %q", r.Header.Get("Idempotency-Key"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"email-1"}`))}, nil
	})}

	sender, err := NewResendSender("secret", client)
	if err != nil {
		t.Fatal(err)
	}
	err = sender.Send(context.Background(), Message{
		From: "OpenPlays <login@example.com>", To: "user@gmail.com", Subject: "Sign in",
		Text: "hello", IdempotencyKey: "email-otp/abc",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got.From != "OpenPlays <login@example.com>" || len(got.To) != 1 || got.To[0] != "user@gmail.com" || got.Text != "hello" {
		t.Fatalf("payload = %#v", got)
	}
}

func TestResendSenderReturnsProviderError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Body:       io.NopCloser(strings.NewReader(`{"message":"bad sender"}`)),
		}, nil
	})}
	sender, _ := NewResendSender("secret", client)
	err := sender.Send(context.Background(), Message{To: "user@gmail.com"})
	if err == nil || !strings.Contains(err.Error(), "HTTP 422") {
		t.Fatalf("error = %v", err)
	}
}
