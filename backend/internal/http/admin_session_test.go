package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/logx"
	sqliterepo "subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"
)

func TestAdminLoginIssuesCookieSessionWithoutReturningStaticToken(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := config.Config{
		AppEnv:             "test",
		SQLitePath:         filepath.Join(tempDir, "admin-session.db"),
		MediaUploadDir:     filepath.Join(tempDir, "uploads"),
		AdminEmail:         "admin@example.com",
		AdminPassword:      "admin-password",
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

	forum := service.NewForumService(sqliterepo.NewForumRepository(db), cfg, nil)
	server := NewServer(cfg, logx.New(io.Discard, logx.LevelError), db, forum)

	loginBody := bytes.NewBufferString(`{"email":"admin@example.com","password":"admin-password"}`)
	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", loginBody)
	loginRequest.Header.Set("Content-Type", "application/json")
	server.Handler.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", loginRecorder.Code, loginRecorder.Body.String())
	}
	if strings.Contains(loginRecorder.Body.String(), "test-admin-token") {
		t.Fatal("admin login response leaked the static admin token")
	}
	var loginEnvelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(loginRecorder.Body.Bytes(), &loginEnvelope); err != nil {
		t.Fatal(err)
	}
	if _, ok := loginEnvelope.Data["token"]; ok {
		t.Fatal("admin login response must not include token")
	}

	emailConfigRecorder := httptest.NewRecorder()
	emailConfigRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/email-config", nil)
	for _, cookie := range loginRecorder.Result().Cookies() {
		emailConfigRequest.AddCookie(cookie)
	}
	server.Handler.ServeHTTP(emailConfigRecorder, emailConfigRequest)
	if emailConfigRecorder.Code != http.StatusOK {
		t.Fatalf("email config status = %d body=%s", emailConfigRecorder.Code, emailConfigRecorder.Body.String())
	}

	headerOnlyRecorder := httptest.NewRecorder()
	headerOnlyRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/email-config", nil)
	headerOnlyRequest.Header.Set("X-Admin-Token", "test-admin-token-that-stays-on-the-server")
	server.Handler.ServeHTTP(headerOnlyRecorder, headerOnlyRequest)
	if headerOnlyRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("header token status = %d, want %d", headerOnlyRecorder.Code, http.StatusUnauthorized)
	}

	logoutRecorder := httptest.NewRecorder()
	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/logout", nil)
	for _, cookie := range loginRecorder.Result().Cookies() {
		logoutRequest.AddCookie(cookie)
		if cookie.Name == "scf_admin_csrf" {
			logoutRequest.Header.Set("X-CSRF-Token", cookie.Value)
		}
	}
	server.Handler.ServeHTTP(logoutRecorder, logoutRequest)
	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("logout status = %d body=%s", logoutRecorder.Code, logoutRecorder.Body.String())
	}
	if !hasExpiredCookie(logoutRecorder.Result().Cookies(), "scf_admin_session") ||
		!hasExpiredCookie(logoutRecorder.Result().Cookies(), "scf_admin_csrf") {
		t.Fatalf("logout did not clear admin cookies: %#v", logoutRecorder.Result().Cookies())
	}

	afterLogoutRecorder := httptest.NewRecorder()
	afterLogoutRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/email-config", nil)
	for _, cookie := range loginRecorder.Result().Cookies() {
		afterLogoutRequest.AddCookie(cookie)
	}
	server.Handler.ServeHTTP(afterLogoutRecorder, afterLogoutRequest)
	if afterLogoutRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("after logout status = %d, want %d", afterLogoutRecorder.Code, http.StatusUnauthorized)
	}
}

func TestAdminContentEditorPermissionsAreEnforcedByBackend(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.Config{
		AppEnv: "test", SQLitePath: filepath.Join(tempDir, "admin-rbac.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
		AdminEmail:     "editor@example.com", AdminPassword: "admin-password", AdminRole: "content_editor",
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	}
	db, err := storage.NewSQLiteDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	forum := service.NewForumService(sqliterepo.NewForumRepository(db), cfg, nil)
	if _, err := db.Exec(`INSERT INTO users (email, password_hash, nickname, role, province, grade, created_at, updated_at) VALUES ('private@example.com', 'hash', '隐私测试用户', 'student', '广东', '高一', '2026-07-31T00:00:00Z', '2026-07-31T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, logx.New(io.Discard, logx.LevelError), db, forum)

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewBufferString(`{"email":"editor@example.com","password":"admin-password"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	server.Handler.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK || !strings.Contains(loginRecorder.Body.String(), `"role":"content_editor"`) {
		t.Fatalf("login status=%d body=%s", loginRecorder.Code, loginRecorder.Body.String())
	}

	for _, testCase := range []struct {
		path string
		want int
	}{
		{path: "/api/v1/admin/content", want: http.StatusOK},
		{path: "/api/v1/admin/reports", want: http.StatusForbidden},
		{path: "/api/v1/admin/email-config", want: http.StatusForbidden},
		{path: "/api/v1/admin/audit-logs", want: http.StatusForbidden},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
		for _, cookie := range loginRecorder.Result().Cookies() {
			request.AddCookie(cookie)
		}
		server.Handler.ServeHTTP(recorder, request)
		if recorder.Code != testCase.want {
			t.Errorf("GET %s status=%d want=%d body=%s", testCase.path, recorder.Code, testCase.want, recorder.Body.String())
		}
		if testCase.path == "/api/v1/admin/content" && strings.Contains(recorder.Body.String(), `"module":"users"`) {
			t.Errorf("content editor response leaked user records: %s", recorder.Body.String())
		}
	}
}

func hasExpiredCookie(cookies []*http.Cookie, name string) bool {
	for _, cookie := range cookies {
		if cookie.Name == name && cookie.MaxAge < 0 {
			return true
		}
	}
	return false
}
