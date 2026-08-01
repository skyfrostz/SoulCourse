package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/http/middleware"
	"subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

type accountHandlerHarness struct {
	db     *sql.DB
	router *gin.Engine
}

func newAccountHandlerHarness(t *testing.T) *accountHandlerHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		AppEnv:         "local",
		SQLitePath:     filepath.Join(t.TempDir(), "accounts.db"),
		MediaUploadDir: filepath.Join(t.TempDir(), "uploads"),
		JWTSecret:      "account-handler-test-secret",
	}
	db, err := storage.NewSQLiteDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	forumService := service.NewForumService(sqlite.NewForumRepository(db), cfg, nil)
	handler := NewForumHandler(forumService, nil, true, cfg.MediaUploadDir, "/choice")

	router := gin.New()
	router.POST("/verification", handler.SendEmailVerificationCode)
	router.POST("/forgot", handler.ForgotPassword)
	router.POST("/register", handler.Register)
	router.POST("/login", handler.Login)
	router.POST("/reset", handler.ResetPassword)
	authenticated := router.Group("/")
	authenticated.Use(middleware.RequireAuth(forumService))
	authenticated.POST("/logout", handler.Logout)
	authenticated.GET("/me", handler.Me)
	authenticated.DELETE("/me", handler.DeleteMe)
	authenticated.GET("/profile", handler.GetMyProfile)
	authenticated.GET("/profiles/:name", handler.GetProfile)
	authenticated.PUT("/profile", handler.UpdateMyProfile)
	authenticated.GET("/sessions", handler.ListMySessions)
	authenticated.DELETE("/sessions/:id", handler.RevokeMySession)
	return &accountHandlerHarness{db: db, router: router}
}

func TestForumAccountHandlersLifecycle(t *testing.T) {
	h := newAccountHandlerHarness(t)
	email := "beta.student@example.com"
	password := "initial-pass"

	assertHandlerError(t, h.request(http.MethodPost, "/verification", `{}`, nil), http.StatusBadRequest, "invalid_payload")
	code := h.verificationCode(t, "/verification", email)
	assertHandlerError(t, h.request(http.MethodPost, "/register", `{"email":"bad"}`, nil), http.StatusBadRequest, "invalid_payload")
	assertHandlerError(t, h.request(http.MethodPost, "/register", registerBody(email, password, "000000"), nil), http.StatusBadRequest, "invalid_verification_code")

	// A failed code attempt does not consume the valid code.
	registered := h.request(http.MethodPost, "/register", registerBody(email, password, code), nil)
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", registered.Code, registered.Body.String())
	}
	if strings.Contains(registered.Body.String(), `"token"`) {
		t.Fatalf("register response leaked session token: %s", registered.Body.String())
	}
	sessionCookie := responseCookie(t, registered, middleware.SessionCookieName)
	csrfCookie := responseCookie(t, registered, middleware.CSRFCookieName)
	if !sessionCookie.Secure || !sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("session cookie flags are incomplete: %+v", sessionCookie)
	}
	if !csrfCookie.Secure || csrfCookie.HttpOnly || csrfCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("csrf cookie flags are incorrect: %+v", csrfCookie)
	}

	assertHandlerError(t, h.request(http.MethodPost, "/login", `{}`, nil), http.StatusBadRequest, "invalid_payload")
	assertHandlerError(t, h.request(http.MethodPost, "/login", loginBody(email, "wrong-pass"), nil), http.StatusUnauthorized, "invalid_credentials")
	loggedIn := h.request(http.MethodPost, "/login", loginBody(email, password), nil)
	if loggedIn.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", loggedIn.Code, loggedIn.Body.String())
	}
	if strings.Contains(loggedIn.Body.String(), `"token"`) {
		t.Fatalf("login response leaked session token: %s", loggedIn.Body.String())
	}
	currentCookie := responseCookie(t, loggedIn, middleware.SessionCookieName)

	me := h.request(http.MethodGet, "/me", "", []*http.Cookie{currentCookie})
	if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), email) {
		t.Fatalf("me status=%d body=%s", me.Code, me.Body.String())
	}
	profile := h.request(http.MethodGet, "/profile", "", []*http.Cookie{currentCookie})
	if profile.Code != http.StatusOK || !strings.Contains(profile.Body.String(), "公测学生") {
		t.Fatalf("profile status=%d body=%s", profile.Code, profile.Body.String())
	}
	publicProfile := h.request(http.MethodGet, "/profiles/公测学生", "", []*http.Cookie{currentCookie})
	if publicProfile.Code != http.StatusOK {
		t.Fatalf("public profile status=%d body=%s", publicProfile.Code, publicProfile.Body.String())
	}
	assertHandlerError(t, h.request(http.MethodGet, "/profiles/不存在", "", []*http.Cookie{currentCookie}), http.StatusNotFound, "not_found")

	assertHandlerError(t, h.request(http.MethodPut, "/profile", `{}`, []*http.Cookie{currentCookie}), http.StatusBadRequest, "invalid_payload")
	invalidElectives := `{"bio":"准备公测","choiceProfile":{"preferredTrack":"physics","preferredSubjects":["chemistry","chemistry"]}}`
	assertHandlerError(t, h.request(http.MethodPut, "/profile", invalidElectives, []*http.Cookie{currentCookie}), http.StatusBadRequest, "invalid_electives")
	validProfile := `{"bio":"准备参加广东公测","choiceProfile":{"city":"广州","preferredTrack":"physics","preferredSubjects":["chemistry","biology"]}}`
	updated := h.request(http.MethodPut, "/profile", validProfile, []*http.Cookie{currentCookie})
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), "准备参加广东公测") {
		t.Fatalf("update profile status=%d body=%s", updated.Code, updated.Body.String())
	}

	sessions := h.request(http.MethodGet, "/sessions", "", []*http.Cookie{currentCookie})
	currentSessionID := currentAccountSessionID(t, sessions)
	assertHandlerError(t, h.request(http.MethodDelete, "/sessions/nope", "", []*http.Cookie{currentCookie}), http.StatusBadRequest, "invalid_session")
	assertHandlerError(t, h.request(http.MethodDelete, "/sessions/999999", "", []*http.Cookie{currentCookie}), http.StatusNotFound, "not_found")
	revoked := h.request(http.MethodDelete, "/sessions/"+stringInt64ForHandlerTest(currentSessionID), "", []*http.Cookie{currentCookie})
	if revoked.Code != http.StatusOK || responseCookie(t, revoked, middleware.SessionCookieName).MaxAge != -1 {
		t.Fatalf("revoke current status=%d body=%s cookies=%v", revoked.Code, revoked.Body.String(), revoked.Result().Cookies())
	}

	// Use the original registration session for the remaining account operations.
	assertHandlerError(t, h.request(http.MethodDelete, "/me", `{}`, []*http.Cookie{sessionCookie}), http.StatusBadRequest, "invalid_payload")
	assertHandlerError(t, h.request(http.MethodDelete, "/me", `{"password":"wrong-pass"}`, []*http.Cookie{sessionCookie}), http.StatusUnauthorized, "invalid_credentials")

	if _, err := h.db.Exec("DELETE FROM email_verification_attempts WHERE email = ?", email); err != nil {
		t.Fatal(err)
	}
	resetCode := h.verificationCode(t, "/forgot", email)
	assertHandlerError(t, h.request(http.MethodPost, "/reset", resetBody(email, "000000", "new-password"), nil), http.StatusBadRequest, "invalid_verification_code")
	reset := h.request(http.MethodPost, "/reset", resetBody(email, resetCode, "new-password"), nil)
	if reset.Code != http.StatusOK || responseCookie(t, reset, middleware.SessionCookieName).MaxAge != -1 {
		t.Fatalf("reset status=%d body=%s cookies=%v", reset.Code, reset.Body.String(), reset.Result().Cookies())
	}
	assertHandlerError(t, h.request(http.MethodPost, "/login", loginBody(email, password), nil), http.StatusUnauthorized, "invalid_credentials")
	newLogin := h.request(http.MethodPost, "/login", loginBody(email, "new-password"), nil)
	newCookie := responseCookie(t, newLogin, middleware.SessionCookieName)

	logout := h.request(http.MethodPost, "/logout", "", []*http.Cookie{newCookie})
	if logout.Code != http.StatusOK || responseCookie(t, logout, middleware.SessionCookieName).MaxAge != -1 {
		t.Fatalf("logout status=%d body=%s", logout.Code, logout.Body.String())
	}
	finalLogin := h.request(http.MethodPost, "/login", loginBody(email, "new-password"), nil)
	finalCookie := responseCookie(t, finalLogin, middleware.SessionCookieName)
	deleted := h.request(http.MethodDelete, "/me", `{"password":"new-password"}`, []*http.Cookie{finalCookie})
	if deleted.Code != http.StatusOK || !strings.Contains(deleted.Body.String(), `"deleted":true`) {
		t.Fatalf("delete account status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	assertHandlerError(t, h.request(http.MethodPost, "/login", loginBody(email, "new-password"), nil), http.StatusUnauthorized, "invalid_credentials")
}

func TestForumAccountHandlerErrorBranches(t *testing.T) {
	h := newAccountHandlerHarness(t)
	assertHandlerError(t, h.request(http.MethodPost, "/reset", `{}`, nil), http.StatusBadRequest, "invalid_payload")
	assertHandlerError(t, h.request(http.MethodPost, "/reset", resetBody("missing@example.com", "123456", "password"), nil), http.StatusBadRequest, "invalid_verification_code")
	assertHandlerError(t, h.request(http.MethodGet, "/me", "", nil), http.StatusUnauthorized, "unauthorized")
}

func (h *accountHandlerHarness) request(method, path, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	request.RemoteAddr = "192.0.2.10:4321"
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	h.router.ServeHTTP(recorder, request)
	return recorder
}

func (h *accountHandlerHarness) verificationCode(t *testing.T, path, email string) string {
	t.Helper()
	response := h.request(http.MethodPost, path, `{"email":"`+email+`"}`, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("verification status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Data struct {
			DebugCode string `json:"debugCode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Data.DebugCode) != 6 {
		t.Fatalf("missing debug verification code: %s", response.Body.String())
	}
	return payload.Data.DebugCode
}

func responseCookie(t *testing.T, response *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("response omitted cookie %q: %v", name, response.Result().Cookies())
	return nil
}

func currentAccountSessionID(t *testing.T, response *httptest.ResponseRecorder) int64 {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("sessions status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Data []domain.AccountSession `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for _, session := range payload.Data {
		if session.Current {
			return session.ID
		}
	}
	t.Fatalf("sessions omitted current session: %s", response.Body.String())
	return 0
}

func registerBody(email, password, code string) string {
	return `{"email":"` + email + `","password":"` + password + `","verificationCode":"` + code + `","nickname":"公测学生","role":"student","province":"广东","grade":"高一"}`
}

func loginBody(email, password string) string {
	return `{"email":"` + email + `","password":"` + password + `"}`
}

func resetBody(email, code, password string) string {
	return `{"email":"` + email + `","verificationCode":"` + code + `","password":"` + password + `"}`
}
