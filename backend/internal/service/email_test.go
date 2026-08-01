package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
)

func startSMTPTestServer(t *testing.T) (string, int, <-chan []string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	commands := make(chan []string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			commands <- nil
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		writer := bufio.NewWriter(conn)
		write := func(response string) {
			_, _ = writer.WriteString(response + "\r\n")
			_ = writer.Flush()
		}
		write("220 localhost test SMTP")
		var seen []string
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				commands <- seen
				return
			}
			line = strings.TrimSpace(line)
			seen = append(seen, line)
			upper := strings.ToUpper(line)
			switch {
			case strings.HasPrefix(upper, "EHLO"):
				write("250-localhost")
				write("250 AUTH PLAIN")
			case strings.HasPrefix(upper, "AUTH PLAIN"):
				write("235 authenticated")
			case strings.HasPrefix(upper, "MAIL FROM"):
				write("250 sender accepted")
			case strings.HasPrefix(upper, "RCPT TO"):
				write("250 recipient accepted")
			case upper == "DATA":
				write("354 send message")
				for {
					bodyLine, bodyErr := reader.ReadString('\n')
					if bodyErr != nil {
						commands <- seen
						return
					}
					if strings.TrimSpace(bodyLine) == "." {
						break
					}
				}
				write("250 queued")
			case upper == "NOOP":
				write("250 ok")
			case upper == "QUIT":
				write("221 bye")
				commands <- seen
				return
			default:
				write("500 unsupported")
			}
		}
	}()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	return host, port, commands
}

func smtpTestConfig(host string, port int) config.Config {
	return config.Config{
		SMTPHost: host, SMTPPort: port,
		SMTPUsername: "sender@example.com", SMTPPassword: "secret",
		SMTPFromEmail: "sender@example.com", SMTPFromName: "SoulCourse",
	}
}

func TestVerificationEmailMessage(t *testing.T) {
	sender := NewSMTPEmailSender(config.Config{
		SMTPFromEmail:            "no-reply@soulcourse.cn",
		SMTPReplyTo:              "contact@soulcourse.cn",
		SMTPFromName:             "SoulCourse",
		EmailVerificationSubject: "SoulCourse 邮箱验证码",
	}, nil)

	rawMessage, err := sender.message("student@example.com", "482731", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	message, err := mail.ReadMessage(bytes.NewReader(rawMessage))
	if err != nil {
		t.Fatalf("parse message: %v", err)
	}
	if got := message.Header.Get("From"); !strings.Contains(got, "no-reply@soulcourse.cn") {
		t.Fatalf("unexpected From header: %q", got)
	}
	if got := message.Header.Get("Reply-To"); got != "<contact@soulcourse.cn>" {
		t.Fatalf("unexpected Reply-To header: %q", got)
	}
	toAddress, err := mail.ParseAddress(message.Header.Get("To"))
	if err != nil {
		t.Fatalf("parse To header: %v", err)
	}
	if toAddress.Address != "student@example.com" {
		t.Fatalf("unexpected To address: %q", toAddress.Address)
	}

	mediaType, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse Content-Type: %v", err)
	}
	if mediaType != "multipart/related" {
		t.Fatalf("unexpected media type: %q", mediaType)
	}

	parts := map[string]string{}
	var logo []byte
	reader := multipart.NewReader(message.Body, params["boundary"])
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read MIME part: %v", err)
		}
		contentType, contentTypeParams, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("parse MIME part Content-Type: %v", err)
		}
		switch contentType {
		case "multipart/alternative":
			alternativeReader := multipart.NewReader(part, contentTypeParams["boundary"])
			for {
				alternativePart, err := alternativeReader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("read alternative MIME part: %v", err)
				}
				alternativeType, _, err := mime.ParseMediaType(alternativePart.Header.Get("Content-Type"))
				if err != nil {
					t.Fatalf("parse alternative MIME part Content-Type: %v", err)
				}
				decoded, err := io.ReadAll(quotedprintable.NewReader(alternativePart))
				if err != nil {
					t.Fatalf("decode alternative MIME part: %v", err)
				}
				parts[alternativeType] = string(decoded)
			}
		case "image/png":
			if got := part.Header.Get("Content-ID"); got != "<soulcourse-logo>" {
				t.Fatalf("unexpected logo Content-ID: %q", got)
			}
			logo, err = io.ReadAll(base64.NewDecoder(base64.StdEncoding, part))
			if err != nil {
				t.Fatalf("decode logo: %v", err)
			}
		}
	}

	plainBody := parts["text/plain"]
	htmlBody := parts["text/html"]
	if !strings.Contains(plainBody, "482731") {
		t.Fatal("plain body does not contain verification code")
	}
	for _, value := range []string{"4", "8", "2", "7", "3", "1", "10 分钟"} {
		if !strings.Contains(htmlBody, value) {
			t.Fatalf("HTML body does not contain %q", value)
		}
	}
	for _, value := range []string{
		"cid:soulcourse-logo",
		"选科π",
		"广东选科社区",
		"#32ade6",
		"#34c759",
		"#ffcc00",
		"#ff9500",
		"#ff3b30",
		"#5856d6",
	} {
		if !strings.Contains(htmlBody, value) {
			t.Fatalf("HTML body does not contain frontend brand value %q", value)
		}
	}
	if !strings.Contains(plainBody, "可直接回复本邮件") || !strings.Contains(htmlBody, "可直接回复本邮件") {
		t.Fatal("message does not explain the reply path")
	}
	if len(logo) < 8 || !bytes.Equal(logo[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		t.Fatal("message does not contain the embedded PNG logo")
	}
	if strings.Contains(htmlBody, "linear-gradient") || strings.Contains(htmlBody, "http://") || strings.Contains(htmlBody, "https://") {
		t.Fatal("HTML body should not depend on gradients or external assets")
	}
}

func TestVerificationEmailSanitizesHeaders(t *testing.T) {
	sender := NewSMTPEmailSender(config.Config{
		SMTPFromEmail:            "no-reply@soulcourse.cn",
		SMTPFromName:             "SoulCourse\r\nBcc: attacker@example.com",
		EmailVerificationSubject: "验证码\r\nBcc: attacker@example.com",
	}, nil)

	rawMessage, err := sender.message("student@example.com", "123456", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(rawMessage, []byte("\r\nBcc:")) {
		t.Fatal("message contains injected Bcc header")
	}
}

func TestVerificationEmailHonorsCanceledContext(t *testing.T) {
	sender := NewSMTPEmailSender(config.Config{
		SMTPHost:      "smtp.example.com",
		SMTPPort:      465,
		SMTPUsername:  "sender@example.com",
		SMTPPassword:  "secret",
		SMTPFromEmail: "sender@example.com",
		SMTPUseTLS:    true,
	}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := sender.SendVerificationCode(ctx, "student@example.com", "123456", time.Minute)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}

func TestSMTPCheckHonorsCanceledContext(t *testing.T) {
	sender := NewSMTPEmailSender(config.Config{
		SMTPHost: "smtp.example.com", SMTPPort: 465,
		SMTPUsername: "sender@example.com", SMTPPassword: "secret",
		SMTPFromEmail: "sender@example.com", SMTPUseTLS: true,
	}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sender.Check(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context, got %v", err)
	}
}

func TestSMTPDisabledBoundaries(t *testing.T) {
	sender := NewSMTPEmailSender(config.Config{}, nil)
	if sender.Enabled() {
		t.Fatal("empty SMTP config must be disabled")
	}
	if err := sender.Check(context.Background()); err == nil || err.Error() != "SMTP is not configured" {
		t.Fatalf("Check error = %v", err)
	}
	if err := sender.SendVerificationCode(context.Background(), "student@example.com", "123456", time.Minute); err != nil {
		t.Fatalf("disabled sender should be a no-op: %v", err)
	}
}

func TestVerificationEmailClampsTTLAndOmitsReplyHeader(t *testing.T) {
	sender := NewSMTPEmailSender(config.Config{SMTPFromEmail: "no-reply@example.com"}, nil)
	raw, err := sender.message("student@example.com", "123456", 0)
	if err != nil {
		t.Fatal(err)
	}
	message, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got := message.Header.Get("Reply-To"); got != "" {
		t.Fatalf("unexpected Reply-To header: %q", got)
	}
	_, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	relatedPart, err := multipart.NewReader(message.Body, params["boundary"]).NextPart()
	if err != nil {
		t.Fatal(err)
	}
	_, alternativeParams, err := mime.ParseMediaType(relatedPart.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	plainPart, err := multipart.NewReader(relatedPart, alternativeParams["boundary"]).NextPart()
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(quotedprintable.NewReader(plainPart))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte("1 分钟")) {
		t.Fatal("zero TTL should be clamped to one minute")
	}
}

func TestPlainSMTPDeliveryExecutesAuthenticatedEnvelope(t *testing.T) {
	host, port, commands := startSMTPTestServer(t)
	sender := NewSMTPEmailSender(smtpTestConfig(host, port), nil)
	if err := sender.SendVerificationCode(context.Background(), "student@example.com", "123456", 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	seen := <-commands
	joined := strings.Join(seen, "\n")
	for _, required := range []string{"EHLO", "AUTH PLAIN", "MAIL FROM:<sender@example.com>", "RCPT TO:<student@example.com>", "DATA", "QUIT"} {
		if !strings.Contains(strings.ToUpper(joined), strings.ToUpper(required)) {
			t.Fatalf("missing %q in SMTP commands:\n%s", required, joined)
		}
	}
}

func TestPlainSMTPCheckAuthenticatesAndUsesNoop(t *testing.T) {
	host, port, commands := startSMTPTestServer(t)
	sender := NewSMTPEmailSender(smtpTestConfig(host, port), nil)
	if err := sender.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(<-commands, "\n")
	for _, required := range []string{"AUTH PLAIN", "NOOP", "QUIT"} {
		if !strings.Contains(strings.ToUpper(joined), required) {
			t.Fatalf("missing %q in SMTP commands:\n%s", required, joined)
		}
	}
}

func TestSMTPProtocolFailureIsReturnedWithDeliveryContext(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			_, _ = conn.Write([]byte("invalid greeting\r\n"))
			_ = conn.Close()
		}
	}()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	sender := NewSMTPEmailSender(smtpTestConfig(host, port), nil)
	err = sender.SendVerificationCode(context.Background(), "student@example.com", "123456", time.Minute)
	if err == nil || !strings.Contains(err.Error(), "plain SMTP delivery") || !strings.Contains(err.Error(), "create client") {
		t.Fatalf("unexpected protocol error: %v", err)
	}
}
