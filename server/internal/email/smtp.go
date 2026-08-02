package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

// SMTPSender delivers mail over SMTP. It is intended for local development
// with a catcher such as Mailpit; production should use a real provider
// adapter such as Resend.
type SMTPSender struct {
	address string
	host    string
	timeout time.Duration
}

// NewSMTPSender creates an unauthenticated SMTP sender. An empty port uses
// port 1025, Mailpit's default SMTP port.
func NewSMTPSender(host, port string) (*SMTPSender, error) {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	if host == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}
	if strings.ContainsAny(host, " \t\r\n") {
		return nil, fmt.Errorf("invalid SMTP host")
	}
	if port == "" {
		port = "1025"
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return nil, fmt.Errorf("invalid SMTP port %q", port)
	}
	return &SMTPSender{
		address: net.JoinHostPort(host, port),
		host:    host,
		timeout: 10 * time.Second,
	}, nil
}

func (s *SMTPSender) Send(_ context.Context, message Message) error {
	from, err := mail.ParseAddress(message.From)
	if err != nil || strings.ContainsAny(message.From, "\r\n") {
		return fmt.Errorf("invalid SMTP from address")
	}
	to, err := mail.ParseAddress(message.To)
	if err != nil || strings.ContainsAny(message.To, "\r\n") {
		return fmt.Errorf("invalid SMTP recipient")
	}
	if strings.ContainsAny(message.Subject, "\r\n") {
		return fmt.Errorf("invalid SMTP subject")
	}
	body, err := buildSMTPMessage(message)
	if err != nil {
		return err
	}

	conn, err := net.DialTimeout("tcp", s.address, s.timeout)
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(s.timeout))
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("initialize SMTP client: %w", err)
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	}
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(to.Address); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP message: %w", err)
	}
	if _, err := writer.Write(body); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("close SMTP connection: %w", err)
	}
	return nil
}

func buildSMTPMessage(message Message) ([]byte, error) {
	if strings.ContainsAny(message.From+message.To+message.Subject, "\r\n") {
		return nil, fmt.Errorf("SMTP headers cannot contain newlines")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	boundary := writer.Boundary()
	fmt.Fprintf(&body, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", message.From, message.To, message.Subject, boundary)

	textHeaders := make(textproto.MIMEHeader)
	textHeaders.Set("Content-Type", "text/plain; charset=UTF-8")
	textPart, err := writer.CreatePart(textHeaders)
	if err != nil {
		return nil, fmt.Errorf("create SMTP text part: %w", err)
	}
	if _, err := io.WriteString(textPart, message.Text); err != nil {
		return nil, fmt.Errorf("write SMTP text part: %w", err)
	}

	htmlHeaders := make(textproto.MIMEHeader)
	htmlHeaders.Set("Content-Type", "text/html; charset=UTF-8")
	htmlPart, err := writer.CreatePart(htmlHeaders)
	if err != nil {
		return nil, fmt.Errorf("create SMTP HTML part: %w", err)
	}
	if _, err := io.WriteString(htmlPart, message.HTML); err != nil {
		return nil, fmt.Errorf("write SMTP HTML part: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finish SMTP multipart message: %w", err)
	}
	return body.Bytes(), nil
}
