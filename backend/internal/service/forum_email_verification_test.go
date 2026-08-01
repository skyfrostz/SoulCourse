package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
)

type verificationRepositoryStub struct {
	ForumRepository
	limit       domain.EmailVerificationAttemptLimit
	reserveErr  error
	createErr   error
	codeCreated bool
}

func (r *verificationRepositoryStub) ReserveEmailVerificationAttempt(
	context.Context,
	string,
	string,
	time.Time,
	time.Duration,
	int,
	int,
) (domain.EmailVerificationAttemptLimit, error) {
	return r.limit, r.reserveErr
}

func (r *verificationRepositoryStub) CreateEmailVerificationCode(context.Context, string, string, time.Time) error {
	r.codeCreated = true
	return r.createErr
}

type verificationEmailSenderStub struct {
	sent    bool
	enabled bool
	err     error
}

func (s *verificationEmailSenderStub) Enabled() bool {
	return s.enabled
}

func (s *verificationEmailSenderStub) SendVerificationCode(context.Context, string, string, time.Duration) error {
	s.sent = true
	return s.err
}

func TestSendEmailVerificationCodeReturnsServerRateLimitState(t *testing.T) {
	repository := &verificationRepositoryStub{limit: domain.EmailVerificationAttemptLimit{
		Allowed:              true,
		RetryAfterSeconds:    60,
		EmailHourlyLimit:     5,
		EmailHourlyRemaining: 4,
	}}
	sender := &verificationEmailSenderStub{enabled: true}
	forumService := NewForumService(repository, config.Config{
		JWTSecret:                         "test-secret",
		EmailVerificationTTLMinutes:       10,
		EmailVerificationCooldownSeconds:  60,
		EmailVerificationEmailHourlyLimit: 5,
		EmailVerificationIPHourlyLimit:    20,
	}, sender)

	result, err := forumService.SendEmailVerificationCode(context.Background(), domain.EmailVerificationCodeInput{
		Email:    "Student@Example.com",
		ClientIP: "192.0.2.30",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !repository.codeCreated || !sender.sent {
		t.Fatal("verification code should be stored and sent")
	}
	if result.Email != "student@example.com" ||
		result.ExpiresInSeconds != 600 ||
		result.RetryAfterSeconds != 60 ||
		result.HourlyLimit != 5 ||
		result.HourlyRemaining != 4 ||
		result.DebugCode != "" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSendEmailVerificationCodeStopsBeforeGenerationWhenRateLimited(t *testing.T) {
	repository := &verificationRepositoryStub{limit: domain.EmailVerificationAttemptLimit{
		Allowed:              false,
		RetryAfterSeconds:    42,
		EmailHourlyLimit:     5,
		EmailHourlyRemaining: 3,
		Scope:                "cooldown",
	}}
	sender := &verificationEmailSenderStub{enabled: true}
	forumService := NewForumService(repository, config.Config{JWTSecret: "test-secret"}, sender)

	_, err := forumService.SendEmailVerificationCode(context.Background(), domain.EmailVerificationCodeInput{
		Email:    "student@example.com",
		ClientIP: "192.0.2.30",
	})
	var rateLimitError *EmailVerificationRateLimitError
	if !errors.As(err, &rateLimitError) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if rateLimitError.Limit.RetryAfterSeconds != 42 {
		t.Fatalf("unexpected rate limit: %+v", rateLimitError.Limit)
	}
	if repository.codeCreated || sender.sent {
		t.Fatal("rate-limited request must not create or send a code")
	}
}

func TestSendEmailVerificationCodeStopsOnRepositoryErrors(t *testing.T) {
	tests := []struct {
		name       string
		repository *verificationRepositoryStub
	}{
		{"reserve", &verificationRepositoryStub{reserveErr: errors.New("reserve failed")}},
		{"create", &verificationRepositoryStub{limit: domain.EmailVerificationAttemptLimit{Allowed: true}, createErr: errors.New("create failed")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &verificationEmailSenderStub{enabled: true}
			forum := NewForumService(tt.repository, config.Config{JWTSecret: "test"}, sender)
			if _, err := forum.SendEmailVerificationCode(context.Background(), domain.EmailVerificationCodeInput{Email: "a@example.com"}); err == nil {
				t.Fatal("expected repository error")
			}
			if sender.sent {
				t.Fatal("repository failure must prevent email delivery")
			}
		})
	}
}

func TestSendEmailVerificationCodeHandlesSenderAvailability(t *testing.T) {
	t.Run("sender failure", func(t *testing.T) {
		repository := &verificationRepositoryStub{limit: domain.EmailVerificationAttemptLimit{Allowed: true}}
		sender := &verificationEmailSenderStub{enabled: true, err: errors.New("smtp failed")}
		forum := NewForumService(repository, config.Config{JWTSecret: "test", AppEnv: "production"}, sender)
		if _, err := forum.SendEmailVerificationCode(context.Background(), domain.EmailVerificationCodeInput{Email: "a@example.com"}); err == nil {
			t.Fatal("expected sender error")
		}
		if !repository.codeCreated || !sender.sent {
			t.Fatal("expected code persistence before attempted delivery")
		}
	})

	t.Run("production disabled", func(t *testing.T) {
		repository := &verificationRepositoryStub{limit: domain.EmailVerificationAttemptLimit{Allowed: true}}
		forum := NewForumService(repository, config.Config{JWTSecret: "test", AppEnv: "production"}, &verificationEmailSenderStub{})
		if _, err := forum.SendEmailVerificationCode(context.Background(), domain.EmailVerificationCodeInput{Email: "a@example.com"}); err == nil {
			t.Fatal("production must reject an unavailable sender")
		}
	})

	t.Run("local debug code", func(t *testing.T) {
		repository := &verificationRepositoryStub{limit: domain.EmailVerificationAttemptLimit{Allowed: true}}
		forum := NewForumService(repository, config.Config{JWTSecret: "test", AppEnv: "local"}, nil)
		result, err := forum.SendEmailVerificationCode(context.Background(), domain.EmailVerificationCodeInput{Email: "a@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.DebugCode) != 6 {
			t.Fatalf("debug code length = %d, want 6", len(result.DebugCode))
		}
	})
}
