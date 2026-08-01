package httpserver

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/logx"
	sqliterepo "subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"
)

func TestPublicBetaSmokeUserAndModerationJourney(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := config.Config{
		AppEnv:                                 "local",
		SQLitePath:                             filepath.Join(tempDir, "public-beta-smoke.db"),
		MediaUploadDir:                         filepath.Join(tempDir, "uploads"),
		CORSAllowedOrigins:                     []string{"http://localhost:5173"},
		AdminEmail:                             "admin@example.com",
		AdminPassword:                          "admin-password",
		EmailVerificationTTLMinutes:            10,
		EmailVerificationCooldownSeconds:       1,
		EmailVerificationEmailHourlyLimit:      20,
		EmailVerificationIPHourlyLimit:         40,
		EmailVerificationMaxValidationAttempts: 5,
		HTTPMaxBodyBytes:                       1024 * 1024,
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
	app := httptest.NewServer(server.Handler)
	t.Cleanup(app.Close)
	runPublicBetaSmokeUserAndModerationJourney(t, app, db)
}

func runPublicBetaSmokeUserAndModerationJourney(t *testing.T, app *httptest.Server, db *sql.DB) {
	t.Helper()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	authorEmail := "smoke-author-" + suffix + "@example.com"
	reporterEmail := "smoke-reporter-" + suffix + "@example.com"
	authorName := "烟测作者" + suffix
	reporterName := "烟测同学" + suffix
	for _, endpoint := range []string{"/healthz", "/readyz"} {
		request, err := http.NewRequest(http.MethodHead, app.URL+endpoint, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := app.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("HEAD %s status = %d", endpoint, response.StatusCode)
		}
	}

	author := newSmokeClient(t, app.URL)
	reporter := newSmokeClient(t, app.URL)
	admin := newSmokeClient(t, app.URL)
	visitor := newSmokeClient(t, app.URL)
	unauthorizedBody, unauthorizedStatus := requestJSONRaw(t, visitor, http.MethodGet, app.URL+"/api/v1/notifications", nil, false, false)
	if unauthorizedStatus != http.StatusUnauthorized {
		t.Fatalf("anonymous notifications status = %d body=%s", unauthorizedStatus, unauthorizedBody)
	}
	assertErrorEnvelope(t, unauthorizedBody)

	registerSmokeUser(t, author, authorEmail, authorName)
	registerSmokeUser(t, reporter, reporterEmail, reporterName)
	reporterID := getSmokeJSON[int64](t, reporter, "/api/v1/me", "data.id")
	if reporterID == 0 {
		t.Fatal("reporter profile did not include user id")
	}
	postID := createSmokePost(t, author)
	commentID := postJSON[int64](t, reporter, app.URL+"/api/v1/posts/"+strconv.FormatInt(postID, 10)+"/comments", map[string]any{
		"content": "这条评论用于公测前真实后端冒烟。",
	}, "data.id")
	if commentID == 0 {
		t.Fatal("comment response did not include id")
	}
	postJSON[bool](t, reporter, app.URL+"/api/v1/posts/"+strconv.FormatInt(postID, 10)+"/like", nil, "data.active")
	postJSON[bool](t, reporter, app.URL+"/api/v1/posts/"+strconv.FormatInt(postID, 10)+"/favorite", nil, "data.active")
	postJSON[bool](t, reporter, app.URL+"/api/v1/authors/"+url.PathEscape(authorName)+"/follow", nil, "data.active")
	messageID := postJSON[int64](t, reporter, app.URL+"/api/v1/messages", map[string]any{
		"recipientName": authorName,
		"content":       "想请教一下广东选科经验。",
	}, "data.id")
	if messageID == 0 {
		t.Fatal("message response did not include id")
	}
	reportID := postJSON[int64](t, reporter, app.URL+"/api/v1/posts/"+strconv.FormatInt(postID, 10)+"/report", map[string]any{
		"reason": "疑似误导",
		"detail": "公测烟测举报内容。",
	}, "data.id")
	if reportID == 0 {
		t.Fatal("report response did not include id")
	}
	for _, targetURL := range []string{
		app.URL + "/api/v1/posts?sort=latest&limit=10",
		app.URL + "/api/v1/notifications?limit=10",
		app.URL + "/api/v1/messages?limit=10",
		app.URL + "/api/v1/messages/" + url.PathEscape(authorName) + "?limit=10",
	} {
		body, status := requestJSONRaw(t, reporter, http.MethodGet, targetURL, nil, true, false)
		if status != http.StatusOK {
			t.Fatalf("paginated GET %s status = %d body=%s", targetURL, status, body)
		}
		assertPaginatedSuccessEnvelope(t, body)
	}
	invalidCursorBody, invalidCursorStatus := requestJSONRaw(t, reporter, http.MethodGet, app.URL+"/api/v1/messages?cursor=not-a-cursor", nil, false, false)
	if invalidCursorStatus != http.StatusBadRequest {
		t.Fatalf("invalid conversation cursor status = %d body=%s", invalidCursorStatus, invalidCursorBody)
	}
	assertErrorEnvelope(t, invalidCursorBody)

	postJSON[string](t, admin, app.URL+"/api/v1/admin/login", map[string]any{
		"email":    "admin@example.com",
		"password": "admin-password",
	}, "data.email")
	postJSON[string](t, admin, app.URL+"/api/v1/admin/reports/"+strconv.FormatInt(reportID, 10)+"/moderate", map[string]any{
		"action": "hide",
		"note":   "公测烟测确认隐藏",
	}, "data.status")

	response, err := reporter.Get("/api/v1/posts/" + strconv.FormatInt(postID, 10))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("hidden post status = %d, want 404, body=%s", response.StatusCode, string(body))
	}
	assertAuditLogExists(t, db, "hide_post", "report-"+strconv.FormatInt(reportID, 10))

	postJSON[bool](t, admin, app.URL+"/api/v1/admin/users/"+strconv.FormatInt(reporterID, 10)+"/ban", map[string]any{
		"reason": "公测烟测封禁",
	}, "data.banned")
	assertStatus(t, reporter, "/api/v1/me", http.StatusUnauthorized)
	if status := requestJSONStatus(t, reporter, http.MethodPost, app.URL+"/api/v1/auth/login", map[string]any{
		"email": reporterEmail, "password": "public-beta-password",
	}, true); status != http.StatusUnauthorized {
		t.Fatalf("banned user login status = %d, want 401", status)
	}
	postJSON[bool](t, admin, app.URL+"/api/v1/admin/users/"+strconv.FormatInt(reporterID, 10)+"/restore", map[string]any{
		"reason": "公测烟测恢复",
	}, "data.restored")
	loginSmokeUser(t, reporter, reporterEmail, "public-beta-password")
	assertStatus(t, reporter, "/api/v1/me", http.StatusOK)
	assertAuditLogExists(t, db, "ban_user", "user-"+strconv.FormatInt(reporterID, 10))
	assertAuditLogExists(t, db, "restore_user", "user-"+strconv.FormatInt(reporterID, 10))
}

func TestPublicBetaSmokeAccountSessionControls(t *testing.T) {
	t.Parallel()

	app, closeServer := newPublicBetaSmokeServer(t, "account-session-smoke.db")
	defer closeServer()

	account := newSmokeClient(t, app.URL)
	registerSmokeUser(t, account, "smoke-account@example.com", "账号烟测")

	currentSessionID := getSmokeJSON[int64](t, account, "/api/v1/me/sessions", "data.0.id")
	if currentSessionID == 0 {
		t.Fatal("session list did not include the current session id")
	}
	currentSession := getSmokeJSON[bool](t, account, "/api/v1/me/sessions", "data.0.current")
	if !currentSession {
		t.Fatal("session list did not mark the current session")
	}

	forbidden := requestJSONStatus(t, account, http.MethodDelete, account.baseURL+"/api/v1/me/sessions/"+strconv.FormatInt(currentSessionID, 10), nil, false)
	if forbidden != http.StatusForbidden {
		t.Fatalf("revoke without csrf status = %d, want 403", forbidden)
	}
	deleteJSON[bool](t, account, account.baseURL+"/api/v1/me/sessions/"+strconv.FormatInt(currentSessionID, 10), nil, "data.revoked")
	assertStatus(t, account, "/api/v1/me", http.StatusUnauthorized)

	loginSmokeUser(t, account, "smoke-account@example.com", "public-beta-password")
	assertStatus(t, account, "/api/v1/me", http.StatusOK)
	postJSON[bool](t, account, account.baseURL+"/api/v1/auth/logout", nil, "data.signedOut")
	assertStatus(t, account, "/api/v1/me", http.StatusUnauthorized)

	loginSmokeUser(t, account, "smoke-account@example.com", "public-beta-password")
	deleteJSON[bool](t, account, account.baseURL+"/api/v1/me", map[string]any{
		"password": "public-beta-password",
	}, "data.deleted")
	assertStatus(t, account, "/api/v1/me", http.StatusUnauthorized)
	loginStatus := requestJSONStatus(t, account, http.MethodPost, account.baseURL+"/api/v1/auth/login", map[string]any{
		"email":    "smoke-account@example.com",
		"password": "public-beta-password",
	}, true)
	if loginStatus != http.StatusUnauthorized {
		t.Fatalf("deleted account login status = %d, want 401", loginStatus)
	}
}

func newPublicBetaSmokeServer(t *testing.T, databaseName string) (*httptest.Server, func()) {
	t.Helper()
	tempDir := t.TempDir()
	cfg := config.Config{
		AppEnv:                                 "local",
		SQLitePath:                             filepath.Join(tempDir, databaseName),
		MediaUploadDir:                         filepath.Join(tempDir, "uploads"),
		CORSAllowedOrigins:                     []string{"http://localhost:5173"},
		AdminEmail:                             "admin@example.com",
		AdminPassword:                          "admin-password",
		EmailVerificationTTLMinutes:            10,
		EmailVerificationCooldownSeconds:       1,
		EmailVerificationEmailHourlyLimit:      20,
		EmailVerificationIPHourlyLimit:         40,
		EmailVerificationMaxValidationAttempts: 5,
		HTTPMaxBodyBytes:                       1024 * 1024,
	}
	db, err := storage.NewSQLiteDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	forum := service.NewForumService(sqliterepo.NewForumRepository(db), cfg, nil)
	server := NewServer(cfg, logx.New(io.Discard, logx.LevelError), db, forum)
	app := httptest.NewServer(server.Handler)
	return app, func() {
		app.Close()
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	}
}

type smokeClient struct {
	baseURL string
	client  *http.Client
}

func newSmokeClient(t *testing.T, baseURL string) *smokeClient {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &smokeClient{baseURL: baseURL, client: &http.Client{Jar: jar}}
}

func (c *smokeClient) Get(path string) (*http.Response, error) {
	return c.client.Get(c.baseURL + path)
}

func registerSmokeUser(t *testing.T, client *smokeClient, email string, nickname string) {
	t.Helper()
	code := postJSON[string](t, client, "", map[string]any{
		"email": email,
	}, "data.debugCode")
	if code == "" {
		t.Fatal("local email verification did not return debugCode")
	}
	postJSON[int64](t, client, "", map[string]any{
		"email":            email,
		"password":         "public-beta-password",
		"verificationCode": code,
		"nickname":         nickname,
		"role":             "student",
		"province":         "广东",
		"grade":            "高一",
	}, "data.user.id")
}

func loginSmokeUser(t *testing.T, client *smokeClient, email string, password string) {
	t.Helper()
	userID := postJSON[int64](t, client, client.baseURL+"/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": password,
	}, "data.user.id")
	if userID == 0 {
		t.Fatal("login response did not include user id")
	}
}

func createSmokePost(t *testing.T, client *smokeClient) int64 {
	t.Helper()
	return postJSON[int64](t, client, "", map[string]any{
		"title":     "广东选科公测烟测帖子",
		"content":   "这是一条覆盖发布、互动、私信和审核链路的真实后端烟测内容。",
		"imageUrls": []string{},
		"tags":      []string{"广东选科"},
		"track":     "physics",
		"electives": []string{"chemistry", "biology"},
		"category":  "question",
		"grade":     "高一",
		"province":  "广东",
	}, "data.id")
}

func postJSON[T comparable](t *testing.T, client *smokeClient, targetURL string, payload map[string]any, path string) T {
	t.Helper()
	responseBody, _ := requestJSONRaw(t, client, http.MethodPost, resolveSmokePostURL(client, targetURL, path), payload, true, true)
	return decodePath[T](t, responseBody, path)
}

func deleteJSON[T comparable](t *testing.T, client *smokeClient, targetURL string, payload map[string]any, path string) T {
	t.Helper()
	responseBody, _ := requestJSONRaw(t, client, http.MethodDelete, targetURL, payload, true, true)
	return decodePath[T](t, responseBody, path)
}

func requestJSONStatus(t *testing.T, client *smokeClient, method string, targetURL string, payload map[string]any, includeCSRF bool) int {
	t.Helper()
	_, status := requestJSONRaw(t, client, method, targetURL, payload, false, includeCSRF)
	return status
}

func assertPaginatedSuccessEnvelope(t *testing.T, body []byte) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["data"]; !ok {
		t.Fatalf("success envelope missing data: %s", body)
	}
	meta, ok := payload["meta"].(map[string]any)
	if !ok || strings.TrimSpace(stringValue(meta["requestId"])) == "" {
		t.Fatalf("success envelope missing meta.requestId: %s", body)
	}
	if _, ok := meta["nextCursor"]; !ok {
		t.Fatalf("paginated envelope missing nextCursor: %s", body)
	}
	if _, ok := meta["hasMore"].(bool); !ok {
		t.Fatalf("paginated envelope missing boolean hasMore: %s", body)
	}
}

func assertErrorEnvelope(t *testing.T, body []byte) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	apiError, ok := payload["error"].(map[string]any)
	if !ok || strings.TrimSpace(stringValue(apiError["code"])) == "" || strings.TrimSpace(stringValue(apiError["requestId"])) == "" {
		t.Fatalf("error envelope missing code or requestId: %s", body)
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func resolveSmokePostURL(client *smokeClient, targetURL string, path string) string {
	if targetURL == "" {
		switch path {
		case "data.debugCode":
			targetURL = client.baseURL + "/api/v1/auth/email-verification-code"
		case "data.user.id":
			targetURL = client.baseURL + "/api/v1/auth/register"
		case "data.id":
			targetURL = client.baseURL + "/api/v1/posts"
		}
	}
	return targetURL
}

func requestJSONRaw(t *testing.T, client *smokeClient, method string, targetURL string, payload map[string]any, requireSuccess bool, includeCSRF bool) ([]byte, int) {
	t.Helper()
	if targetURL == "" {
		t.Fatal("targetURL is required")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(method, targetURL, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if includeCSRF {
		if csrf := csrfCookieValue(client, targetURL); csrf != "" {
			request.Header.Set("X-CSRF-Token", csrf)
		}
	}
	response, err := client.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)
	if requireSuccess && (response.StatusCode < 200 || response.StatusCode >= 300) {
		t.Fatalf("%s %s status=%d body=%s", method, targetURL, response.StatusCode, string(responseBody))
	}
	return responseBody, response.StatusCode
}

func getSmokeJSON[T comparable](t *testing.T, client *smokeClient, path string, jsonPath string) T {
	t.Helper()
	response, err := client.Get(path)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		t.Fatalf("GET %s status=%d body=%s", path, response.StatusCode, string(responseBody))
	}
	return decodePath[T](t, responseBody, jsonPath)
}

func assertStatus(t *testing.T, client *smokeClient, path string, want int) {
	t.Helper()
	response, err := client.Get(path)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != want {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("GET %s status=%d want=%d body=%s", path, response.StatusCode, want, string(body))
	}
}

func csrfCookieValue(client *smokeClient, targetURL string) string {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return ""
	}
	cookies := client.client.Jar.Cookies(parsed)
	for _, cookie := range cookies {
		if cookie.Name == "scf_csrf" || cookie.Name == "scf_admin_csrf" {
			return cookie.Value
		}
	}
	return ""
}

func decodePath[T comparable](t *testing.T, body []byte, path string) T {
	t.Helper()
	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("decode JSON: %v body=%s", err, string(body))
	}
	current := parsed
	for _, key := range strings.Split(path, ".") {
		if array, ok := current.([]any); ok {
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 || index >= len(array) {
				t.Fatalf("path %q array index %q missing in body=%s", path, key, string(body))
			}
			current = array[index]
			continue
		}
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("path %q missing at %q in body=%s", path, key, string(body))
		}
		current = object[key]
	}
	switch any(*new(T)).(type) {
	case int64:
		value, ok := current.(float64)
		if !ok {
			t.Fatalf("path %q is %T, want number", path, current)
		}
		return any(int64(value)).(T)
	case string:
		value, ok := current.(string)
		if !ok {
			t.Fatalf("path %q is %T, want string", path, current)
		}
		return any(value).(T)
	case bool:
		value, ok := current.(bool)
		if !ok {
			t.Fatalf("path %q is %T, want bool", path, current)
		}
		return any(value).(T)
	default:
		t.Fatalf("unsupported decode type for %q", path)
	}
	var zero T
	return zero
}

func assertAuditLogExists(t *testing.T, db *sql.DB, action string, recordID string) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = $1 AND record_id = $2`, action, recordID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("audit log count for %s/%s = %d, want 1", action, recordID, count)
	}
}
