package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/config"
)

type EmailSender interface {
	Enabled() bool
	SendVerificationCode(ctx context.Context, to string, code string, ttl time.Duration) error
}

type SMTPEmailSender struct {
	cfg config.Config
}

func NewSMTPEmailSender(cfg config.Config) *SMTPEmailSender {
	return &SMTPEmailSender{cfg: cfg}
}

func (s *SMTPEmailSender) Enabled() bool {
	return s.cfg.SMTPEnabled()
}

func (s *SMTPEmailSender) SendVerificationCode(ctx context.Context, to string, code string, ttl time.Duration) error {
	if !s.Enabled() {
		return nil
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.send(to, code, ttl)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (s *SMTPEmailSender) send(to string, code string, ttl time.Duration) error {
	addr := net.JoinHostPort(s.cfg.SMTPHost, fmt.Sprintf("%d", s.cfg.SMTPPort))
	auth := smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	message := s.message(to, code, ttl)

	if s.cfg.SMTPUseTLS {
		return s.sendWithTLS(addr, auth, to, message)
	}
	if s.cfg.SMTPStartTLS {
		return s.sendWithStartTLS(addr, auth, to, message)
	}
	return smtp.SendMail(addr, auth, s.cfg.SMTPFromEmail, []string{to}, message)
}

func (s *SMTPEmailSender) sendWithTLS(addr string, auth smtp.Auth, to string, message []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.cfg.SMTPHost, MinVersion: tls.VersionTLS12})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.SMTPHost)
	if err != nil {
		return err
	}
	defer client.Quit()

	return s.sendWithClient(client, auth, to, message)
}

func (s *SMTPEmailSender) sendWithStartTLS(addr string, auth smtp.Auth, to string, message []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Quit()

	if err := client.StartTLS(&tls.Config{ServerName: s.cfg.SMTPHost, MinVersion: tls.VersionTLS12}); err != nil {
		return err
	}
	return s.sendWithClient(client, auth, to, message)
}

func (s *SMTPEmailSender) sendWithClient(client *smtp.Client, auth smtp.Auth, to string, message []byte) error {
	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(s.cfg.SMTPFromEmail); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}

func (s *SMTPEmailSender) message(to string, code string, ttl time.Duration) []byte {
	from := mail.Address{Name: s.cfg.SMTPFromName, Address: s.cfg.SMTPFromEmail}
	toAddress := mail.Address{Address: to}
	subject := mime.QEncoding.Encode("UTF-8", s.cfg.EmailVerificationSubject)
	body := fmt.Sprintf(`你的邮箱验证码是：%s

验证码 %d 分钟内有效。若不是你本人操作，请忽略这封邮件。

选科知谈`, code, int(ttl.Minutes()))

	var buffer bytes.Buffer
	headers := map[string]string{
		"From":         from.String(),
		"To":           toAddress.String(),
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": `text/plain; charset="UTF-8"`,
	}
	for key, value := range headers {
		buffer.WriteString(key)
		buffer.WriteString(": ")
		buffer.WriteString(strings.ReplaceAll(value, "\n", ""))
		buffer.WriteString("\r\n")
	}
	buffer.WriteString("\r\n")
	buffer.WriteString(body)
	buffer.WriteString("\r\n")
	return buffer.Bytes()
}
