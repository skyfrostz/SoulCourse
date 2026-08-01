package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/logx"
	sqliterepo "subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"

	"golang.org/x/crypto/bcrypt"
)

func TestReportModerationHidesPostAndWritesAuditLog(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := config.Config{
		AppEnv:             "test",
		SQLitePath:         filepath.Join(tempDir, "moderation.db"),
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

	author := createModerationUser(t, repository, "author@example.com", "作者同学")
	reporter := createModerationUser(t, repository, "reporter@example.com", "举报同学")
	post, err := repository.CreatePost(context.Background(), author, domain.CreatePostInput{
		Title:     "需要审核的测试帖子",
		Content:   "这是一段用于举报审核闭环的测试内容。",
		Track:     domain.TrackPhysics,
		Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology},
		Category:  domain.CategoryQuestion,
	})
	if err != nil {
		t.Fatal(err)
	}
	reporterSession, err := forum.Login(context.Background(), domain.LoginInput{Email: reporter.Email, Password: "moderation-password"})
	if err != nil {
		t.Fatal(err)
	}

	reportRecorder := httptest.NewRecorder()
	reportRequest := httptest.NewRequest(http.MethodPost, "/api/v1/posts/"+stringInt64(post.ID)+"/report", bytes.NewBufferString(`{"reason":"疑似误导","detail":"包含未经核实的建议"}`))
	reportRequest.Header.Set("Authorization", "Bearer "+reporterSession.Token)
	reportRequest.Header.Set("Content-Type", "application/json")
	server.Handler.ServeHTTP(reportRecorder, reportRequest)
	if reportRecorder.Code != http.StatusCreated {
		t.Fatalf("report status = %d body=%s", reportRecorder.Code, reportRecorder.Body.String())
	}
	reportID := decodeModerationReportID(t, reportRecorder.Body.Bytes())

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"admin-password"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	server.Handler.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("admin login status = %d body=%s", loginRecorder.Code, loginRecorder.Body.String())
	}
	csrf := cookieValue(loginRecorder.Result().Cookies(), "scf_admin_csrf")

	moderateRecorder := httptest.NewRecorder()
	moderateRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/reports/"+stringInt64(reportID)+"/moderate", bytes.NewBufferString(`{"action":"hide","note":"已确认违规"}`))
	moderateRequest.Header.Set("Content-Type", "application/json")
	moderateRequest.Header.Set("X-CSRF-Token", csrf)
	for _, cookie := range loginRecorder.Result().Cookies() {
		moderateRequest.AddCookie(cookie)
	}
	server.Handler.ServeHTTP(moderateRecorder, moderateRequest)
	if moderateRecorder.Code != http.StatusOK {
		t.Fatalf("moderate status = %d body=%s", moderateRecorder.Code, moderateRecorder.Body.String())
	}

	getPostRecorder := httptest.NewRecorder()
	getPostRequest := httptest.NewRequest(http.MethodGet, "/api/v1/posts/"+stringInt64(post.ID), nil)
	server.Handler.ServeHTTP(getPostRecorder, getPostRequest)
	if getPostRecorder.Code != http.StatusNotFound {
		t.Fatalf("hidden post status = %d, want %d", getPostRecorder.Code, http.StatusNotFound)
	}

	var auditCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'hide_post' AND record_id = ?`, "report-"+stringInt64(reportID)).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("audit count = %d, want 1", auditCount)
	}
}

func createModerationUser(t *testing.T, repository *sqliterepo.ForumRepository, email string, nickname string) domain.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("moderation-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := repository.CreateUser(context.Background(), domain.RegisterInput{
		Email: email, Nickname: nickname, Role: "student", Province: "广东", Grade: "高一",
	}, string(hash))
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func decodeModerationReportID(t *testing.T, body []byte) int64 {
	t.Helper()
	var envelope struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.ID == 0 {
		t.Fatalf("missing report id in body: %s", string(body))
	}
	return envelope.Data.ID
}

func cookieValue(cookies []*http.Cookie, name string) string {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

func stringInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}
