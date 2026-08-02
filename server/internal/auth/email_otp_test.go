package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"openplays/server/internal/db"
	mailpkg "openplays/server/internal/email"
	"openplays/server/internal/testdb"
)

const testEmailOTPSecret = "test-email-otp-secret-that-is-at-least-32-characters"

type recordingSender struct {
	messages []mailpkg.Message
	err      error
}

func (s *recordingSender) Send(_ context.Context, message mailpkg.Message) error {
	s.messages = append(s.messages, message)
	return s.err
}

func newTestEmailOTPService(t *testing.T, sqlDB *sql.DB, sender *recordingSender) *EmailOTPService {
	return newTestEmailOTPServiceWithSecret(t, sqlDB, sender, testEmailOTPSecret)
}

func newTestEmailOTPServiceWithSecret(t *testing.T, sqlDB *sql.DB, sender *recordingSender, secret string) *EmailOTPService {
	t.Helper()
	policy, err := NewDomainAllowlist([]string{"gmail.com", "outlook.com"})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewEmailOTPService(db.New(sqlDB), NewService(db.New(sqlDB)), sender, EmailOTPConfig{
		From: "OpenPlays <login@openplays.app>", Policy: policy, CodeSecret: secret,
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestNewEmailOTPServiceRequiresLongSecret(t *testing.T) {
	sqlDB := testdb.New(t)
	policy, err := NewDomainAllowlist([]string{"gmail.com"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewEmailOTPService(db.New(sqlDB), NewService(db.New(sqlDB)), &recordingSender{}, EmailOTPConfig{
		From: "OpenPlays <login@openplays.app>", Policy: policy, CodeSecret: "too-short",
	})
	if err == nil || !strings.Contains(err.Error(), "at least 32") {
		t.Fatalf("error = %v, want short-secret validation error", err)
	}
}

func codeFromMessage(t *testing.T, message mailpkg.Message) string {
	t.Helper()
	code := regexp.MustCompile(`\b[0-9]{6}\b`).FindString(message.Text)
	if code == "" {
		t.Fatalf("message does not contain a six-digit code: %q", message.Text)
	}
	return code
}

func TestRenderEmailOTPHTMLUsesOpenPlaysEmailStyle(t *testing.T) {
	body := RenderEmailOTPHTML("<123456>")
	for _, want := range []string{
		"OpenPlays", "-apple-system", "background:#fafaf9", "background:rgb(225,82,65)", "&lt;123456&gt;",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("email HTML does not contain %q", want)
		}
	}
	if strings.Contains(body, "<123456>") {
		t.Fatalf("email HTML contains unescaped code: %q", body)
	}
}

func TestEmailOTPRequestAndVerify(t *testing.T) {
	sqlDB := testdb.New(t)
	sender := &recordingSender{}
	svc := newTestEmailOTPService(t, sqlDB, sender)

	if err := svc.Request(context.Background(), "Player@GMAIL.com"); err != nil {
		t.Fatalf("Request: %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(sender.messages))
	}
	code := codeFromMessage(t, sender.messages[0])
	if len(code) != 6 || !allDigits(code) {
		t.Fatalf("code = %q", code)
	}
	if strings.Contains(sender.messages[0].IdempotencyKey, code) {
		t.Fatal("idempotency key contains raw OTP")
	}
	var storedHash string
	if err := sqlDB.QueryRow("SELECT code_hash FROM email_otp_challenges WHERE email = ?", "player@gmail.com").Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if storedHash == code || storedHash == "" {
		t.Fatalf("stored code hash = %q", storedHash)
	}
	rawHash := sha256.Sum256([]byte(code))
	if storedHash == hex.EncodeToString(rawHash[:]) {
		t.Fatal("stored verifier is an unkeyed code hash")
	}

	wrongSecretSvc := newTestEmailOTPServiceWithSecret(t, sqlDB, sender, "different-email-otp-secret-that-is-at-least-32-chars")
	if _, err := wrongSecretSvc.Verify(context.Background(), "player@gmail.com", code); !errors.Is(err, ErrInvalidEmailOTP) {
		t.Fatalf("wrong-secret Verify error = %v, want ErrInvalidEmailOTP", err)
	}

	result, err := svc.Verify(context.Background(), "player@gmail.com", code)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.User.Email != "player@gmail.com" || result.SessionToken == "" {
		t.Fatalf("login result = %#v", result)
	}
	if _, err := svc.Verify(context.Background(), "player@gmail.com", code); !errors.Is(err, ErrInvalidEmailOTP) {
		t.Fatalf("second Verify error = %v, want ErrInvalidEmailOTP", err)
	}
}

func TestEmailOTPRequestUsesUniqueIdempotencyKeysForTheSameCode(t *testing.T) {
	sqlDB := testdb.New(t)
	sender := &recordingSender{}
	svc := newTestEmailOTPService(t, sqlDB, sender)
	svc.code = func() (string, error) { return "123456", nil }
	now := time.Now()
	svc.now = func() time.Time { return now }
	svc.limiter.now = func() time.Time { return now }

	if err := svc.Request(context.Background(), "player@gmail.com"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(emailOTPResendCooldown)
	if err := svc.Request(context.Background(), "player@gmail.com"); err != nil {
		t.Fatal(err)
	}
	if len(sender.messages) != 2 {
		t.Fatalf("sent messages = %d, want 2", len(sender.messages))
	}
	if sender.messages[0].IdempotencyKey == sender.messages[1].IdempotencyKey {
		t.Fatalf("idempotency keys match for separate sends: %q", sender.messages[0].IdempotencyKey)
	}
}

func TestEmailOTPFailedResendPreservesPreviousCode(t *testing.T) {
	sqlDB := testdb.New(t)
	sender := &recordingSender{}
	svc := newTestEmailOTPService(t, sqlDB, sender)
	now := time.Now()
	svc.now = func() time.Time { return now }
	svc.limiter.now = func() time.Time { return now }

	if err := svc.Request(context.Background(), "player@gmail.com"); err != nil {
		t.Fatal(err)
	}
	firstCode := codeFromMessage(t, sender.messages[0])
	now = now.Add(emailOTPResendCooldown)
	sender.err = errors.New("provider unavailable")
	if err := svc.Request(context.Background(), "player@gmail.com"); err == nil {
		t.Fatal("expected resend failure")
	}
	sender.err = nil

	if _, err := svc.Verify(context.Background(), "player@gmail.com", firstCode); err != nil {
		t.Fatalf("Verify original code after resend failure: %v", err)
	}
}

func TestEmailOTPWrongCodeHasAttemptCeiling(t *testing.T) {
	sqlDB := testdb.New(t)
	sender := &recordingSender{}
	svc := newTestEmailOTPService(t, sqlDB, sender)
	if err := svc.Request(context.Background(), "player@gmail.com"); err != nil {
		t.Fatal(err)
	}
	wrongCode := "000000"
	if wrongCode == codeFromMessage(t, sender.messages[0]) {
		wrongCode = "111111"
	}
	for i := 0; i < emailOTPMaxAttempts; i++ {
		if _, err := svc.Verify(context.Background(), "player@gmail.com", wrongCode); !errors.Is(err, ErrInvalidEmailOTP) {
			t.Fatalf("attempt %d error = %v", i+1, err)
		}
	}
	var attempts int64
	if err := sqlDB.QueryRow("SELECT attempts FROM email_otp_challenges WHERE email = ?", "player@gmail.com").Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != emailOTPMaxAttempts {
		t.Fatalf("attempts = %d, want %d", attempts, emailOTPMaxAttempts)
	}
}

func TestEmailOTPExpiredCodeIsRejected(t *testing.T) {
	sqlDB := testdb.New(t)
	sender := &recordingSender{}
	svc := newTestEmailOTPService(t, sqlDB, sender)
	now := time.Now()
	svc.now = func() time.Time { return now }
	svc.limiter.now = func() time.Time { return now }
	if err := svc.Request(context.Background(), "player@gmail.com"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(EmailOTPDuration + time.Second)
	if _, err := svc.Verify(context.Background(), "player@gmail.com", codeFromMessage(t, sender.messages[0])); !errors.Is(err, ErrInvalidEmailOTP) {
		t.Fatalf("expired Verify error = %v, want ErrInvalidEmailOTP", err)
	}
}

func TestEmailOTPRequestRejectsUnallowedDomain(t *testing.T) {
	sender := &recordingSender{}
	svc := newTestEmailOTPService(t, testdb.New(t), sender)
	if err := svc.Request(context.Background(), "player@example.com"); !errors.Is(err, ErrEmailDomainDenied) {
		t.Fatalf("error = %v, want ErrEmailDomainDenied", err)
	}
	if len(sender.messages) != 0 {
		t.Fatal("email sent to denied domain")
	}
}
