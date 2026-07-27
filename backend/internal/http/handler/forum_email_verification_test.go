package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type rateLimitRepositoryStub struct {
	service.ForumRepository
}

func (rateLimitRepositoryStub) ReserveEmailVerificationAttempt(
	context.Context,
	string,
	string,
	time.Time,
	time.Duration,
	int,
	int,
) (domain.EmailVerificationAttemptLimit, error) {
	return domain.EmailVerificationAttemptLimit{
		Allowed:              false,
		RetryAfterSeconds:    42,
		EmailHourlyLimit:     5,
		EmailHourlyRemaining: 3,
		Scope:                "cooldown",
	}, nil
}

func TestSendEmailVerificationCodeReturnsRateLimitMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	forumService := service.NewForumService(rateLimitRepositoryStub{}, config.Config{
		JWTSecret: "test-secret",
	}, nil)
	forumHandler := NewForumHandler(forumService, nil)
	router := gin.New()
	router.POST("/api/v1/auth/email-verification-code", forumHandler.SendEmailVerificationCode)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/email-verification-code",
		bytes.NewBufferString(`{"email":"student@example.com"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = "192.0.2.40:54321"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("unexpected status: %d body=%s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Retry-After"); got != "42" {
		t.Fatalf("unexpected Retry-After header: %q", got)
	}

	var body struct {
		Error struct {
			Code              string `json:"code"`
			RetryAfterSeconds int    `json:"retryAfterSeconds"`
			HourlyLimit       int    `json:"hourlyLimit"`
			HourlyRemaining   int    `json:"hourlyRemaining"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "email_verification_rate_limited" ||
		body.Error.RetryAfterSeconds != 42 ||
		body.Error.HourlyLimit != 5 ||
		body.Error.HourlyRemaining != 3 {
		t.Fatalf("unexpected response: %+v", body.Error)
	}
}
