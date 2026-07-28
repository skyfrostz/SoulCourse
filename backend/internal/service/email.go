package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"html/template"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/brandassets"
	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/logx"
)

type EmailSender interface {
	Enabled() bool
	SendVerificationCode(ctx context.Context, to string, code string, ttl time.Duration) error
}

type SMTPEmailSender struct {
	cfg    config.Config
	logger *logx.Logger
}

type verificationEmailTemplateData struct {
	BrandName        string
	Digits           []string
	ExpiresInMinutes int
}

var verificationEmailHTMLTemplate = template.Must(template.New("verification-email").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>邮箱验证码</title>
</head>
<body style="margin:0;padding:0;background:#f5f5f7;color:#0f172a;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','PingFang SC','Microsoft YaHei',Arial,sans-serif;">
  <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;background:#f5f5f7;">
    <tr>
      <td align="center" style="padding:40px 16px;">
        <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;max-width:620px;background:#ffffff;border:1px solid #e2e8f0;border-radius:12px;overflow:hidden;">
          <tr>
            <td style="padding:24px 32px;border-bottom:1px solid #e2e8f0;">
              <table role="presentation" cellspacing="0" cellpadding="0" border="0">
                <tr>
                  <td style="width:52px;vertical-align:middle;">
                    <img src="cid:soulcourse-logo" width="52" height="52" alt="SoulCourse" style="display:block;width:52px;height:52px;border:0;object-fit:contain;">
                  </td>
                  <td style="padding-left:14px;vertical-align:middle;">
                    <div style="font-size:22px;font-weight:800;line-height:1.1;color:#0f172a;">选科π</div>
                    <div style="margin-top:4px;font-size:11px;font-weight:700;line-height:1.2;color:#86868b;">广东选科社区</div>
                    <table role="presentation" cellspacing="0" cellpadding="0" border="0" style="margin-top:6px;width:84px;">
                      <tr>
                        <td style="height:3px;background:#32ade6;font-size:0;line-height:0;">&nbsp;</td>
                        <td style="width:3px;font-size:0;line-height:0;">&nbsp;</td>
                        <td style="height:3px;background:#34c759;font-size:0;line-height:0;">&nbsp;</td>
                        <td style="width:3px;font-size:0;line-height:0;">&nbsp;</td>
                        <td style="height:3px;background:#ffcc00;font-size:0;line-height:0;">&nbsp;</td>
                        <td style="width:3px;font-size:0;line-height:0;">&nbsp;</td>
                        <td style="height:3px;background:#ff9500;font-size:0;line-height:0;">&nbsp;</td>
                        <td style="width:3px;font-size:0;line-height:0;">&nbsp;</td>
                        <td style="height:3px;background:#ff3b30;font-size:0;line-height:0;">&nbsp;</td>
                        <td style="width:3px;font-size:0;line-height:0;">&nbsp;</td>
                        <td style="height:3px;background:#5856d6;font-size:0;line-height:0;">&nbsp;</td>
                      </tr>
                    </table>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <tr>
            <td style="padding:32px;">
              <div style="font-size:13px;font-weight:800;line-height:1.5;color:#0f9f7a;">账号安全</div>
              <h1 style="margin:6px 0 0;font-size:28px;font-weight:800;line-height:1.3;color:#0f172a;">邮箱验证码</h1>
              <p style="margin:12px 0 0;font-size:15px;line-height:1.7;color:#475569;">你正在注册选科π账号，请在验证页面输入以下 6 位验证码。</p>

              <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;margin:24px 0;">
                <tr>
                  {{range .Digits}}<td width="16.66%" style="padding:0 3px;">
                    <div style="padding:16px 0;border:1px solid #dbe3ee;border-radius:8px;background:#f8fafc;color:#0f172a;font-family:SFMono-Regular,Consolas,'Liberation Mono',monospace;font-size:30px;font-weight:800;line-height:1;text-align:center;">{{.}}</div>
                  </td>{{end}}
                </tr>
              </table>

              <table role="presentation" width="100%" cellspacing="0" cellpadding="0" border="0" style="width:100%;">
                <tr>
                  <td style="width:4px;background:#0f9f7a;font-size:0;line-height:0;">&nbsp;</td>
                  <td style="padding:12px 14px;background:#f8fafc;font-size:14px;line-height:1.7;color:#475569;">
                    验证码将在 <strong style="color:#0f172a;">{{.ExpiresInMinutes}} 分钟</strong>后失效。为保障账号安全，请勿转发给任何人。
                  </td>
                </tr>
              </table>

              <div style="margin-top:24px;padding-top:20px;border-top:1px solid #e2e8f0;font-size:13px;line-height:1.7;color:#64748b;">如果这不是你的操作，请忽略此邮件，你的账号不会受到影响。</div>
            </td>
          </tr>
          <tr>
            <td style="padding:18px 32px;border-top:1px solid #e2e8f0;background:#f8fafc;font-size:12px;line-height:1.6;color:#86868b;">
              此邮件由选科π系统自动发送，请勿直接回复。
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`))

func NewSMTPEmailSender(cfg config.Config, logger *logx.Logger) *SMTPEmailSender {
	return &SMTPEmailSender{cfg: cfg, logger: logger}
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
		if err != nil && s.logger != nil {
			s.logger.Error("邮件", "验证码发送失败", logx.F("SMTP服务器", net.JoinHostPort(s.cfg.SMTPHost, fmt.Sprintf("%d", s.cfg.SMTPPort))), logx.F("错误", err))
		}
		return err
	}
}

func (s *SMTPEmailSender) send(to string, code string, ttl time.Duration) error {
	addr := net.JoinHostPort(s.cfg.SMTPHost, fmt.Sprintf("%d", s.cfg.SMTPPort))
	auth := smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	message, err := s.message(to, code, ttl)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	if s.cfg.SMTPUseTLS {
		if err := s.sendWithTLS(addr, auth, to, message); err != nil {
			return fmt.Errorf("implicit TLS delivery: %w", err)
		}
		return nil
	}
	if s.cfg.SMTPStartTLS {
		if err := s.sendWithStartTLS(addr, auth, to, message); err != nil {
			return fmt.Errorf("STARTTLS delivery: %w", err)
		}
		return nil
	}
	if err := smtp.SendMail(addr, auth, s.cfg.SMTPFromEmail, []string{to}, message); err != nil {
		return fmt.Errorf("plain SMTP delivery: %w", err)
	}
	return nil
}

func (s *SMTPEmailSender) sendWithTLS(addr string, auth smtp.Auth, to string, message []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.cfg.SMTPHost, MinVersion: tls.VersionTLS12})
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	defer client.Quit()

	return s.sendWithClient(client, auth, to, message)
}

func (s *SMTPEmailSender) sendWithStartTLS(addr string, auth smtp.Auth, to string, message []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Quit()

	if err := client.StartTLS(&tls.Config{ServerName: s.cfg.SMTPHost, MinVersion: tls.VersionTLS12}); err != nil {
		return fmt.Errorf("negotiate TLS: %w", err)
	}
	return s.sendWithClient(client, auth, to, message)
}

func (s *SMTPEmailSender) sendWithClient(client *smtp.Client, auth smtp.Auth, to string, message []byte) error {
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}
	if err := client.Mail(s.cfg.SMTPFromEmail); err != nil {
		return fmt.Errorf("set sender: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("set recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open message body: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		writer.Close()
		return fmt.Errorf("write message body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("commit message: %w", err)
	}
	return nil
}

func (s *SMTPEmailSender) message(to string, code string, ttl time.Duration) ([]byte, error) {
	brandName := sanitizeEmailHeader(strings.TrimSpace(s.cfg.SMTPFromName))
	if brandName == "" {
		brandName = "SoulCourse"
	}
	expiresInMinutes := int(ttl.Minutes())
	if expiresInMinutes < 1 {
		expiresInMinutes = 1
	}
	templateData := verificationEmailTemplateData{
		BrandName:        brandName,
		Digits:           strings.Split(code, ""),
		ExpiresInMinutes: expiresInMinutes,
	}

	plainBody := fmt.Sprintf(`%s 邮箱验证

你正在注册 %s 账号。

验证码：%s

验证码将在 %d 分钟后失效，请勿转发给任何人。
如果这不是你的操作，可以忽略此邮件，你的账号不会受到影响。

此邮件由 %s 系统自动发送，请勿直接回复。`,
		brandName, brandName, code, expiresInMinutes, brandName)

	var htmlBody bytes.Buffer
	if err := verificationEmailHTMLTemplate.Execute(&htmlBody, templateData); err != nil {
		return nil, fmt.Errorf("render verification email: %w", err)
	}

	var alternativeBody bytes.Buffer
	alternativeWriter := multipart.NewWriter(&alternativeBody)
	if err := writeQuotedPrintablePart(alternativeWriter, `text/plain; charset="UTF-8"`, plainBody); err != nil {
		return nil, err
	}
	if err := writeQuotedPrintablePart(alternativeWriter, `text/html; charset="UTF-8"`, htmlBody.String()); err != nil {
		return nil, err
	}
	if err := alternativeWriter.Close(); err != nil {
		return nil, fmt.Errorf("close verification email body: %w", err)
	}

	var relatedBody bytes.Buffer
	relatedWriter := multipart.NewWriter(&relatedBody)
	alternativeHeaders := make(textproto.MIMEHeader)
	alternativeHeaders.Set("Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, alternativeWriter.Boundary()))
	alternativePart, err := relatedWriter.CreatePart(alternativeHeaders)
	if err != nil {
		return nil, fmt.Errorf("create verification email alternatives: %w", err)
	}
	if _, err := alternativePart.Write(alternativeBody.Bytes()); err != nil {
		return nil, fmt.Errorf("write verification email alternatives: %w", err)
	}
	if err := writeInlinePNG(relatedWriter, "soulcourse-logo", "logo-mark.png", brandassets.LogoMarkPNG); err != nil {
		return nil, err
	}
	if err := relatedWriter.Close(); err != nil {
		return nil, fmt.Errorf("close verification email related body: %w", err)
	}

	from := mail.Address{Name: sanitizeEmailHeader(brandName), Address: sanitizeEmailHeader(s.cfg.SMTPFromEmail)}
	toAddress := mail.Address{Address: sanitizeEmailHeader(to)}
	subject := mime.QEncoding.Encode("UTF-8", sanitizeEmailHeader(s.cfg.EmailVerificationSubject))

	var message bytes.Buffer
	headers := [][2]string{
		{"From", from.String()},
		{"To", toAddress.String()},
		{"Subject", subject},
		{"Date", time.Now().Format(time.RFC1123Z)},
		{"MIME-Version", "1.0"},
		{"Content-Type", fmt.Sprintf(`multipart/related; boundary="%s"`, relatedWriter.Boundary())},
	}
	for _, header := range headers {
		message.WriteString(header[0])
		message.WriteString(": ")
		message.WriteString(header[1])
		message.WriteString("\r\n")
	}
	message.WriteString("\r\n")
	message.Write(relatedBody.Bytes())
	return message.Bytes(), nil
}

func writeQuotedPrintablePart(writer *multipart.Writer, contentType string, content string) error {
	headers := make(textproto.MIMEHeader)
	headers.Set("Content-Type", contentType)
	headers.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := writer.CreatePart(headers)
	if err != nil {
		return fmt.Errorf("create verification email part: %w", err)
	}
	quotedPrintableWriter := quotedprintable.NewWriter(part)
	if _, err := quotedPrintableWriter.Write([]byte(content)); err != nil {
		quotedPrintableWriter.Close()
		return fmt.Errorf("write verification email part: %w", err)
	}
	if err := quotedPrintableWriter.Close(); err != nil {
		return fmt.Errorf("close verification email part: %w", err)
	}
	return nil
}

func writeInlinePNG(writer *multipart.Writer, contentID string, fileName string, image []byte) error {
	headers := make(textproto.MIMEHeader)
	headers.Set("Content-Type", fmt.Sprintf(`image/png; name="%s"`, fileName))
	headers.Set("Content-Transfer-Encoding", "base64")
	headers.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, fileName))
	headers.Set("Content-ID", fmt.Sprintf("<%s>", contentID))
	part, err := writer.CreatePart(headers)
	if err != nil {
		return fmt.Errorf("create verification email logo: %w", err)
	}

	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(image)))
	base64.StdEncoding.Encode(encoded, image)
	for len(encoded) > 0 {
		lineLength := 76
		if len(encoded) < lineLength {
			lineLength = len(encoded)
		}
		if _, err := part.Write(encoded[:lineLength]); err != nil {
			return fmt.Errorf("write verification email logo: %w", err)
		}
		if _, err := part.Write([]byte("\r\n")); err != nil {
			return fmt.Errorf("write verification email logo line ending: %w", err)
		}
		encoded = encoded[lineLength:]
	}
	return nil
}

func sanitizeEmailHeader(value string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(value)
}
