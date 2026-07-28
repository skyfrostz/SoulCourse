package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
)

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
