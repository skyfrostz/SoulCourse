package handler

import (
	"net/http"
	"strings"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/http/middleware"
	"subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func TestAdminOperationalHandlers(t *testing.T) {
	db, cfg := newAdminHandlerDB(t)
	cfg.AppEnv = "local"
	cfg.SMTPHost = "smtp.example.com"
	cfg.SMTPPort = 587
	cfg.SMTPUsername = "mailer"
	cfg.SMTPPassword = "must-not-leak"
	cfg.SMTPFromEmail = "noreply@example.com"
	cfg.SMTPReplyTo = "support@example.com"
	cfg.SMTPFromName = "Beta Mailer"
	cfg.SMTPStartTLS = true
	cfg.EmailVerificationTTLMinutes = 12
	cfg.EmailVerificationCooldownSeconds = 30
	cfg.EmailVerificationEmailHourlyLimit = 4
	cfg.EmailVerificationIPHourlyLimit = 10
	cfg.EmailVerificationMaxValidationAttempts = 3
	forumService := service.NewForumService(sqlite.NewForumRepository(db), cfg, nil)
	store := middleware.NewAdminSessionStore(0)
	handler := NewAdminHandler(cfg, forumService, db, store)
	router := gin.New()
	router.Use(middleware.RequestID())
	router.GET("/email-config", handler.EmailConfig)
	router.POST("/test-email", handler.SendTestEmail)
	router.GET("/published", handler.ListPublishedContent)
	router.POST("/logout", handler.Logout)

	configResponse := performHandlerRequest(router, http.MethodGet, "/email-config", "")
	if configResponse.Code != http.StatusOK {
		t.Fatalf("email config status=%d body=%s", configResponse.Code, configResponse.Body.String())
	}
	for _, expected := range []string{`"enabled":true`, `"host":"smtp.example.com"`, `"passwordConfigured":true`, `"emailVerificationTTLMinutes":12`} {
		if !strings.Contains(configResponse.Body.String(), expected) {
			t.Fatalf("email config missing %s: %s", expected, configResponse.Body.String())
		}
	}
	if strings.Contains(configResponse.Body.String(), cfg.SMTPPassword) {
		t.Fatalf("email config leaked SMTP password: %s", configResponse.Body.String())
	}

	assertHandlerError(t, performHandlerRequest(router, http.MethodPost, "/test-email", `{}`), http.StatusBadRequest, "invalid_payload")
	testEmail := performHandlerRequest(router, http.MethodPost, "/test-email", `{"email":"operator@example.com"}`)
	if testEmail.Code != http.StatusOK || !strings.Contains(testEmail.Body.String(), `"debugCode"`) {
		t.Fatalf("test email status=%d body=%s", testEmail.Code, testEmail.Body.String())
	}

	created := performHandlerRequest(adminHandlerRouter(t, handler, middleware.AdminRoleSuperAdmin), http.MethodPost, "/admin/content", `{
		"id":"published-beta-policy","module":"policies","title":"公测政策",
		"status":"已上架","scope":"广东","owner":"考试院"
	}`)
	if created.Code != http.StatusOK {
		t.Fatalf("seed published content status=%d body=%s", created.Code, created.Body.String())
	}
	published := performHandlerRequest(router, http.MethodGet, "/published?module=policies&q=公测", "")
	if published.Code != http.StatusOK || !strings.Contains(published.Body.String(), "published-beta-policy") {
		t.Fatalf("published status=%d body=%s", published.Code, published.Body.String())
	}
	users := performHandlerRequest(router, http.MethodGet, "/published?module=users", "")
	if users.Code != http.StatusOK || !strings.Contains(users.Body.String(), `"records":[]`) {
		t.Fatalf("published users status=%d body=%s", users.Code, users.Body.String())
	}

	logout := performHandlerRequest(router, http.MethodPost, "/logout", "")
	if logout.Code != http.StatusOK || !strings.Contains(logout.Body.String(), `"signedOut":true`) {
		t.Fatalf("logout status=%d body=%s", logout.Code, logout.Body.String())
	}
	if responseCookie(t, logout, middleware.AdminSessionCookieName).MaxAge != -1 || responseCookie(t, logout, middleware.AdminCSRFCookieName).MaxAge != -1 {
		t.Fatalf("admin logout did not clear cookies: %v", logout.Result().Cookies())
	}
}

func TestAdminEmailConfigReportsMissingSettings(t *testing.T) {
	handler := NewAdminHandler(config.Config{}, nil, nil, middleware.NewAdminSessionStore(0))
	router := gin.New()
	router.GET("/email-config", handler.EmailConfig)
	response := performHandlerRequest(router, http.MethodGet, "/email-config", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"enabled":false`) {
		t.Fatalf("email config status=%d body=%s", response.Code, response.Body.String())
	}
	for _, setting := range []string{"SMTP_HOST", "SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_FROM_EMAIL"} {
		if !strings.Contains(response.Body.String(), setting) {
			t.Fatalf("missing setting %s was not reported: %s", setting, response.Body.String())
		}
	}
}
