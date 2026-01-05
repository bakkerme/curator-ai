package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/bakkerme/curator-ai/internal/outputs/email"
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
	_ = ctx
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	msg := buildMessage(message)

	if s.useTLS {
		return s.sendTLS(addr, auth, msg, message)
	}
	return smtp.SendMail(addr, auth, message.From, []string{message.To}, []byte(msg))
}

func (s *Sender) sendTLS(addr string, auth smtp.Auth, msg string, message email.Message) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: s.host,
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return err
	}
	defer client.Quit()

	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(message.From); err != nil {
		return err
	}
	if err := client.Rcpt(message.To); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(msg))
	if err != nil {
		return err
	}
	return writer.Close()
}

func buildMessage(message email.Message) string {
	headers := []string{
		fmt.Sprintf("From: %s", message.From),
		fmt.Sprintf("To: %s", message.To),
		fmt.Sprintf("Subject: %s", message.Subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=\"UTF-8\"",
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + message.Body
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
