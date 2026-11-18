package mail

import (
	"context"
	"fmt"
	"mime"
	"net/smtp"
	"strings"
)

type Message struct {
	To      string
	Subject string
	Body    string
}

type Sender interface {
	Send(ctx context.Context, message Message) error
}

type SMTPSender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPSender(
	host string,
	port string,
	username string,
	password string,
	from string,
) *SMTPSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPSender) Send(
	ctx context.Context,
	message Message,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	address := fmt.Sprintf("%s:%s", s.host, s.port)

	var auth smtp.Auth

	if s.username != "" {
		auth = smtp.PlainAuth(
			"",
			s.username,
			s.password,
			s.host,
		)
	}

	subject := mime.QEncoding.Encode("UTF-8", message.Subject)

	body := strings.ReplaceAll(message.Body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\n", "\r\n")

	email := strings.Join([]string{
		"From: " + s.from,
		"To: " + message.To,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")

	err := smtp.SendMail(
		address,
		auth,
		s.from,
		[]string{message.To},
		[]byte(email),
	)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}
