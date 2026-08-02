// Package email defines provider-neutral transactional email delivery.
package email

import "context"

// Message is a transactional email independent of any delivery provider.
type Message struct {
	From           string
	To             string
	Subject        string
	HTML           string
	Text           string
	IdempotencyKey string
}

// Sender delivers transactional email messages.
type Sender interface {
	Send(ctx context.Context, message Message) error
}
