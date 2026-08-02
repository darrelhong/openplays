package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"math/big"
	"net/mail"
	"strings"
	"sync"
	"time"

	"openplays/server/internal/db"
	mailpkg "openplays/server/internal/email"
)

const (
	EmailOTPDuration        = 15 * time.Minute
	emailOTPCodeLength      = 6
	emailOTPMaxAttempts     = 5
	emailOTPResendCooldown  = time.Minute
	emailOTPEmailHourlyMax  = 5
	emailOTPGlobalHourlyMax = 100
)

var (
	ErrInvalidEmail      = errors.New("invalid email address")
	ErrEmailDomainDenied = errors.New("email domain is not allowed")
	ErrEmailOTPLimited   = errors.New("email otp request rate limited")
	ErrInvalidEmailOTP   = errors.New("email otp is invalid or expired")
)

// DomainPolicy decides whether an email domain is eligible for email login.
type DomainPolicy interface {
	Allows(domain string) bool
}

// DomainAllowlist allows exact, case-insensitive domain matches.
type DomainAllowlist map[string]struct{}

func NewDomainAllowlist(domains []string) (DomainAllowlist, error) {
	list := make(DomainAllowlist, len(domains))
	for _, domain := range domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" || strings.ContainsAny(domain, "@/ :") {
			return nil, fmt.Errorf("invalid allowed email domain %q", domain)
		}
		list[domain] = struct{}{}
	}
	if len(list) == 0 {
		return nil, errors.New("at least one allowed email domain is required")
	}
	return list, nil
}

func (d DomainAllowlist) Allows(domain string) bool {
	_, ok := d[strings.ToLower(strings.TrimSpace(domain))]
	return ok
}

// EmailOTPStore is the persistence boundary for one-time email codes.
// Only a hash of the code is stored.
type EmailOTPStore interface {
	CreateEmailOTPChallenge(context.Context, db.CreateEmailOTPChallengeParams) error
	GetEmailOTPChallenge(context.Context, string) (db.EmailOtpChallenge, error)
	IncrementEmailOTPAttempts(context.Context, string) error
	ConsumeEmailOTPChallenge(context.Context, db.ConsumeEmailOTPChallengeParams) (string, error)
	DeleteEmailOTPChallenge(context.Context, db.DeleteEmailOTPChallengeParams) error
	RestoreEmailOTPChallenge(context.Context, db.RestoreEmailOTPChallengeParams) error
	DeleteExpiredEmailOTPChallenges(context.Context) error
}

type EmailOTPConfig struct {
	From       string
	Policy     DomainPolicy
	CodeSecret string
}

// EmailOTPService sends short-lived, single-use verification codes and turns
// a valid code into a normal OpenPlays session.
type EmailOTPService struct {
	store      EmailOTPStore
	auth       *Service
	sender     mailpkg.Sender
	from       string
	policy     DomainPolicy
	limiter    *emailOTPLimiter
	now        func() time.Time
	code       func() (string, error)
	requestID  func() (string, error)
	codeSecret []byte
}

func NewEmailOTPService(store EmailOTPStore, authService *Service, sender mailpkg.Sender, cfg EmailOTPConfig) (*EmailOTPService, error) {
	if store == nil || authService == nil || sender == nil || cfg.Policy == nil {
		return nil, errors.New("email otp dependencies are required")
	}
	if strings.TrimSpace(cfg.From) == "" {
		return nil, errors.New("email otp sender address is required")
	}
	if _, err := mail.ParseAddress(cfg.From); err != nil {
		return nil, fmt.Errorf("invalid email otp sender address: %w", err)
	}
	codeSecret := []byte(strings.TrimSpace(cfg.CodeSecret))
	if len(codeSecret) < 32 {
		return nil, errors.New("email otp secret must be at least 32 characters")
	}
	now := time.Now
	return &EmailOTPService{
		store: store, auth: authService, sender: sender, from: strings.TrimSpace(cfg.From),
		policy: cfg.Policy, limiter: newEmailOTPLimiter(now), now: now,
		code: newEmailOTPCode, requestID: newEmailOTPRequestID, codeSecret: codeSecret,
	}, nil
}

// Request generates a code, stores its hash, and sends the code by email.
func (s *EmailOTPService) Request(ctx context.Context, requestedEmail string) error {
	deliveryEmail, normalizedEmail, domain, err := validateEmailAddress(requestedEmail)
	if err != nil {
		return err
	}
	if !s.policy.Allows(domain) {
		return ErrEmailDomainDenied
	}
	if !s.limiter.Allow(normalizedEmail) {
		return ErrEmailOTPLimited
	}

	code, err := s.code()
	if err != nil {
		return err
	}
	requestID, err := s.requestID()
	if err != nil {
		return err
	}
	hash := s.codeHash(normalizedEmail, code)
	_ = s.store.DeleteExpiredEmailOTPChallenges(ctx)
	previous, err := s.store.GetEmailOTPChallenge(ctx, normalizedEmail)
	hasPrevious := err == nil && s.now().Before(previous.ExpiresAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("get existing email otp: %w", err)
	}
	if err := s.store.CreateEmailOTPChallenge(ctx, db.CreateEmailOTPChallengeParams{
		Email: normalizedEmail, CodeHash: hash, ExpiresAt: s.now().Add(EmailOTPDuration),
	}); err != nil {
		return fmt.Errorf("store email otp: %w", err)
	}

	if err := s.sender.Send(ctx, emailOTPMessage(s.from, deliveryEmail, code, requestID)); err != nil {
		if hasPrevious {
			_ = s.store.RestoreEmailOTPChallenge(ctx, db.RestoreEmailOTPChallengeParams{
				Email: normalizedEmail, ExpectedCodeHash: hash, CodeHash: previous.CodeHash,
				ExpiresAt: previous.ExpiresAt, Attempts: previous.Attempts, CreatedAt: previous.CreatedAt,
			})
		} else {
			_ = s.store.DeleteEmailOTPChallenge(ctx, db.DeleteEmailOTPChallengeParams{
				Email: normalizedEmail, CodeHash: hash,
			})
		}
		return fmt.Errorf("send email otp: %w", err)
	}
	return nil
}

// Verify consumes a valid code and creates an authenticated session.
func (s *EmailOTPService) Verify(ctx context.Context, requestedEmail, code string) (*LoginResult, error) {
	_, normalizedEmail, _, err := validateEmailAddress(requestedEmail)
	if err != nil || len(code) != emailOTPCodeLength || !allDigits(code) {
		return nil, ErrInvalidEmailOTP
	}

	challenge, err := s.store.GetEmailOTPChallenge(ctx, normalizedEmail)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidEmailOTP
	}
	if err != nil {
		return nil, fmt.Errorf("get email otp: %w", err)
	}
	if challenge.Attempts >= emailOTPMaxAttempts || !s.now().Before(challenge.ExpiresAt) {
		return nil, ErrInvalidEmailOTP
	}

	hash := s.codeHash(normalizedEmail, code)
	if subtle.ConstantTimeCompare([]byte(challenge.CodeHash), []byte(hash)) != 1 {
		_ = s.store.IncrementEmailOTPAttempts(ctx, normalizedEmail)
		return nil, ErrInvalidEmailOTP
	}

	email, err := s.store.ConsumeEmailOTPChallenge(ctx, db.ConsumeEmailOTPChallengeParams{
		Email: normalizedEmail, CodeHash: hash,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidEmailOTP
	}
	if err != nil {
		return nil, fmt.Errorf("consume email otp: %w", err)
	}
	return s.auth.LoginEmail(ctx, email)
}

func newEmailOTPCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("generate email otp: %w", err)
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func newEmailOTPRequestID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate email otp request ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func allDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (s *EmailOTPService) codeHash(email, code string) string {
	mac := hmac.New(sha256.New, s.codeSecret)
	_, _ = mac.Write([]byte(email))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func emailOTPMessage(from, to, code, requestID string) mailpkg.Message {
	return mailpkg.Message{
		From: from, To: to, Subject: "Your OpenPlays sign-in code",
		Text:           "Your OpenPlays sign-in code is " + code + ".\n\nIt expires in 15 minutes. If you didn't ask for this, you can ignore this email.",
		HTML:           RenderEmailOTPHTML(code),
		IdempotencyKey: "email-otp/" + requestID,
	}
}

// RenderEmailOTPHTML returns the HTML body used for email sign-in codes.
// It is kept separate from delivery so local development tools can preview the
// exact email sent by configured providers.
func RenderEmailOTPHTML(code string) string {
	escapedCode := html.EscapeString(code)
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body style="margin:0;padding:0;background:#fafaf9;color:#1c1917;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
  <div style="display:none;max-height:0;overflow:hidden;opacity:0;">Your OpenPlays sign-in code is ` + escapedCode + `.</div>
  <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;background:#fafaf9;">
    <tr>
      <td align="center" style="padding:40px 16px;">
        <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;max-width:480px;background:#ffffff;border:1px solid #e7e5e4;border-radius:12px;">
          <tr>
            <td style="padding:32px 32px 12px;">
              <table role="presentation" cellspacing="0" cellpadding="0" border="0">
                <tr>
                  <td style="width:10px;height:10px;background:rgb(225,82,65);border-radius:999px;font-size:0;line-height:0;">&nbsp;</td>
                  <td style="padding-left:8px;color:#1c1917;font-size:18px;font-weight:700;letter-spacing:-0.02em;line-height:24px;">OpenPlays</td>
                </tr>
              </table>
            </td>
          </tr>
          <tr>
            <td style="padding:16px 32px 8px;color:#1c1917;font-size:24px;font-weight:700;letter-spacing:-0.02em;line-height:32px;">Sign in to OpenPlays</td>
          </tr>
          <tr>
            <td style="padding:0 32px 24px;color:#57534e;font-size:16px;line-height:24px;">Enter this code to finish signing in.</td>
          </tr>
          <tr>
            <td style="padding:0 32px;">
              <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;background:#f5f5f4;border:1px solid #e7e5e4;border-radius:8px;">
                <tr>
                  <td align="center" style="padding:20px 16px;color:#1c1917;font-size:30px;font-weight:700;letter-spacing:0.24em;line-height:36px;">` + escapedCode + `</td>
                </tr>
              </table>
            </td>
          </tr>
          <tr>
            <td style="padding:24px 32px 32px;color:#78716c;font-size:14px;line-height:20px;">This code expires in 15 minutes. If you didn’t ask to sign in, you can safely ignore this email.</td>
          </tr>
        </table>
        <p style="margin:20px 0 0;color:#a8a29e;font-size:12px;line-height:18px;">OpenPlays</p>
      </td>
    </tr>
  </table>
</body>
</html>`
}

func validateEmailAddress(input string) (deliveryEmail, normalizedEmail, domain string, err error) {
	input = strings.TrimSpace(input)
	address, parseErr := mail.ParseAddress(input)
	if parseErr != nil || !strings.EqualFold(address.Address, input) || len(input) > 254 {
		return "", "", "", ErrInvalidEmail
	}
	at := strings.LastIndex(address.Address, "@")
	if at <= 0 || at == len(address.Address)-1 {
		return "", "", "", ErrInvalidEmail
	}
	domain = strings.ToLower(address.Address[at+1:])
	return address.Address, normalizeEmail(address.Address), domain, nil
}

type emailOTPLimiter struct {
	mu      sync.Mutex
	byEmail map[string][]time.Time
	global  []time.Time
	now     func() time.Time
}

func newEmailOTPLimiter(now func() time.Time) *emailOTPLimiter {
	return &emailOTPLimiter{byEmail: make(map[string][]time.Time), now: now}
}

func (l *emailOTPLimiter) Allow(email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	windowStart := now.Add(-time.Hour)
	l.global = pruneAttempts(l.global, windowStart)
	for key, attempts := range l.byEmail {
		attempts = pruneAttempts(attempts, windowStart)
		if len(attempts) == 0 {
			delete(l.byEmail, key)
		} else {
			l.byEmail[key] = attempts
		}
	}
	emailAttempts := l.byEmail[email]
	if len(l.global) >= emailOTPGlobalHourlyMax || len(emailAttempts) >= emailOTPEmailHourlyMax {
		return false
	}
	if len(emailAttempts) > 0 && now.Sub(emailAttempts[len(emailAttempts)-1]) < emailOTPResendCooldown {
		return false
	}
	l.global = append(l.global, now)
	l.byEmail[email] = append(emailAttempts, now)
	return true
}

func pruneAttempts(attempts []time.Time, cutoff time.Time) []time.Time {
	first := 0
	for first < len(attempts) && attempts[first].Before(cutoff) {
		first++
	}
	return attempts[first:]
}
