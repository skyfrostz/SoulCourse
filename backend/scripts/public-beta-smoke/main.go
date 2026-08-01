package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"subject-choice-forum/backend/internal/config"
	httpserver "subject-choice-forum/backend/internal/http"
	"subject-choice-forum/backend/internal/http/middleware"
	"subject-choice-forum/backend/internal/logx"
	sqliterepo "subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"
)

type smokeConfig struct {
	baseURL       string
	adminEmail    string
	adminPassword string
	keepDB        bool
}

type smokeRunner struct {
	baseURL string
	client  *http.Client
	emails  *captureEmailSender
	cleanup func()
}

type captureEmailSender struct {
	mu    sync.Mutex
	codes map[string]string
}

func (s *captureEmailSender) Enabled() bool {
	return true
}

func (s *captureEmailSender) SendVerificationCode(ctx context.Context, to string, code string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[strings.ToLower(strings.TrimSpace(to))] = code
	return nil
}

func (s *captureEmailSender) codeFor(email string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	code, ok := s.codes[strings.ToLower(strings.TrimSpace(email))]
	return code, ok
}

type apiEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type postResponse struct {
	ID         int64  `json:"id"`
	AuthorName string `json:"authorName"`
}

type reportResponse struct {
	ID             int64  `json:"id"`
	Status         string `json:"status"`
	ResolutionNote string `json:"resolutionNote"`
}

type toggleResponse struct {
	Active bool `json:"active"`
	Count  int  `json:"count"`
}

type conversationResponse struct {
	User struct {
		Nickname string `json:"nickname"`
	} `json:"user"`
	LastMessage string `json:"lastMessage"`
}

type directMessageResponse struct {
	ID      int64  `json:"id"`
	Content string `json:"content"`
}

type reportsResponse struct {
	Reports []reportResponse `json:"reports"`
}

func main() {
	cfg := smokeConfig{}
	flag.StringVar(&cfg.baseURL, "base-url", os.Getenv("BASE_URL"), "existing backend base URL; empty starts an isolated in-process HTTP server")
	flag.StringVar(&cfg.adminEmail, "admin-email", envOr("SMOKE_ADMIN_EMAIL", "admin-smoke@example.com"), "admin email for smoke moderation")
	flag.StringVar(&cfg.adminPassword, "admin-password", envOr("SMOKE_ADMIN_PASSWORD", "admin-smoke-password"), "admin password for smoke moderation")
	flag.BoolVar(&cfg.keepDB, "keep-db", false, "keep the temporary SQLite directory after the run")
	flag.Parse()

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "public beta backend smoke failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("public beta backend smoke passed")
}

func run(smokeCfg smokeConfig) error {
	runner, err := newSmokeRunner(smokeCfg)
	if err != nil {
		return err
	}
	defer runner.cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := runner.expectOK(ctx, http.MethodGet, "/healthz", nil, nil); err != nil {
		return fmt.Errorf("healthz: %w", err)
	}
	if err := runner.expectOK(ctx, http.MethodGet, "/readyz", nil, nil); err != nil {
		return fmt.Errorf("readyz: %w", err)
	}

	stamp := time.Now().UnixNano()
	authorEmail := fmt.Sprintf("smoke-author-%d@example.com", stamp)
	reporterEmail := fmt.Sprintf("smoke-reporter-%d@example.com", stamp)
	authorName := fmt.Sprintf("Smoke作者%d", stamp%1000000)
	reporterName := fmt.Sprintf("Smoke同学%d", stamp%1000000)
	password := "smoke-password-123"

	if runner.emails == nil {
		return errors.New("external -base-url mode cannot run the registration smoke because the script cannot read production verification codes; run without -base-url for the supported isolated end-to-end smoke")
	}

	if err := runner.register(ctx, authorEmail, password, authorName); err != nil {
		return fmt.Errorf("register author: %w", err)
	}
	if err := runner.login(ctx, authorEmail, password); err != nil {
		return fmt.Errorf("login author: %w", err)
	}
	post, err := runner.createPost(ctx)
	if err != nil {
		return fmt.Errorf("create post: %w", err)
	}

	if err := runner.register(ctx, reporterEmail, password, reporterName); err != nil {
		return fmt.Errorf("register reporter: %w", err)
	}
	if err := runner.login(ctx, reporterEmail, password); err != nil {
		return fmt.Errorf("login reporter: %w", err)
	}
	if err := runner.createComment(ctx, post.ID); err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	if err := runner.toggle(ctx, fmt.Sprintf("/api/v1/posts/%d/like", post.ID)); err != nil {
		return fmt.Errorf("like post: %w", err)
	}
	if err := runner.toggle(ctx, fmt.Sprintf("/api/v1/posts/%d/favorite", post.ID)); err != nil {
		return fmt.Errorf("favorite post: %w", err)
	}
	if err := runner.toggle(ctx, "/api/v1/authors/"+url.PathEscape(post.AuthorName)+"/follow"); err != nil {
		return fmt.Errorf("follow author: %w", err)
	}
	if err := runner.sendMessage(ctx, post.AuthorName); err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	if err := runner.listConversations(ctx, post.AuthorName); err != nil {
		return fmt.Errorf("list conversations: %w", err)
	}
	if err := runner.listDirectMessages(ctx, post.AuthorName); err != nil {
		return fmt.Errorf("list direct messages: %w", err)
	}

	report, err := runner.reportPost(ctx, post.ID)
	if err != nil {
		return fmt.Errorf("report post: %w", err)
	}
	if err := runner.adminLogin(ctx, smokeCfg.adminEmail, smokeCfg.adminPassword); err != nil {
		return fmt.Errorf("admin login: %w", err)
	}
	if err := runner.listReports(ctx, report.ID); err != nil {
		return fmt.Errorf("list reports: %w", err)
	}
	if err := runner.moderateReport(ctx, report.ID); err != nil {
		return fmt.Errorf("moderate report: %w", err)
	}
	if err := runner.expectStatus(ctx, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", post.ID), nil, nil, http.StatusNotFound); err != nil {
		return fmt.Errorf("hidden post should return 404: %w", err)
	}
	return nil
}

func newSmokeRunner(smokeCfg smokeConfig) (*smokeRunner, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	runner := &smokeRunner{
		baseURL: strings.TrimRight(smokeCfg.baseURL, "/"),
		client:  &http.Client{Jar: jar, Timeout: 10 * time.Second},
		cleanup: func() {},
	}
	if runner.baseURL != "" {
		return runner, nil
	}

	tempDir, err := os.MkdirTemp("", "soulcourse-public-beta-smoke-*")
	if err != nil {
		return nil, err
	}
	cleanup := func() {
		if smokeCfg.keepDB {
			fmt.Println("kept smoke temp dir:", tempDir)
			return
		}
		_ = os.RemoveAll(tempDir)
	}

	cfg, err := config.Load()
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("load backend config: %w", err)
	}
	cfg.AppEnv = "test"
	cfg.SQLitePath = filepath.Join(tempDir, "smoke.db")
	cfg.MediaUploadDir = filepath.Join(tempDir, "uploads")
	cfg.FrontendDistDir = ""
	cfg.CORSAllowedOrigins = []string{"http://127.0.0.1"}
	cfg.AdminEmail = smokeCfg.adminEmail
	cfg.AdminPassword = smokeCfg.adminPassword
	cfg.AdminPasswordHash = ""

	db, err := storage.NewSQLiteDB(cfg)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("open smoke sqlite: %w", err)
	}
	emailSender := &captureEmailSender{codes: map[string]string{}}
	repo := sqliterepo.NewForumRepository(db)
	forum := service.NewForumService(repo, cfg, emailSender)
	server := httpserver.NewServer(cfg, logx.New(io.Discard, logx.LevelError), db, forum)
	testServer := httptest.NewServer(server.Handler)

	runner.baseURL = testServer.URL
	runner.emails = emailSender
	runner.cleanup = func() {
		testServer.Close()
		if err := db.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close smoke sqlite:", err)
		}
		cleanup()
	}
	return runner, nil
}

func (r *smokeRunner) register(ctx context.Context, email string, password string, nickname string) error {
	if err := r.expectOK(ctx, http.MethodPost, "/api/v1/auth/email-verification-code", map[string]any{"email": email}, nil); err != nil {
		return err
	}
	code, ok := r.emails.codeFor(email)
	if !ok {
		return fmt.Errorf("verification code was not captured for %s", email)
	}
	var session map[string]any
	return r.expectStatus(ctx, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email":            email,
		"password":         password,
		"verificationCode": code,
		"nickname":         nickname,
		"role":             "student",
		"province":         "广东",
		"grade":            "高一",
	}, &session, http.StatusCreated)
}

func (r *smokeRunner) login(ctx context.Context, email string, password string) error {
	return r.expectOK(ctx, http.MethodPost, "/api/v1/auth/login", map[string]any{"email": email, "password": password}, nil)
}

func (r *smokeRunner) adminLogin(ctx context.Context, email string, password string) error {
	return r.expectOK(ctx, http.MethodPost, "/api/v1/admin/login", map[string]any{"email": email, "password": password}, nil)
}

func (r *smokeRunner) createPost(ctx context.Context) (postResponse, error) {
	var post postResponse
	err := r.expectStatus(ctx, http.MethodPost, "/api/v1/posts", map[string]any{
		"title":     "公测 smoke 主链路帖子",
		"content":   "这是一条由真实 HTTP smoke 创建的帖子，用于验证注册、互动和审核主链路。",
		"track":     "physics",
		"electives": []string{"chemistry", "biology"},
		"category":  "question",
		"grade":     "高一",
		"province":  "广东",
		"tags":      []string{"smoke"},
	}, &post, http.StatusCreated)
	return post, err
}

func (r *smokeRunner) createComment(ctx context.Context, postID int64) error {
	var comment struct {
		ID      int64  `json:"id"`
		Content string `json:"content"`
	}
	if err := r.expectStatus(ctx, http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/comments", postID), map[string]any{
		"content": "真实 HTTP smoke 评论",
	}, &comment, http.StatusCreated); err != nil {
		return err
	}
	if comment.ID == 0 || !strings.Contains(comment.Content, "smoke") {
		return fmt.Errorf("comment response did not include expected content: %+v", comment)
	}
	return nil
}

func (r *smokeRunner) toggle(ctx context.Context, path string) error {
	var result toggleResponse
	if err := r.expectOK(ctx, http.MethodPost, path, map[string]any{}, &result); err != nil {
		return err
	}
	if !result.Active {
		return fmt.Errorf("%s did not activate the toggle: %+v", path, result)
	}
	return nil
}

func (r *smokeRunner) sendMessage(ctx context.Context, recipientName string) error {
	var message struct {
		ID            int64  `json:"id"`
		RecipientName string `json:"recipientName"`
		Content       string `json:"content"`
	}
	if err := r.expectStatus(ctx, http.MethodPost, "/api/v1/messages", map[string]any{
		"recipientName": recipientName,
		"content":       "真实 HTTP smoke 私信",
	}, &message, http.StatusCreated); err != nil {
		return err
	}
	if message.ID == 0 || message.RecipientName != recipientName {
		return fmt.Errorf("message response did not target %q: %+v", recipientName, message)
	}
	return nil
}

func (r *smokeRunner) listConversations(ctx context.Context, peerName string) error {
	var conversations []conversationResponse
	if err := r.expectOK(ctx, http.MethodGet, "/api/v1/messages", nil, &conversations); err != nil {
		return err
	}
	for _, item := range conversations {
		if item.User.Nickname == peerName && strings.Contains(item.LastMessage, "smoke") {
			return nil
		}
	}
	return fmt.Errorf("conversation with %q was not returned: %+v", peerName, conversations)
}

func (r *smokeRunner) listDirectMessages(ctx context.Context, peerName string) error {
	var messages []directMessageResponse
	if err := r.expectOK(ctx, http.MethodGet, "/api/v1/messages/"+url.PathEscape(peerName), nil, &messages); err != nil {
		return err
	}
	for _, item := range messages {
		if item.ID != 0 && strings.Contains(item.Content, "smoke") {
			return nil
		}
	}
	return fmt.Errorf("direct message with %q was not returned: %+v", peerName, messages)
}

func (r *smokeRunner) reportPost(ctx context.Context, postID int64) (reportResponse, error) {
	var report reportResponse
	err := r.expectStatus(ctx, http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/report", postID), map[string]any{
		"reason": "公测 smoke 举报",
		"detail": "验证举报到管理员审核闭环",
	}, &report, http.StatusCreated)
	if err == nil && (report.ID == 0 || report.Status != "open") {
		err = fmt.Errorf("report response is not open: %+v", report)
	}
	return report, err
}

func (r *smokeRunner) moderateReport(ctx context.Context, reportID int64) error {
	var report reportResponse
	if err := r.expectOK(ctx, http.MethodPost, fmt.Sprintf("/api/v1/admin/reports/%d/moderate", reportID), map[string]any{
		"action": "hide",
		"note":   "smoke confirmed",
	}, &report); err != nil {
		return err
	}
	if report.ID != reportID || report.Status != "actioned" || !strings.Contains(report.ResolutionNote, "smoke") {
		return fmt.Errorf("moderation response did not hide report %d: %+v", reportID, report)
	}
	return nil
}

func (r *smokeRunner) listReports(ctx context.Context, reportID int64) error {
	var response reportsResponse
	if err := r.expectOK(ctx, http.MethodGet, "/api/v1/admin/reports", nil, &response); err != nil {
		return err
	}
	for _, report := range response.Reports {
		if report.ID == reportID && report.Status == "open" {
			return nil
		}
	}
	return fmt.Errorf("open report %d was not returned: %+v", reportID, response.Reports)
}

func (r *smokeRunner) expectOK(ctx context.Context, method string, path string, body any, out any) error {
	return r.expectStatus(ctx, method, path, body, out, http.StatusOK)
}

func (r *smokeRunner) expectStatus(ctx context.Context, method string, path string, body any, out any, want int) error {
	req, err := r.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != want {
		return fmt.Errorf("%s %s status=%d want=%d body=%s", method, path, resp.StatusCode, want, strings.TrimSpace(string(raw)))
	}
	if out == nil {
		return nil
	}
	var envelope apiEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode envelope: %w body=%s", err, string(raw))
	}
	if envelope.Error != nil {
		return fmt.Errorf("api error %s: %s", envelope.Error.Code, envelope.Error.Message)
	}
	if len(envelope.Data) == 0 {
		return errors.New("response envelope missing data")
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("decode data: %w body=%s", err, string(raw))
	}
	return nil
}

func (r *smokeRunner) newRequest(ctx context.Context, method string, path string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if csrf := r.csrfToken(path); csrf != "" {
		req.Header.Set(middleware.CSRFHeaderName, csrf)
	}
	return req, nil
}

func (r *smokeRunner) csrfToken(path string) string {
	parsed, err := url.Parse(r.baseURL + path)
	if err != nil {
		return ""
	}
	preferred := middleware.CSRFCookieName
	if strings.Contains(path, "/admin/") {
		preferred = middleware.AdminCSRFCookieName
	}
	fallback := ""
	for _, cookie := range r.client.Jar.Cookies(parsed) {
		if cookie.Name == preferred {
			return cookie.Value
		}
		if fallback == "" && (cookie.Name == middleware.CSRFCookieName || cookie.Name == middleware.AdminCSRFCookieName) {
			fallback = cookie.Value
		}
	}
	return fallback
}

func envOr(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
