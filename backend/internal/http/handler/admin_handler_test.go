package handler

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/http/middleware"
	"subject-choice-forum/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

func TestAdminLoginHandlerErrorAndSuccessBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := middleware.NewAdminSessionStore(0)

	tests := []struct {
		name   string
		cfg    config.Config
		body   string
		status int
		code   string
	}{
		{name: "invalid payload", cfg: config.Config{}, body: `{}`, status: http.StatusBadRequest, code: "invalid_payload"},
		{name: "disabled", cfg: config.Config{}, body: `{"email":"admin@example.com","password":"password"}`, status: http.StatusServiceUnavailable, code: "admin_login_disabled"},
		{name: "bad credentials", cfg: config.Config{AdminEmail: "admin@example.com", AdminPassword: "password"}, body: `{"email":"admin@example.com","password":"wrong"}`, status: http.StatusUnauthorized, code: "invalid_admin_credentials"},
		{name: "invalid role", cfg: config.Config{AdminEmail: "admin@example.com", AdminPassword: "password", AdminRole: "owner"}, body: `{"email":"admin@example.com","password":"password"}`, status: http.StatusServiceUnavailable, code: "admin_role_invalid"},
		{name: "success", cfg: config.Config{AdminEmail: "admin@example.com", AdminPassword: "password", AdminRole: middleware.AdminRoleModerator}, body: `{"email":"admin@example.com","password":"password"}`, status: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(middleware.RequestID())
			handler := NewAdminHandler(tt.cfg, nil, nil, store)
			router.POST("/admin/login", handler.Login)
			recorder := performHandlerRequest(router, http.MethodPost, "/admin/login", tt.body)
			if recorder.Code != tt.status {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, tt.status, recorder.Body.String())
			}
			if tt.code != "" && !strings.Contains(recorder.Body.String(), `"code":"`+tt.code+`"`) {
				t.Fatalf("missing error code %q: %s", tt.code, recorder.Body.String())
			}
			if tt.status == http.StatusOK {
				if !strings.Contains(recorder.Body.String(), `"role":"moderator"`) || !hasCookie(recorder.Result().Cookies(), middleware.AdminSessionCookieName) {
					t.Fatalf("successful login omitted principal or cookie: %s", recorder.Body.String())
				}
			}
		})
	}
}

func TestAdminContentLifecycleFiltersUsersAndAuditsPrincipal(t *testing.T) {
	db, cfg := newAdminHandlerDB(t)
	_ = seedAdminTestUser(t, db)
	handler := NewAdminHandler(cfg, nil, db, middleware.NewAdminSessionStore(0))
	router := adminHandlerRouter(t, handler, middleware.AdminRoleSuperAdmin)

	invalid := performHandlerRequest(router, http.MethodPost, "/admin/content", `{}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid create status=%d body=%s", invalid.Code, invalid.Body.String())
	}

	created := performHandlerRequest(router, http.MethodPost, "/admin/content", `{
		"id":"policy-handler-test","module":"policies","title":" 广东政策测试 ",
		"type":"政策","status":"草稿","scope":"广东","owner":"考试院",
		"tags":["广东","政策"],"summary":"待复核内容","url":"https://example.edu/policy"
	}`)
	if created.Code != http.StatusOK || !strings.Contains(created.Body.String(), `"id":"policy-handler-test"`) {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}

	missingUpdate := performHandlerRequest(router, http.MethodPut, "/admin/content/missing", `{"module":"policies","title":"missing"}`)
	if missingUpdate.Code != http.StatusNotFound {
		t.Fatalf("missing update status=%d body=%s", missingUpdate.Code, missingUpdate.Body.String())
	}
	updated := performHandlerRequest(router, http.MethodPut, "/admin/content/policy-handler-test", `{"module":"policies","title":"更新政策","status":"待审核","scope":"广东"}`)
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), "更新政策") {
		t.Fatalf("update status=%d body=%s", updated.Code, updated.Body.String())
	}

	workflow := performHandlerRequest(router, http.MethodPost, "/admin/content/policy-handler-test/workflow", `{
		"action":"approve-content","actionLabel":"发布","nextStatus":"已上架","note":"复核完成"
	}`)
	if workflow.Code != http.StatusOK || !strings.Contains(workflow.Body.String(), `"status":"已上架"`) {
		t.Fatalf("workflow status=%d body=%s", workflow.Code, workflow.Body.String())
	}

	listed := performHandlerRequest(router, http.MethodGet, "/admin/content?module=policies&q=更新", "")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), "policy-handler-test") {
		t.Fatalf("list status=%d body=%s", listed.Code, listed.Body.String())
	}
	summary := performHandlerRequest(router, http.MethodGet, "/admin/content-summary", "")
	if summary.Code != http.StatusOK || !strings.Contains(summary.Body.String(), `"module":"policies"`) {
		t.Fatalf("summary status=%d body=%s", summary.Code, summary.Body.String())
	}

	audits := performHandlerRequest(router, http.MethodGet, "/admin/audit-logs", "")
	if audits.Code != http.StatusOK || !strings.Contains(audits.Body.String(), "owner@example.com (super_admin)") {
		t.Fatalf("audit status=%d body=%s", audits.Code, audits.Body.String())
	}

	deleted := performHandlerRequest(router, http.MethodDelete, "/admin/content/policy-handler-test", "")
	if deleted.Code != http.StatusOK || !strings.Contains(deleted.Body.String(), `"deleted":true`) {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	missingDelete := performHandlerRequest(router, http.MethodDelete, "/admin/content/missing", "")
	if missingDelete.Code != http.StatusNotFound {
		t.Fatalf("missing delete status=%d body=%s", missingDelete.Code, missingDelete.Body.String())
	}

	assertUserRecordVisibility(t, handler, middleware.AdminRoleContentEditor, false)
	assertUserRecordVisibility(t, handler, middleware.AdminRoleModerator, true)
}

func TestAdminContentWorkflowEnforcesServerStateMachine(t *testing.T) {
	db, cfg := newAdminHandlerDB(t)
	handler := NewAdminHandler(cfg, nil, db, middleware.NewAdminSessionStore(0))
	router := adminHandlerRouter(t, handler, middleware.AdminRoleSuperAdmin)

	created := performHandlerRequest(router, http.MethodPost, "/admin/content", `{
		"id":"workflow-state-machine","module":"policies","title":"状态机测试",
		"status":"草稿","payload":{"source":"trusted","workflow":[]}
	}`)
	if created.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}

	unknown := performHandlerRequest(router, http.MethodPost, "/admin/content/workflow-state-machine/workflow", `{
		"action":"force-publish","actionLabel":"强制发布","nextStatus":"已上架",
		"payload":{"source":"attacker"}
	}`)
	if unknown.Code != http.StatusBadRequest || !strings.Contains(unknown.Body.String(), `"code":"invalid_workflow_action"`) {
		t.Fatalf("unknown action status=%d body=%s", unknown.Code, unknown.Body.String())
	}

	wrongStatus := performHandlerRequest(router, http.MethodPost, "/admin/content/workflow-state-machine/workflow", `{
		"action":"submit-content-review","actionLabel":"提交审核","nextStatus":"已上架"
	}`)
	if wrongStatus.Code != http.StatusBadRequest || !strings.Contains(wrongStatus.Body.String(), `"code":"invalid_workflow_status"`) {
		t.Fatalf("wrong next status=%d body=%s", wrongStatus.Code, wrongStatus.Body.String())
	}

	accepted := performHandlerRequest(router, http.MethodPost, "/admin/content/workflow-state-machine/workflow", `{
		"action":"submit-content-review","actionLabel":"伪造标签","nextStatus":"待审核",
		"note":"送审","priority":"高","payload":{"source":"attacker","workflow":[{"action":"伪造记录"}]}
	}`)
	if accepted.Code != http.StatusOK {
		t.Fatalf("valid transition status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	body := accepted.Body.String()
	for _, expected := range []string{`"status":"待审核"`, `"priority":"中"`, `"source":"trusted"`, `"action":"提交审核"`, `"from":"草稿"`, `"to":"待审核"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	for _, forbidden := range []string{`"source":"attacker"`, `伪造标签`, `伪造记录`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response contains client-controlled workflow value %s: %s", forbidden, body)
		}
	}

	replayed := performHandlerRequest(router, http.MethodPost, "/admin/content/workflow-state-machine/workflow", `{
		"action":"submit-content-review","actionLabel":"再次提交","nextStatus":"待审核"
	}`)
	if replayed.Code != http.StatusConflict || !strings.Contains(replayed.Body.String(), `"code":"invalid_workflow_transition"`) {
		t.Fatalf("replayed transition status=%d body=%s", replayed.Code, replayed.Body.String())
	}
}

func TestAdminContentWorkflowAllowsFrontendTransitions(t *testing.T) {
	expected := map[string]string{
		"submit-content-review:草稿":   "待审核",
		"submit-content-review:退回修改": "待审核",
		"submit-content-review:下架":   "待审核",
		"approve-content:待审核":        "已上架",
		"send-to-review:待审核":         "需复核",
		"return-content:待审核":         "退回修改",
		"pass-review:需复核":            "已上架",
		"keep-review:需复核":            "需复核",
		"reject-after-review:需复核":    "下架",
		"start-review:已上架":           "需复核",
		"unpublish-content:已上架":      "下架",
	}
	if len(adminContentWorkflowTransitions) != len(expected) {
		t.Fatalf("transition count=%d want=%d", len(adminContentWorkflowTransitions), len(expected))
	}
	for key, target := range expected {
		transition, ok := adminContentWorkflowTransitions[key]
		if !ok {
			t.Errorf("missing frontend transition %s", key)
			continue
		}
		if transition.To != target {
			t.Errorf("transition %s target=%s want=%s", key, transition.To, target)
		}
	}
}

func TestAdminUserAndModerationHandlers(t *testing.T) {
	db, cfg := newAdminHandlerDB(t)
	userID := seedAdminTestUser(t, db)
	handler := NewAdminHandler(cfg, nil, db, middleware.NewAdminSessionStore(0))
	router := adminHandlerRouter(t, handler, middleware.AdminRoleSuperAdmin)

	badPassword := performHandlerRequest(router, http.MethodPut, "/admin/users/"+stringInt64ForHandlerTest(userID)+"/password", `{"password":"short"}`)
	if badPassword.Code != http.StatusBadRequest {
		t.Fatalf("short password status=%d body=%s", badPassword.Code, badPassword.Body.String())
	}
	reset := performHandlerRequest(router, http.MethodPut, "/admin/users/"+stringInt64ForHandlerTest(userID)+"/password", `{"password":"new-password-123"}`)
	if reset.Code != http.StatusOK {
		t.Fatalf("reset status=%d body=%s", reset.Code, reset.Body.String())
	}
	badUser := performHandlerRequest(router, http.MethodPost, "/admin/users/not-an-id/ban", `{}`)
	if badUser.Code != http.StatusBadRequest {
		t.Fatalf("invalid user status=%d body=%s", badUser.Code, badUser.Body.String())
	}
	ban := performHandlerRequest(router, http.MethodPost, "/admin/users/"+stringInt64ForHandlerTest(userID)+"/ban", `{"reason":"恶意骚扰"}`)
	if ban.Code != http.StatusOK {
		t.Fatalf("ban status=%d body=%s", ban.Code, ban.Body.String())
	}
	restore := performHandlerRequest(router, http.MethodPost, "/admin/users/"+stringInt64ForHandlerTest(userID)+"/restore", `{"reason":"复核通过"}`)
	if restore.Code != http.StatusOK {
		t.Fatalf("restore status=%d body=%s", restore.Code, restore.Body.String())
	}
	missingUser := performHandlerRequest(router, http.MethodPost, "/admin/users/999999/ban", `{}`)
	if missingUser.Code != http.StatusNotFound {
		t.Fatalf("missing user status=%d body=%s", missingUser.Code, missingUser.Body.String())
	}

	postID, reportID := seedAdminReport(t, db, userID)
	reports := performHandlerRequest(router, http.MethodGet, "/admin/reports?status=open", "")
	if reports.Code != http.StatusOK || !strings.Contains(reports.Body.String(), `"reason":"spam"`) {
		t.Fatalf("reports status=%d body=%s", reports.Code, reports.Body.String())
	}
	invalidReport := performHandlerRequest(router, http.MethodPost, "/admin/reports/not-an-id/moderate", `{"action":"hide"}`)
	if invalidReport.Code != http.StatusBadRequest {
		t.Fatalf("invalid report status=%d body=%s", invalidReport.Code, invalidReport.Body.String())
	}
	missingReport := performHandlerRequest(router, http.MethodPost, "/admin/reports/999999/moderate", `{"action":"hide"}`)
	if missingReport.Code != http.StatusNotFound {
		t.Fatalf("missing report status=%d body=%s", missingReport.Code, missingReport.Body.String())
	}
	hide := performHandlerRequest(router, http.MethodPost, "/admin/reports/"+stringInt64ForHandlerTest(reportID)+"/moderate", `{"action":"hide","note":"确认违规"}`)
	if hide.Code != http.StatusOK || !strings.Contains(hide.Body.String(), `"status":"actioned"`) {
		t.Fatalf("hide status=%d body=%s", hide.Code, hide.Body.String())
	}
	var deletedAt sql.NullString
	if err := db.QueryRow(`SELECT deleted_at FROM posts WHERE id = ?`, postID).Scan(&deletedAt); err != nil || !deletedAt.Valid {
		t.Fatalf("hidden post deletedAt=%#v err=%v", deletedAt, err)
	}
	restorePost := performHandlerRequest(router, http.MethodPost, "/admin/reports/"+stringInt64ForHandlerTest(reportID)+"/moderate", `{"action":"restore","note":"恢复展示"}`)
	if restorePost.Code != http.StatusOK {
		t.Fatalf("restore post status=%d body=%s", restorePost.Code, restorePost.Body.String())
	}
	if err := db.QueryRow(`SELECT deleted_at FROM posts WHERE id = ?`, postID).Scan(&deletedAt); err != nil || deletedAt.Valid {
		t.Fatalf("restored post deletedAt=%#v err=%v", deletedAt, err)
	}
	dismiss := performHandlerRequest(router, http.MethodPost, "/admin/reports/"+stringInt64ForHandlerTest(reportID)+"/moderate", `{"action":"dismiss","note":"证据不足"}`)
	if dismiss.Code != http.StatusOK || !strings.Contains(dismiss.Body.String(), `"status":"dismissed"`) {
		t.Fatalf("dismiss status=%d body=%s", dismiss.Code, dismiss.Body.String())
	}

	var actorCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_audit_logs WHERE actor = ? AND action IN ('reset_password','ban_user','restore_user','hide_post','restore_post','dismiss_report')`, "owner@example.com (super_admin)").Scan(&actorCount); err != nil {
		t.Fatal(err)
	}
	if actorCount != 6 {
		t.Fatalf("audited action count=%d want=6", actorCount)
	}
}

func TestAdminBanUserRollsBackWhenSessionRevocationFails(t *testing.T) {
	db, cfg := newAdminHandlerDB(t)
	userID := seedAdminTestUser(t, db)
	if _, err := db.Exec(`INSERT INTO auth_sessions (user_id, token_hash, created_at, expires_at) VALUES (?, 'test-token', ?, ?)`, userID, nowString(), time.Now().Add(time.Hour).Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER fail_session_revocation BEFORE UPDATE OF revoked_at ON auth_sessions BEGIN SELECT RAISE(ABORT, 'forced session failure'); END`); err != nil {
		t.Fatal(err)
	}
	handler := NewAdminHandler(cfg, nil, db, middleware.NewAdminSessionStore(0))
	router := adminHandlerRouter(t, handler, middleware.AdminRoleSuperAdmin)

	response := performHandlerRequest(router, http.MethodPost, "/admin/users/"+stringInt64ForHandlerTest(userID)+"/ban", `{"reason":"测试回滚"}`)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("ban status=%d body=%s", response.Code, response.Body.String())
	}
	var bannedAt sql.NullString
	if err := db.QueryRow(`SELECT banned_at FROM users WHERE id = ?`, userID).Scan(&bannedAt); err != nil || bannedAt.Valid {
		t.Fatalf("user ban was not rolled back: bannedAt=%#v err=%v", bannedAt, err)
	}
}

func TestAdminModerationRollsBackPostWhenReportUpdateFails(t *testing.T) {
	db, cfg := newAdminHandlerDB(t)
	userID := seedAdminTestUser(t, db)
	postID, reportID := seedAdminReport(t, db, userID)
	if _, err := db.Exec(`CREATE TRIGGER fail_report_resolution BEFORE UPDATE OF status ON content_reports BEGIN SELECT RAISE(ABORT, 'forced report failure'); END`); err != nil {
		t.Fatal(err)
	}
	handler := NewAdminHandler(cfg, nil, db, middleware.NewAdminSessionStore(0))
	router := adminHandlerRouter(t, handler, middleware.AdminRoleSuperAdmin)

	response := performHandlerRequest(router, http.MethodPost, "/admin/reports/"+stringInt64ForHandlerTest(reportID)+"/moderate", `{"action":"hide","note":"测试回滚"}`)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("moderation status=%d body=%s", response.Code, response.Body.String())
	}
	var deletedAt sql.NullString
	if err := db.QueryRow(`SELECT deleted_at FROM posts WHERE id = ?`, postID).Scan(&deletedAt); err != nil || deletedAt.Valid {
		t.Fatalf("post moderation was not rolled back: deletedAt=%#v err=%v", deletedAt, err)
	}
}

func TestSQLiteRealDataHandlersNeverClaimVerifiedCoverage(t *testing.T) {
	db, cfg := newAdminHandlerDB(t)
	handler := NewAdminHandler(cfg, nil, db, middleware.NewAdminSessionStore(0))
	router := gin.New()
	router.Use(middleware.RequestID())
	router.GET("/provinces", handler.ListProvinces)
	router.GET("/policies", handler.ListPolicies)
	router.GET("/requirements", handler.ListRequirements)
	router.GET("/sources/:id", handler.GetSource)

	for _, path := range []string{"/provinces", "/policies", "/requirements"} {
		recorder := performHandlerRequest(router, http.MethodGet, path, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
		if strings.Contains(recorder.Body.String(), `"coverageStatus":"verified"`) {
			t.Fatalf("SQLite fallback claimed verified coverage for %s: %s", path, recorder.Body.String())
		}
	}

	invalid := performHandlerRequest(router, http.MethodGet, "/sources/not-a-number", "")
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid source status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	missing := performHandlerRequest(router, http.MethodGet, "/sources/999999", "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing source status=%d body=%s", missing.Code, missing.Body.String())
	}

	sourceID := seedSQLiteContentSource(t, db)
	source := performHandlerRequest(router, http.MethodGet, "/sources/"+sourceID, "")
	if source.Code != http.StatusOK || !strings.Contains(source.Body.String(), `"coverageStatus":"unverified"`) || !strings.Contains(source.Body.String(), `"fileHash":""`) {
		t.Fatalf("source mapping status=%d body=%s", source.Code, source.Body.String())
	}
}

func newAdminHandlerDB(t *testing.T) (*sql.DB, config.Config) {
	t.Helper()
	tempDir := t.TempDir()
	cfg := config.Config{SQLitePath: filepath.Join(tempDir, "handler.db"), MediaUploadDir: filepath.Join(tempDir, "uploads")}
	db, err := storage.NewSQLiteDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, cfg
}

func adminHandlerRouter(t *testing.T, handler *AdminHandler, role string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	principal, err := middleware.NewAdminPrincipal("owner@example.com", role)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(func(c *gin.Context) {
		if !middleware.SetAdminPrincipal(c, principal) {
			t.Fatal("could not set admin principal")
		}
		c.Request = c.Request.WithContext(middleware.ContextWithAdminPrincipal(c.Request.Context(), principal))
		c.Next()
	})
	router.GET("/admin/content", handler.ListContent)
	router.POST("/admin/content", handler.CreateContent)
	router.PUT("/admin/content/:id", handler.UpdateContent)
	router.POST("/admin/content/:id/workflow", handler.WorkflowContent)
	router.DELETE("/admin/content/:id", handler.DeleteContent)
	router.GET("/admin/content-summary", handler.ContentSummary)
	router.GET("/admin/audit-logs", handler.AuditLogs)
	router.GET("/admin/reports", handler.ListReports)
	router.POST("/admin/reports/:id/moderate", handler.ModerateReport)
	router.PUT("/admin/users/:id/password", handler.ResetUserPassword)
	router.POST("/admin/users/:id/ban", handler.BanUser)
	router.POST("/admin/users/:id/restore", handler.RestoreUser)
	return router
}

func assertUserRecordVisibility(t *testing.T, handler *AdminHandler, role string, visible bool) {
	t.Helper()
	router := adminHandlerRouter(t, handler, role)
	recorder := performHandlerRequest(router, http.MethodGet, "/admin/content?module=users", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("role %s list status=%d body=%s", role, recorder.Code, recorder.Body.String())
	}
	hasUser := strings.Contains(recorder.Body.String(), "student@example.com")
	if hasUser != visible {
		t.Fatalf("role %s user visibility=%t want=%t body=%s", role, hasUser, visible, recorder.Body.String())
	}
}

func seedAdminTestUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	result, err := db.Exec(`INSERT INTO users (email, password_hash, nickname, role, province, grade, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"student@example.com", "hash", "测试学生", "student", "广东", "高一", nowString(), nowString())
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func seedAdminReport(t *testing.T, db *sql.DB, userID int64) (int64, int64) {
	t.Helper()
	result, err := db.Exec(`INSERT INTO posts (user_id, author_name, author_role, title, content, track, electives, category, grade, province, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, "测试学生", "student", "待审核帖子", "正文", "physics", `["chemistry","biology"]`, "question", "高一", "广东", nowString(), nowString())
	if err != nil {
		t.Fatal(err)
	}
	postID, _ := result.LastInsertId()
	result, err = db.Exec(`INSERT INTO content_reports (reporter_user_id, target_type, target_id, reason, detail, status, created_at, updated_at) VALUES (?, 'post', ?, 'spam', '重复推广', 'open', ?, ?)`,
		userID, postID, nowString(), nowString())
	if err != nil {
		t.Fatal(err)
	}
	reportID, _ := result.LastInsertId()
	return postID, reportID
}

func seedSQLiteContentSource(t *testing.T, db *sql.DB) string {
	t.Helper()
	result, err := db.Exec(`INSERT INTO posts (author_name, author_role, title, content, track, electives, category, grade, province, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"来源用户", "counselor", "来源测试", "测试", "physics", `["chemistry","biology"]`, "data", "高一", "广东", nowString(), nowString())
	if err != nil {
		t.Fatal(err)
	}
	postID, _ := result.LastInsertId()
	result, err = db.Exec(`INSERT INTO content_sources (post_id, source_platform, source_url, source_note_id, source_title, source_author, transformation_note, captured_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		postID, "official", "https://example.edu/source.pdf", "note-1", "官方来源", "考试院", "待复核", nowString())
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	return stringInt64ForHandlerTest(id)
}

func performHandlerRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func hasCookie(cookies []*http.Cookie, name string) bool {
	for _, cookie := range cookies {
		if cookie.Name == name && cookie.Value != "" {
			return true
		}
	}
	return false
}

func stringInt64ForHandlerTest(value int64) string {
	return strconv.FormatInt(value, 10)
}
