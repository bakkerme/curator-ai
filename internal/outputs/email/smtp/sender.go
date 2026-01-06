package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

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

	if s.username != "" {
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
