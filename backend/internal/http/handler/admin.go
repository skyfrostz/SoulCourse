package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	cfg     config.Config
	service *service.ForumService
	db      *pgxpool.Pool
}

func NewAdminHandler(cfg config.Config, forumService *service.ForumService, db *pgxpool.Pool) *AdminHandler {
	return &AdminHandler{cfg: cfg, service: forumService, db: db}
}

type AdminContentRecord struct {
	ID        string          `json:"id"`
	Module    string          `json:"module"`
	Title     string          `json:"title"`
	Type      string          `json:"type"`
	Status    string          `json:"status"`
	Scope     string          `json:"scope"`
	Owner     string          `json:"owner"`
	Tags      []string        `json:"tags"`
	Summary   string          `json:"summary"`
	URL       string          `json:"url"`
	Priority  string          `json:"priority"`
	SortOrder int             `json:"sortOrder"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type AdminContentInput struct {
	ID        string          `json:"id" binding:"omitempty,max=120"`
	Module    string          `json:"module" binding:"required,max=40"`
	Title     string          `json:"title" binding:"required,max=160"`
	Type      string          `json:"type" binding:"omitempty,max=80"`
	Status    string          `json:"status" binding:"omitempty,max=30"`
	Scope     string          `json:"scope" binding:"omitempty,max=80"`
	Owner     string          `json:"owner" binding:"omitempty,max=120"`
	Tags      []string        `json:"tags" binding:"omitempty,max=20,dive,max=40"`
	Summary   string          `json:"summary" binding:"omitempty,max=4000"`
	URL       string          `json:"url" binding:"omitempty,max=500"`
	Priority  string          `json:"priority" binding:"omitempty,max=20"`
	SortOrder int             `json:"sortOrder"`
	Payload   json.RawMessage `json:"payload"`
}

type AdminLoginInput struct {
	Email    string `json:"email" binding:"required,email,max=120"`
	Password string `json:"password" binding:"required,min=1,max=120"`
}

type AdminWorkflowInput struct {
	Action      string          `json:"action" binding:"required,max=80"`
	ActionLabel string          `json:"actionLabel" binding:"required,max=80"`
	NextStatus  string          `json:"nextStatus" binding:"required,max=30"`
	Note        string          `json:"note" binding:"omitempty,max=1200"`
	Priority    string          `json:"priority" binding:"omitempty,max=20"`
	Payload     json.RawMessage `json:"payload"`
}

type AdminContentSummary struct {
	Module    string `json:"module"`
	Total     int    `json:"total"`
	Published int    `json:"published"`
	Pending   int    `json:"pending"`
	Review    int    `json:"review"`
}

func (h *AdminHandler) Login(c *gin.Context) {
	var input AdminLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if h.cfg.AdminToken == "" || h.cfg.AdminEmail == "" || (h.cfg.AdminPassword == "" && h.cfg.AdminPasswordHash == "") {
		fail(c, http.StatusServiceUnavailable, "admin_login_disabled", "admin login is not configured")
		return
	}
	if email != h.cfg.AdminEmail || !h.validAdminPassword(input.Password) {
		fail(c, http.StatusUnauthorized, "invalid_admin_credentials", "invalid admin email or password")
		return
	}
	ok(c, envelope{
		"email": email,
		"token": h.cfg.AdminToken,
	})
}

func (h *AdminHandler) validAdminPassword(password string) bool {
	if h.cfg.AdminPasswordHash != "" {
		return bcrypt.CompareHashAndPassword([]byte(h.cfg.AdminPasswordHash), []byte(password)) == nil
	}
	return h.cfg.AdminPassword != "" && h.cfg.AdminPassword == password
}

func (h *AdminHandler) EmailConfig(c *gin.Context) {
	missing := make([]string, 0)
	if h.cfg.SMTPHost == "" {
		missing = append(missing, "SMTP_HOST")
	}
	if h.cfg.SMTPUsername == "" {
		missing = append(missing, "SMTP_USERNAME")
	}
	if h.cfg.SMTPPassword == "" {
		missing = append(missing, "SMTP_PASSWORD")
	}
	if h.cfg.SMTPFromEmail == "" {
		missing = append(missing, "SMTP_FROM_EMAIL")
	}
	ok(c, envelope{
		"enabled":                     h.cfg.SMTPEnabled(),
		"host":                        h.cfg.SMTPHost,
		"port":                        h.cfg.SMTPPort,
		"usernameConfigured":          h.cfg.SMTPUsername != "",
		"passwordConfigured":          h.cfg.SMTPPassword != "",
		"fromEmail":                   h.cfg.SMTPFromEmail,
		"fromName":                    h.cfg.SMTPFromName,
		"useTLS":                      h.cfg.SMTPUseTLS,
		"startTLS":                    h.cfg.SMTPStartTLS,
		"emailVerificationTTLMinutes": h.cfg.EmailVerificationTTLMinutes,
		"missing":                     missing,
	})
}

func (h *AdminHandler) SendTestEmail(c *gin.Context) {
	var input domain.EmailVerificationCodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	result, err := h.service.SendEmailVerificationCode(c.Request.Context(), input)
	if err != nil {
		fail(c, http.StatusInternalServerError, "email_send_failed", "could not send test email")
		return
	}
	ok(c, result)
}

func (h *AdminHandler) ListContent(c *gin.Context) {
	module := strings.TrimSpace(c.Query("module"))
	status := strings.TrimSpace(c.Query("status"))
	keyword := strings.TrimSpace(c.Query("q"))
	args := make([]any, 0, 4)
	clauses := []string{"deleted_at IS NULL"}

	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if module != "" {
		clauses = append(clauses, "module = "+addArg(module))
	}
	if status != "" && status != "全部" {
		clauses = append(clauses, "status = "+addArg(status))
	}
	if keyword != "" {
		arg := addArg("%" + keyword + "%")
		clauses = append(clauses, "(title ILIKE "+arg+" OR summary ILIKE "+arg+" OR owner ILIKE "+arg+" OR scope ILIKE "+arg+" OR array_to_string(tags, ',') ILIKE "+arg+")")
	}

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT id, module, title, content_type, status, scope, owner, tags, summary, url,
		       priority, sort_order, payload, created_at, updated_at
		FROM admin_content_records
		WHERE `+strings.Join(clauses, " AND ")+`
		ORDER BY module ASC, sort_order ASC, updated_at DESC
	`, args...)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_query_failed", "could not load admin content")
		return
	}
	defer rows.Close()

	records := make([]AdminContentRecord, 0)
	for rows.Next() {
		record, err := scanAdminContent(rows)
		if err != nil {
			fail(c, http.StatusInternalServerError, "content_scan_failed", "could not read admin content")
			return
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		fail(c, http.StatusInternalServerError, "content_rows_failed", "could not read admin content")
		return
	}

	ok(c, envelope{"records": records})
}

func (h *AdminHandler) ListPublishedContent(c *gin.Context) {
	module := strings.TrimSpace(c.Query("module"))
	keyword := strings.TrimSpace(c.Query("q"))
	args := make([]any, 0, 3)
	clauses := []string{"deleted_at IS NULL", "status IN ('已上架', '正常')", "module <> 'users'"}

	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if module == "users" {
		ok(c, envelope{"records": []AdminContentRecord{}})
		return
	}
	if module != "" {
		clauses = append(clauses, "module = "+addArg(module))
	}
	if keyword != "" {
		arg := addArg("%" + keyword + "%")
		clauses = append(clauses, "(title ILIKE "+arg+" OR summary ILIKE "+arg+" OR owner ILIKE "+arg+" OR scope ILIKE "+arg+" OR array_to_string(tags, ',') ILIKE "+arg+")")
	}

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT id, module, title, content_type, status, scope, owner, tags, summary, url,
		       priority, sort_order, payload, created_at, updated_at
		FROM admin_content_records
		WHERE `+strings.Join(clauses, " AND ")+`
		ORDER BY module ASC, sort_order ASC, updated_at DESC
	`, args...)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_query_failed", "could not load content")
		return
	}
	defer rows.Close()

	records := make([]AdminContentRecord, 0)
	for rows.Next() {
		record, err := scanAdminContent(rows)
		if err != nil {
			fail(c, http.StatusInternalServerError, "content_scan_failed", "could not read content")
			return
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		fail(c, http.StatusInternalServerError, "content_rows_failed", "could not read content")
		return
	}

	ok(c, envelope{"records": records})
}

func (h *AdminHandler) ContentSummary(c *gin.Context) {
	rows, err := h.db.Query(c.Request.Context(), `
		SELECT module,
		       COUNT(*)::integer AS total,
		       COUNT(*) FILTER (WHERE status IN ('已上架', '正常'))::integer AS published,
		       COUNT(*) FILTER (WHERE status IN ('待审核', '认证中'))::integer AS pending,
		       COUNT(*) FILTER (WHERE status = '需复核')::integer AS review
		FROM admin_content_records
		WHERE deleted_at IS NULL
		GROUP BY module
		ORDER BY module
	`)
	if err != nil {
		fail(c, http.StatusInternalServerError, "summary_query_failed", "could not load admin summary")
		return
	}
	defer rows.Close()

	summary := make([]AdminContentSummary, 0)
	for rows.Next() {
		var item AdminContentSummary
		if err := rows.Scan(&item.Module, &item.Total, &item.Published, &item.Pending, &item.Review); err != nil {
			fail(c, http.StatusInternalServerError, "summary_scan_failed", "could not read admin summary")
			return
		}
		summary = append(summary, item)
	}
	ok(c, envelope{"summary": summary})
}

func (h *AdminHandler) CreateContent(c *gin.Context) {
	var input AdminContentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	normalizeAdminContentInput(&input)
	if input.ID == "" {
		input.ID = input.Module + "-" + time.Now().Format("20060102150405.000000000")
		input.ID = strings.ReplaceAll(input.ID, ".", "")
	}

	record, err := h.upsertContent(c, input, false)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_create_failed", "could not create admin content")
		return
	}
	record, err = h.syncContentRecord(c, record)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_sync_failed", "could not sync admin content")
		return
	}
	h.logAudit(c, "create", record.ID, record.Module, "新建后台内容："+record.Title)
	ok(c, record)
}

func (h *AdminHandler) UpdateContent(c *gin.Context) {
	var input AdminContentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	input.ID = c.Param("id")
	normalizeAdminContentInput(&input)

	record, err := h.upsertContent(c, input, true)
	if err != nil {
		if err == pgx.ErrNoRows {
			fail(c, http.StatusNotFound, "content_not_found", "admin content record was not found")
			return
		}
		fail(c, http.StatusInternalServerError, "content_update_failed", "could not update admin content")
		return
	}
	record, err = h.syncContentRecord(c, record)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_sync_failed", "could not sync admin content")
		return
	}
	h.logAudit(c, "update", record.ID, record.Module, "保存后台内容："+record.Title)
	ok(c, record)
}

func (h *AdminHandler) WorkflowContent(c *gin.Context) {
	var input AdminWorkflowInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	input.Action = strings.TrimSpace(input.Action)
	input.ActionLabel = strings.TrimSpace(input.ActionLabel)
	input.NextStatus = strings.TrimSpace(input.NextStatus)
	input.Note = strings.TrimSpace(input.Note)
	input.Priority = strings.TrimSpace(input.Priority)
	if input.Priority == "" {
		input.Priority = "常规"
	}
	if len(input.Payload) == 0 || !json.Valid(input.Payload) {
		input.Payload = json.RawMessage(`{}`)
	}

	record, err := scanAdminContent(h.db.QueryRow(c.Request.Context(), `
		UPDATE admin_content_records
		SET status = $2, priority = $3, payload = $4, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, module, title, content_type, status, scope, owner, tags, summary, url,
		          priority, sort_order, payload, created_at, updated_at
	`, c.Param("id"), input.NextStatus, input.Priority, []byte(input.Payload)))
	if err != nil {
		if err == pgx.ErrNoRows {
			fail(c, http.StatusNotFound, "content_not_found", "admin content record was not found")
			return
		}
		fail(c, http.StatusInternalServerError, "workflow_update_failed", "could not update admin workflow")
		return
	}
	record, err = h.syncContentRecord(c, record)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_sync_failed", "could not sync admin workflow")
		return
	}

	detail := input.ActionLabel + "：" + record.Title
	if input.Note != "" {
		detail += "；意见：" + input.Note
	}
	h.logAudit(c, "workflow:"+input.Action, record.ID, record.Module, detail)
	ok(c, record)
}

func (h *AdminHandler) DeleteContent(c *gin.Context) {
	id := c.Param("id")
	var module string
	var title string
	var payload []byte
	err := h.db.QueryRow(c.Request.Context(), `
		UPDATE admin_content_records
		SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING module, title, payload
	`, id).Scan(&module, &title, &payload)
	if err != nil {
		if err == pgx.ErrNoRows {
			fail(c, http.StatusNotFound, "content_not_found", "admin content record was not found")
			return
		}
		fail(c, http.StatusInternalServerError, "content_delete_failed", "could not delete admin content")
		return
	}
	if module == "posts" {
		if err := h.softDeleteSyncedPost(c, payload); err != nil {
			fail(c, http.StatusInternalServerError, "content_sync_failed", "could not remove synced post")
			return
		}
	}
	h.logAudit(c, "delete", id, module, "删除后台内容："+title)
	ok(c, envelope{"deleted": true})
}

type syncedPostPayload struct {
	PostID    int64
	Content   string
	Track     string
	Electives []string
	Category  string
	Grade     string
	Province  string
	ImageURLs []string
}

func (h *AdminHandler) syncContentRecord(c *gin.Context, record AdminContentRecord) (AdminContentRecord, error) {
	if record.Module != "posts" {
		return record, nil
	}

	payloadMap := decodePayloadMap(record.Payload)
	postPayload := buildSyncedPostPayload(record, payloadMap)
	postID := postPayload.PostID

	if postID == 0 {
		_ = h.db.QueryRow(c.Request.Context(), `
			SELECT id
			FROM posts
			WHERE title = $1 AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1
		`, record.Title).Scan(&postID)
	}

	if postID > 0 {
		err := h.db.QueryRow(c.Request.Context(), `
			UPDATE posts
			SET author_name = $2,
			    author_role = $3,
			    title = $4,
			    content = $5,
			    image_urls = $6,
			    tags = $7,
			    track = $8,
			    electives = $9,
			    category = $10,
			    grade = $11,
			    province = $12,
			    deleted_at = NULL,
			    updated_at = now()
			WHERE id = $1 AND deleted_at IS NULL
			RETURNING id
		`, postID, record.Owner, inferAuthorRole(record), record.Title, postPayload.Content, postPayload.ImageURLs, record.Tags, postPayload.Track, postPayload.Electives, postPayload.Category, postPayload.Grade, postPayload.Province).Scan(&postID)
		if err != nil && err != pgx.ErrNoRows {
			return AdminContentRecord{}, err
		}
		if err == pgx.ErrNoRows {
			postID = 0
		}
	}

	if postID == 0 {
		err := h.db.QueryRow(c.Request.Context(), `
			INSERT INTO posts (author_name, author_role, title, content, image_urls, tags, track, electives, category, grade, province)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id
		`, record.Owner, inferAuthorRole(record), record.Title, postPayload.Content, postPayload.ImageURLs, record.Tags, postPayload.Track, postPayload.Electives, postPayload.Category, postPayload.Grade, postPayload.Province).Scan(&postID)
		if err != nil {
			return AdminContentRecord{}, err
		}
	}

	payloadMap["postId"] = strconv.FormatInt(postID, 10)
	payloadMap["content"] = postPayload.Content
	payloadMap["track"] = postPayload.Track
	payloadMap["electives"] = postPayload.Electives
	payloadMap["category"] = postPayload.Category
	payloadMap["grade"] = postPayload.Grade
	payloadMap["province"] = postPayload.Province
	payloadMap["imageUrls"] = postPayload.ImageURLs

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return AdminContentRecord{}, err
	}
	return scanAdminContent(h.db.QueryRow(c.Request.Context(), `
		UPDATE admin_content_records
		SET payload = $2,
		    url = $3,
		    updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, module, title, content_type, status, scope, owner, tags, summary, url,
		          priority, sort_order, payload, created_at, updated_at
	`, record.ID, payloadBytes, fmt.Sprintf("/posts/%d", postID)))
}

func (h *AdminHandler) softDeleteSyncedPost(c *gin.Context, payload []byte) error {
	payloadMap := decodePayloadMap(payload)
	postID := payloadInt64(payloadMap, "postId")
	if postID == 0 {
		return nil
	}
	_, err := h.db.Exec(c.Request.Context(), `
		UPDATE posts
		SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`, postID)
	return err
}

func decodePayloadMap(raw []byte) map[string]any {
	payload := map[string]any{}
	if len(raw) == 0 {
		return payload
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return map[string]any{}
	}
	return payload
}

func buildSyncedPostPayload(record AdminContentRecord, payload map[string]any) syncedPostPayload {
	content := payloadString(payload, "content")
	if content == "" {
		content = strings.TrimSpace(record.Summary)
	}
	if content == "" {
		content = "这条内容来自后台内容库，请补充正文、来源和审核说明。"
	}

	return syncedPostPayload{
		PostID:    payloadInt64(payload, "postId"),
		Content:   content,
		Track:     normalizeTrack(payloadString(payload, "track"), record),
		Electives: normalizeElectives(payloadStringSlice(payload, "electives"), record),
		Category:  normalizeCategory(payloadString(payload, "category"), record),
		Grade:     defaultString(payloadString(payload, "grade"), "高一"),
		Province:  normalizeProvince(defaultString(payloadString(payload, "province"), record.Scope)),
		ImageURLs: payloadStringSlice(payload, "imageUrls"),
	}
}

func payloadString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return fmt.Sprintf("%g", typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func payloadInt64(payload map[string]any, key string) int64 {
	value := payloadString(payload, key)
	if value == "" {
		return 0
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func payloadStringSlice(payload map[string]any, key string) []string {
	value, ok := payload[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return cleanStringSlice(typed)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				values = append(values, text)
			}
		}
		return cleanStringSlice(values)
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return cleanStringSlice(strings.Split(typed, ","))
	default:
		return nil
	}
}

func cleanStringSlice(values []string) []string {
	cleaned := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func inferAuthorRole(record AdminContentRecord) string {
	text := record.Type + " " + record.Owner + " " + strings.Join(record.Tags, " ")
	switch {
	case strings.Contains(text, "老师") || strings.Contains(text, "教师"):
		return "teacher"
	case strings.Contains(text, "规划") || strings.Contains(text, "顾问") || strings.Contains(text, "研究") || strings.Contains(text, "数据"):
		return "counselor"
	case strings.Contains(text, "家长") || strings.Contains(text, "妈妈") || strings.Contains(text, "爸爸"):
		return "parent"
	default:
		return "student"
	}
}

func normalizeTrack(value string, record AdminContentRecord) string {
	if value == "physics" || value == "history" {
		return value
	}
	text := record.Title + " " + record.Summary + " " + strings.Join(record.Tags, " ")
	if strings.Contains(text, "历史") || strings.Contains(text, "史政") || strings.Contains(text, "文科") {
		return "history"
	}
	return "physics"
}

func normalizeElectives(values []string, record AdminContentRecord) []string {
	allowed := map[string]bool{"chemistry": true, "biology": true, "politics": true, "geography": true}
	cleaned := make([]string, 0, 2)
	seen := map[string]bool{}
	for _, value := range values {
		if allowed[value] && !seen[value] {
			seen[value] = true
			cleaned = append(cleaned, value)
		}
	}
	if len(cleaned) == 2 {
		return cleaned
	}

	text := record.Title + " " + record.Summary + " " + strings.Join(record.Tags, " ")
	switch {
	case strings.Contains(text, "物化政"):
		return []string{"chemistry", "politics"}
	case strings.Contains(text, "物化地"):
		return []string{"chemistry", "geography"}
	case strings.Contains(text, "物生地"):
		return []string{"biology", "geography"}
	case strings.Contains(text, "史政地"):
		return []string{"politics", "geography"}
	case strings.Contains(text, "史化生"):
		return []string{"chemistry", "biology"}
	default:
		return []string{"chemistry", "biology"}
	}
}

func normalizeCategory(value string, record AdminContentRecord) string {
	if value == "experience" || value == "question" || value == "data" {
		return value
	}
	text := record.Type + " " + record.Title + " " + strings.Join(record.Tags, " ")
	switch {
	case strings.Contains(text, "数据") || strings.Contains(text, "政策") || strings.Contains(text, "要求"):
		return "data"
	case strings.Contains(text, "提问") || strings.Contains(text, "问") || strings.Contains(text, "纠结"):
		return "question"
	default:
		return "experience"
	}
}

func normalizeProvince(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "全站" || value == "首页" {
		return "全国"
	}
	return value
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (h *AdminHandler) AuditLogs(c *gin.Context) {
	rows, err := h.db.Query(c.Request.Context(), `
		SELECT action, COALESCE(record_id, ''), COALESCE(module, ''), detail, actor, created_at
		FROM admin_audit_logs
		ORDER BY created_at DESC
		LIMIT 30
	`)
	if err != nil {
		fail(c, http.StatusInternalServerError, "audit_query_failed", "could not load audit logs")
		return
	}
	defer rows.Close()

	logs := make([]envelope, 0)
	for rows.Next() {
		var action, recordID, module, detail, actor string
		var createdAt time.Time
		if err := rows.Scan(&action, &recordID, &module, &detail, &actor, &createdAt); err != nil {
			fail(c, http.StatusInternalServerError, "audit_scan_failed", "could not read audit logs")
			return
		}
		logs = append(logs, envelope{
			"action":    action,
			"recordId":  recordID,
			"module":    module,
			"detail":    detail,
			"actor":     actor,
			"createdAt": createdAt,
		})
	}
	ok(c, envelope{"logs": logs})
}

func (h *AdminHandler) upsertContent(c *gin.Context, input AdminContentInput, updateOnly bool) (AdminContentRecord, error) {
	if updateOnly {
		return scanAdminContent(h.db.QueryRow(c.Request.Context(), `
			UPDATE admin_content_records
			SET module = $2, title = $3, content_type = $4, status = $5, scope = $6, owner = $7,
			    tags = $8, summary = $9, url = $10, priority = $11, sort_order = $12,
			    payload = $13, updated_at = now()
			WHERE id = $1 AND deleted_at IS NULL
			RETURNING id, module, title, content_type, status, scope, owner, tags, summary, url,
			          priority, sort_order, payload, created_at, updated_at
		`, input.ID, input.Module, input.Title, input.Type, input.Status, input.Scope, input.Owner, input.Tags, input.Summary, input.URL, input.Priority, input.SortOrder, []byte(input.Payload)))
	}

	return scanAdminContent(h.db.QueryRow(c.Request.Context(), `
		INSERT INTO admin_content_records (id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE
		SET module = EXCLUDED.module,
		    title = EXCLUDED.title,
		    content_type = EXCLUDED.content_type,
		    status = EXCLUDED.status,
		    scope = EXCLUDED.scope,
		    owner = EXCLUDED.owner,
		    tags = EXCLUDED.tags,
		    summary = EXCLUDED.summary,
		    url = EXCLUDED.url,
		    priority = EXCLUDED.priority,
		    sort_order = EXCLUDED.sort_order,
		    payload = EXCLUDED.payload,
		    deleted_at = NULL,
		    updated_at = now()
		RETURNING id, module, title, content_type, status, scope, owner, tags, summary, url,
		          priority, sort_order, payload, created_at, updated_at
	`, input.ID, input.Module, input.Title, input.Type, input.Status, input.Scope, input.Owner, input.Tags, input.Summary, input.URL, input.Priority, input.SortOrder, []byte(input.Payload)))
}

func normalizeAdminContentInput(input *AdminContentInput) {
	input.ID = strings.TrimSpace(input.ID)
	input.Module = strings.TrimSpace(input.Module)
	input.Title = strings.TrimSpace(input.Title)
	input.Type = strings.TrimSpace(input.Type)
	input.Status = strings.TrimSpace(input.Status)
	input.Scope = strings.TrimSpace(input.Scope)
	input.Owner = strings.TrimSpace(input.Owner)
	input.URL = strings.TrimSpace(input.URL)
	input.Priority = strings.TrimSpace(input.Priority)
	if input.Type == "" {
		input.Type = "未分类"
	}
	if input.Status == "" {
		input.Status = "草稿"
	}
	if input.Scope == "" {
		input.Scope = "全国"
	}
	if input.Owner == "" {
		input.Owner = "内容运营"
	}
	if input.Priority == "" {
		input.Priority = "常规"
	}
	if len(input.Payload) == 0 || !json.Valid(input.Payload) {
		input.Payload = json.RawMessage(`{}`)
	}
	tags := make([]string, 0, len(input.Tags))
	for _, tag := range input.Tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	input.Tags = tags
}

type adminContentScanner interface {
	Scan(dest ...any) error
}

func scanAdminContent(scanner adminContentScanner) (AdminContentRecord, error) {
	var record AdminContentRecord
	var payload []byte
	if err := scanner.Scan(
		&record.ID,
		&record.Module,
		&record.Title,
		&record.Type,
		&record.Status,
		&record.Scope,
		&record.Owner,
		&record.Tags,
		&record.Summary,
		&record.URL,
		&record.Priority,
		&record.SortOrder,
		&payload,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return AdminContentRecord{}, err
	}
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	record.Payload = json.RawMessage(payload)
	return record, nil
}

func (h *AdminHandler) logAudit(c *gin.Context, action string, recordID string, module string, detail string) {
	_, _ = h.db.Exec(c.Request.Context(), `
		INSERT INTO admin_audit_logs (action, record_id, module, detail)
		VALUES ($1, $2, $3, $4)
	`, action, recordID, module, detail)
}
