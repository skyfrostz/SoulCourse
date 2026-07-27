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
	return r.limit, nil
}

func (r *verificationRepositoryStub) CreateEmailVerificationCode(context.Context, string, string, time.Time) error {
	r.codeCreated = true
	return nil
}

type verificationEmailSenderStub struct {
	sent bool
}

func (s *verificationEmailSenderStub) Enabled() bool {
	return true
}

func (s *verificationEmailSenderStub) SendVerificationCode(context.Context, string, string, time.Duration) error {
	s.sent = true
	return nil
}

func TestSendEmailVerificationCodeReturnsServerRateLimitState(t *testing.T) {
	repository := &verificationRepositoryStub{limit: domain.EmailVerificationAttemptLimit{
		Allowed:              true,
		RetryAfterSeconds:    60,
		EmailHourlyLimit:     5,
		EmailHourlyRemaining: 4,
	}}
	sender := &verificationEmailSenderStub{}
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
	sender := &verificationEmailSenderStub{}
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
