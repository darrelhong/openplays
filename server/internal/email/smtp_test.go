package email

import (
	"context"
	"strings"
	"testing"
)

func TestNewSMTPSenderDefaultsToMailpitPort(t *testing.T) {
	sender, err := NewSMTPSender("127.0.0.1", "")
	if err != nil {
		t.Fatal(err)
	}
	if sender.address != "127.0.0.1:1025" {
		t.Fatalf("address = %q, want 127.0.0.1:1025", sender.address)
	}
}

func TestSMTPSenderRejectsInvalidConfiguration(t *testing.T) {
	if _, err := NewSMTPSender("", "1025"); err == nil {
		t.Fatal("empty host was accepted")
	}
	if _, err := NewSMTPSender("localhost", "not-a-port"); err == nil {
		t.Fatal("invalid port was accepted")
	}
}

func TestBuildSMTPMessageContainsTextAndHTMLParts(t *testing.T) {
	body, err := buildSMTPMessage(Message{
		From: "OpenPlays <login@example.com>", To: "player@gmail.com", Subject: "Sign in",
		Text: "plain login link", HTML: "<p>html login link</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	content := string(body)
	for _, expected := range []string{
		"From: OpenPlays <login@example.com>",
		"To: player@gmail.com",
		"Subject: Sign in",
		"plain login link",
		"<p>html login link</p>",
	} {
		if !strings.Contains(content, expected) {
			t.Errorf("message does not contain %q", expected)
		}
	}
}

func TestSMTPSenderRejectsHeaderInjectionBeforeDial(t *testing.T) {
	sender, err := NewSMTPSender("127.0.0.1", "1")
	if err != nil {
		t.Fatal(err)
	}
	err = sender.Send(context.Background(), Message{From: "login@example.com\r\nBcc: bad@example.com", To: "player@gmail.com"})
	if err == nil || !strings.Contains(err.Error(), "invalid SMTP from") {
		t.Fatalf("error = %v, want header validation error", err)
	}
}
