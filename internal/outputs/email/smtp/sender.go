package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"github.com/bakkerme/curator-ai/internal/outputs/email"
	mail "github.com/wneessen/go-mail"
)

type Sender struct {
	host     string
	port     int
	username string
	password string
	useTLS   bool
}

func NewSender(host string, port int, username, password string, useTLS bool) *Sender {
	return &Sender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		useTLS:   useTLS,
	}
}

func (s *Sender) Send(ctx context.Context, message email.Message) error {
	if message.From == "" {
		message.From = s.username
	}
	if ctx == nil {
		ctx = context.Background()
	}

	m := mail.NewMsg()
	if err := m.From(message.From); err != nil {
		return fmt.Errorf("invalid from address %q: %w", message.From, err)
	}
	if err := m.ToFromString(message.To); err != nil {
		return fmt.Errorf("invalid to address(es) %q: %w", message.To, err)
	}
	m.Subject(message.Subject)
	m.SetBodyString(mail.TypeTextHTML, message.Body)
	if err := m.EnvelopeFrom(message.From); err != nil {
		return fmt.Errorf("invalid envelope from address %q: %w", message.From, err)
	}

	sendWithAuth := func(enableAuth bool) error {
		clientOpts := []mail.Option{
			mail.WithPort(s.port),
			mail.WithTLSConfig(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}),
		}
		if s.useTLS {
			// Preserve prior behavior: when useTLS is true and port is 465, use implicit TLS.
			// Otherwise, prefer STARTTLS.
			if s.port == 465 {
				clientOpts = append(clientOpts, mail.WithSSL())
			} else {
				clientOpts = append(clientOpts, mail.WithTLSPortPolicy(mail.TLSMandatory))
			}
		} else {
			clientOpts = append(clientOpts, mail.WithTLSPortPolicy(mail.NoTLS))
		}

		if enableAuth && s.username != "" {
			clientOpts = append(
				clientOpts,
				mail.WithUsername(s.username),
				mail.WithPassword(s.password),
				mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
			)
		}

		client, err := mail.NewClient(s.host, clientOpts...)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}

		if err := client.DialAndSendWithContext(ctx, m); err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
		return nil
	}

	err := sendWithAuth(s.username != "")
	if err == nil {
		return nil
	}

	// Mailpit (and similar local SMTP sinks) intentionally do not support SMTP AUTH.
	// If we were configured with credentials (often via a shared `.envrc`) but are
	// sending to a local sink, retry without auth.
	if s.username != "" && isAuthUnsupported(err) && isLocalDevSMTPHost(s.host) {
		if retryErr := sendWithAuth(false); retryErr == nil {
			return nil
		}
	}

	return err
}

func ValidateConfig(host string, port int) error {
	if host == "" {
		return fmt.Errorf("smtp host is required")
	}
	if port <= 0 {
		return fmt.Errorf("smtp port must be positive")
	}
	if _, err := net.LookupHost(host); err != nil {
		return nil
	}
	return nil
}

func isAuthUnsupported(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "server does not support SMTP AUTH") ||
		strings.Contains(msg, "SMTP Auth autodiscover was not able to detect a supported authentication mechanism")
}

func isLocalDevSMTPHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return false
	}
	if host == "localhost" || host == "mailpit" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}
