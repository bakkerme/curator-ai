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
	host               string
	port               int
	username           string
	password           string
	tlsMode            string
	insecureSkipVerify bool
}

// NewSender creates an SMTP sender with explicit TLS mode support.
// The tlsMode value is optional; if empty, port-based defaults apply.
func NewSender(host string, port int, username, password string, tlsMode string, insecureSkipVerify bool) *Sender {
	return &Sender{
		host:               host,
		port:               port,
		username:           username,
		password:           password,
		tlsMode:            tlsMode,
		insecureSkipVerify: insecureSkipVerify,
	}
}

// TLSMode determines how the SMTP client should negotiate TLS.
type TLSMode string

const (
	// TLSModeAuto uses port-based defaults (implicit TLS on 465, STARTTLS otherwise).
	TLSModeAuto TLSMode = "auto"
	// TLSModeDisabled forces cleartext SMTP.
	TLSModeDisabled TLSMode = "disabled"
	// TLSModeStartTLS requires STARTTLS on the SMTP connection.
	TLSModeStartTLS TLSMode = "starttls"
	// TLSModeImplicit uses implicit TLS (SMTPS), typically on port 465.
	TLSModeImplicit TLSMode = "implicit"
)

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
		mode, modeErr := s.resolveTLSMode()
		if modeErr != nil {
			return modeErr
		}

		clientOpts := []mail.Option{
			mail.WithPort(s.port),
			// Allow self-signed or otherwise invalid TLS certs when explicitly configured.
			mail.WithTLSConfig(&tls.Config{
				ServerName:         s.host,
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: s.insecureSkipVerify,
			}),
		}

		switch mode {
		case TLSModeDisabled:
			clientOpts = append(clientOpts, mail.WithTLSPortPolicy(mail.NoTLS))
		case TLSModeStartTLS:
			clientOpts = append(clientOpts, mail.WithTLSPortPolicy(mail.TLSMandatory))
		case TLSModeImplicit:
			clientOpts = append(clientOpts, mail.WithSSL())
		default:
			return fmt.Errorf("unsupported smtp tls mode %q", mode)
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

// resolveTLSMode returns the configured TLS behavior, falling back to port defaults.
func (s *Sender) resolveTLSMode() (TLSMode, error) {
	mode, err := parseTLSMode(s.tlsMode)
	if err != nil {
		return "", err
	}
	if mode == TLSModeAuto {
		if s.port == 465 {
			return TLSModeImplicit, nil
		}
		return TLSModeStartTLS, nil
	}
	return mode, nil
}

// parseTLSMode normalizes the TLS mode string and validates supported values.
func parseTLSMode(mode string) (TLSMode, error) {
	normalized := strings.TrimSpace(strings.ToLower(mode))
	if normalized == "" || normalized == string(TLSModeAuto) {
		return TLSModeAuto, nil
	}
	switch normalized {
	case "disabled", "off", "none":
		return TLSModeDisabled, nil
	case "starttls", "start_tls":
		return TLSModeStartTLS, nil
	case "implicit", "smtptls", "smtp_tls":
		return TLSModeImplicit, nil
	default:
		return "", fmt.Errorf("invalid smtp tls mode %q (expected: auto, disabled/off/none, starttls/start_tls, implicit/smtptls/smtp_tls)", mode)
	}
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
