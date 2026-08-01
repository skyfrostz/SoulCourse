package httpserver

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/logx"
	sqliterepo "subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"

	"golang.org/x/crypto/bcrypt"
)

func TestAdminBanUserRevokesSessionsAndRestoreAllowsLogin(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := config.Config{
		AppEnv:             "test",
		SQLitePath:         filepath.Join(tempDir, "admin-user-moderation.db"),
		MediaUploadDir:     filepath.Join(tempDir, "uploads"),
		AdminEmail:         "admin@example.com",
		AdminPassword:      "admin-password",
		AdminToken:         "test-admin-token-that-stays-on-the-server",
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	}
	db, err := storage.NewSQLiteDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	repository := sqliterepo.NewForumRepository(db)
	forum := service.NewForumService(repository, cfg, nil)
	server := NewServer(cfg, logx.New(io.Discard, logx.LevelError), db, forum)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("student-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := repository.CreateUser(context.Background(), domain.RegisterInput{
		Email: "student@example.com", Nickname: "被封禁用户", Role: "student", Province: "广东", Grade: "高一",
	}, string(passwordHash))
	if err != nil {
		t.Fatal(err)
	}
	session, err := forum.Login(context.Background(), domain.LoginInput{Email: user.Email, Password: "student-password"})
	if err != nil {
		t.Fatal(err)
	}

	adminCookies := loginAdminForTest(t, server)
	csrf := cookieValue(adminCookies, "scf_admin_csrf")

	banRecorder := httptest.NewRecorder()
	banRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+stringInt64(user.ID)+"/ban", bytes.NewBufferString(`{"reason":"测试封禁"}`))
	banRequest.Header.Set("Content-Type", "application/json")
	banRequest.Header.Set("X-CSRF-Token", csrf)
	for _, cookie := range adminCookies {
		banRequest.AddCookie(cookie)
	}
	server.Handler.ServeHTTP(banRecorder, banRequest)
	if banRecorder.Code != http.StatusOK {
		t.Fatalf("ban status = %d body=%s", banRecorder.Code, banRecorder.Body.String())
	}

	meRecorder := httptest.NewRecorder()
	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+session.Token)
	server.Handler.ServeHTTP(meRecorder, meRequest)
	if meRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status = %d, want %d", meRecorder.Code, http.StatusUnauthorized)
	}

	if _, err := forum.Login(context.Background(), domain.LoginInput{Email: user.Email, Password: "student-password"}); err == nil {
		t.Fatal("banned user login unexpectedly succeeded")
	}

	restoreRecorder := httptest.NewRecorder()
	restoreRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/"+stringInt64(user.ID)+"/restore", bytes.NewBufferString(`{"reason":"测试恢复"}`))
	restoreRequest.Header.Set("Content-Type", "application/json")
	restoreRequest.Header.Set("X-CSRF-Token", csrf)
	for _, cookie := range adminCookies {
		restoreRequest.AddCookie(cookie)
	}
	server.Handler.ServeHTTP(restoreRecorder, restoreRequest)
	if restoreRecorder.Code != http.StatusOK {
		t.Fatalf("restore status = %d body=%s", restoreRecorder.Code, restoreRecorder.Body.String())
	}
	if _, err := forum.Login(context.Background(), domain.LoginInput{Email: user.Email, Password: "student-password"}); err != nil {
		t.Fatalf("restored user login failed: %v", err)
	}

	var auditCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_audit_logs WHERE action IN ('ban_user', 'restore_user') AND record_id = ?`, "user-"+stringInt64(user.ID)).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 2 {
		t.Fatalf("audit count = %d, want 2", auditCount)
	}
}

func loginAdminForTest(t *testing.T, server *http.Server) []*http.Cookie {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"admin-password"}`))
	request.Header.Set("Content-Type", "application/json")
	server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("admin login status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	return recorder.Result().Cookies()
}
