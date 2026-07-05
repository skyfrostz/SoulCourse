package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	cfg     config.Config
	service *service.ForumService
	db      *sql.DB
}

func NewAdminHandler(cfg config.Config, forumService *service.ForumService, db *sql.DB) *AdminHandler {
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

const maxAdminImageUploadBytes int64 = 8 * 1024 * 1024

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

func (h *AdminHandler) UploadImage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAdminImageUploadBytes+1024*1024)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "missing_file", "please choose an image file")
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxAdminImageUploadBytes {
		fail(c, http.StatusBadRequest, "file_too_large", "image must be smaller than 8MB")
		return
	}

	contentType, ext, err := detectImageUpload(fileHeader)
	if err != nil {
		fail(c, http.StatusBadRequest, "unsupported_file_type", err.Error())
		return
	}

	dateDir := time.Now().UTC().Format("20060102")
	fileName := randomHex(16) + ext
	targetDir := filepath.Join(h.cfg.MediaUploadDir, "images", dateDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fail(c, http.StatusInternalServerError, "upload_dir_failed", "could not prepare upload directory")
		return
	}

	targetPath := filepath.Join(targetDir, fileName)
	if err := saveUploadedFile(fileHeader, targetPath); err != nil {
		fail(c, http.StatusInternalServerError, "upload_save_failed", "could not save uploaded image")
		return
	}

	ok(c, envelope{
		"url":         h.cfg.RoutePath("/uploads/images/" + dateDir + "/" + fileName),
		"contentType": contentType,
		"size":        fileHeader.Size,
		"name":        fileHeader.Filename,
	})
}

func (h *AdminHandler) ListContent(c *gin.Context) {
	records, err := h.listContentRecords(c.Request.Context(), false)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_query_failed", "could not load admin content")
		return
	}
	module := strings.TrimSpace(c.Query("module"))
	status := strings.TrimSpace(c.Query("status"))
	keyword := strings.TrimSpace(c.Query("q"))
	records = filterAdminRecords(records, module, status, keyword)
	ok(c, envelope{"records": records})
}

func (h *AdminHandler) ListPublishedContent(c *gin.Context) {
	module := strings.TrimSpace(c.Query("module"))
	if module == "users" {
		ok(c, envelope{"records": []AdminContentRecord{}})
		return
	}
	records, err := h.listContentRecords(c.Request.Context(), true)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_query_failed", "could not load content")
		return
	}
	keyword := strings.TrimSpace(c.Query("q"))
	records = filterAdminRecords(records, module, "", keyword)
	ok(c, envelope{"records": records})
}

func (h *AdminHandler) ContentSummary(c *gin.Context) {
	records, err := h.listContentRecords(c.Request.Context(), false)
	if err != nil {
		fail(c, http.StatusInternalServerError, "summary_query_failed", "could not load admin summary")
		return
	}
	index := map[string]*AdminContentSummary{}
	for _, record := range records {
		item := index[record.Module]
		if item == nil {
			item = &AdminContentSummary{Module: record.Module}
			index[record.Module] = item
		}
		item.Total++
		switch record.Status {
		case "已上架", "正常":
			item.Published++
		case "待审核", "认证中":
			item.Pending++
		case "需复核":
			item.Review++
		}
	}
	summary := make([]AdminContentSummary, 0, len(index))
	for _, item := range index {
		summary = append(summary, *item)
	}
	sort.Slice(summary, func(i int, j int) bool { return summary[i].Module < summary[j].Module })
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
		input.ID = input.Module + "-" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	}

	record, err := h.upsertContent(c.Request.Context(), input, false)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_create_failed", "could not create admin content")
		return
	}
	record, err = h.syncContentRecord(c.Request.Context(), record)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_sync_failed", "could not sync admin content")
		return
	}
	h.logAudit(c.Request.Context(), "create", record.ID, record.Module, "新建后台内容："+record.Title)
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

	record, err := h.upsertContent(c.Request.Context(), input, true)
	if err != nil {
		if err == sql.ErrNoRows {
			fail(c, http.StatusNotFound, "content_not_found", "admin content record was not found")
			return
		}
		fail(c, http.StatusInternalServerError, "content_update_failed", "could not update admin content")
		return
	}
	record, err = h.syncContentRecord(c.Request.Context(), record)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_sync_failed", "could not sync admin content")
		return
	}
	h.logAudit(c.Request.Context(), "update", record.ID, record.Module, "保存后台内容："+record.Title)
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

	now := nowString()
	result, err := h.db.ExecContext(c.Request.Context(), `
		UPDATE admin_content_records
		SET status = ?, priority = ?, payload = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, input.NextStatus, input.Priority, normalizeJSON(input.Payload), now, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "workflow_update_failed", "could not update admin workflow")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		fail(c, http.StatusNotFound, "content_not_found", "admin content record was not found")
		return
	}

	record, err := h.getContentByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "workflow_update_failed", "could not load admin workflow")
		return
	}
	record, err = h.syncContentRecord(c.Request.Context(), record)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_sync_failed", "could not sync admin workflow")
		return
	}

	detail := input.ActionLabel + "：" + record.Title
	if input.Note != "" {
		detail += "；意见：" + input.Note
	}
	h.logAudit(c.Request.Context(), "workflow:"+input.Action, record.ID, record.Module, detail)
	ok(c, record)
}

func (h *AdminHandler) DeleteContent(c *gin.Context) {
	id := c.Param("id")
	record, err := h.getContentByID(c.Request.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			fail(c, http.StatusNotFound, "content_not_found", "admin content record was not found")
			return
		}
		fail(c, http.StatusInternalServerError, "content_delete_failed", "could not delete admin content")
		return
	}
	now := nowString()
	if _, err := h.db.ExecContext(c.Request.Context(), `
		UPDATE admin_content_records
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, now, now, id); err != nil {
		fail(c, http.StatusInternalServerError, "content_delete_failed", "could not delete admin content")
		return
	}
	if record.Module == "posts" {
		if err := h.softDeleteSyncedPost(c.Request.Context(), record.Payload); err != nil {
			fail(c, http.StatusInternalServerError, "content_sync_failed", "could not remove synced post")
			return
		}
	}
	h.logAudit(c.Request.Context(), "delete", id, record.Module, "删除后台内容："+record.Title)
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

func (h *AdminHandler) syncContentRecord(ctx context.Context, record AdminContentRecord) (AdminContentRecord, error) {
	if record.Module != "posts" {
		return record, nil
	}

	payloadMap := decodePayloadMap(record.Payload)
	postPayload := buildSyncedPostPayload(record, payloadMap)
	postID := postPayload.PostID

	if postID == 0 {
		_ = h.db.QueryRowContext(ctx, `
			SELECT id
			FROM posts
			WHERE title = ? AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1
		`, record.Title).Scan(&postID)
	}

	now := nowString()
	if postID > 0 {
		result, err := h.db.ExecContext(ctx, `
			UPDATE posts
			SET author_name = ?, author_role = ?, title = ?, content = ?, image_urls = ?, tags = ?, track = ?, electives = ?, category = ?, grade = ?, province = ?, deleted_at = NULL, updated_at = ?
			WHERE id = ? AND deleted_at IS NULL
		`,
			record.Owner,
			inferAuthorRole(record),
			record.Title,
			postPayload.Content,
			marshalJSON(postPayload.ImageURLs),
			marshalJSON(record.Tags),
			postPayload.Track,
			marshalJSON(postPayload.Electives),
			postPayload.Category,
			postPayload.Grade,
			postPayload.Province,
			now,
			postID,
		)
		if err != nil {
			return AdminContentRecord{}, err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			postID = 0
		}
	}

	if postID == 0 {
		result, err := h.db.ExecContext(ctx, `
			INSERT INTO posts (author_name, author_role, title, content, image_urls, tags, track, electives, category, grade, province, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			record.Owner,
			inferAuthorRole(record),
			record.Title,
			postPayload.Content,
			marshalJSON(postPayload.ImageURLs),
			marshalJSON(record.Tags),
			postPayload.Track,
			marshalJSON(postPayload.Electives),
			postPayload.Category,
			postPayload.Grade,
			postPayload.Province,
			now,
			now,
		)
		if err != nil {
			return AdminContentRecord{}, err
		}
		lastID, err := result.LastInsertId()
		if err != nil {
			return AdminContentRecord{}, err
		}
		postID = lastID
	}

	payloadMap["postId"] = strconv.FormatInt(postID, 10)
	payloadMap["content"] = postPayload.Content
	payloadMap["track"] = postPayload.Track
	payloadMap["electives"] = postPayload.Electives
	payloadMap["category"] = postPayload.Category
	payloadMap["grade"] = postPayload.Grade
	payloadMap["province"] = postPayload.Province
	payloadMap["imageUrls"] = postPayload.ImageURLs

	if _, err := h.db.ExecContext(ctx, `
		UPDATE admin_content_records
		SET payload = ?, url = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, marshalJSON(payloadMap), h.cfg.RoutePath(fmt.Sprintf("/posts/%d", postID)), now, record.ID); err != nil {
		return AdminContentRecord{}, err
	}
	return h.getContentByID(ctx, record.ID)
}

func (h *AdminHandler) softDeleteSyncedPost(ctx context.Context, payload []byte) error {
	payloadMap := decodePayloadMap(payload)
	postID := payloadInt64(payloadMap, "postId")
	if postID == 0 {
		return nil
	}
	_, err := h.db.ExecContext(ctx, `
		UPDATE posts
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, nowString(), nowString(), postID)
	return err
}

func (h *AdminHandler) AuditLogs(c *gin.Context) {
	rows, err := h.db.QueryContext(c.Request.Context(), `
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
		var createdAt string
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
			"createdAt": parseSQLiteTime(createdAt),
		})
	}
	ok(c, envelope{"logs": logs})
}

func (h *AdminHandler) upsertContent(ctx context.Context, input AdminContentInput, updateOnly bool) (AdminContentRecord, error) {
	now := nowString()
	if updateOnly {
		result, err := h.db.ExecContext(ctx, `
			UPDATE admin_content_records
			SET module = ?, title = ?, content_type = ?, status = ?, scope = ?, owner = ?, tags = ?, summary = ?, url = ?, priority = ?, sort_order = ?, payload = ?, updated_at = ?
			WHERE id = ? AND deleted_at IS NULL
		`,
			input.Module, input.Title, input.Type, input.Status, input.Scope, input.Owner, marshalJSON(input.Tags), input.Summary,
			input.URL, input.Priority, input.SortOrder, normalizeJSON(input.Payload), now, input.ID,
		)
		if err != nil {
			return AdminContentRecord{}, err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return AdminContentRecord{}, sql.ErrNoRows
		}
		return h.getContentByID(ctx, input.ID)
	}

	if _, err := h.db.ExecContext(ctx, `
		INSERT INTO admin_content_records (id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
		ON CONFLICT(id) DO UPDATE SET
			module = excluded.module,
			title = excluded.title,
			content_type = excluded.content_type,
			status = excluded.status,
			scope = excluded.scope,
			owner = excluded.owner,
			tags = excluded.tags,
			summary = excluded.summary,
			url = excluded.url,
			priority = excluded.priority,
			sort_order = excluded.sort_order,
			payload = excluded.payload,
			deleted_at = NULL,
			updated_at = excluded.updated_at
	`,
		input.ID, input.Module, input.Title, input.Type, input.Status, input.Scope, input.Owner, marshalJSON(input.Tags),
		input.Summary, input.URL, input.Priority, input.SortOrder, normalizeJSON(input.Payload), now, now,
	); err != nil {
		return AdminContentRecord{}, err
	}
	return h.getContentByID(ctx, input.ID)
}

func (h *AdminHandler) listContentRecords(ctx context.Context, publishedOnly bool) ([]AdminContentRecord, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT id, module, title, content_type, status, scope, owner, tags, summary, url,
		       priority, sort_order, payload, created_at, updated_at
		FROM admin_content_records
		WHERE deleted_at IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]AdminContentRecord, 0)
	for rows.Next() {
		record, err := scanAdminContent(rows)
		if err != nil {
			return nil, err
		}
		if publishedOnly {
			if record.Module == "users" {
				continue
			}
			if record.Status != "已上架" && record.Status != "正常" {
				continue
			}
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(records, func(i int, j int) bool {
		if records[i].Module == records[j].Module {
			if records[i].SortOrder == records[j].SortOrder {
				return records[i].UpdatedAt.After(records[j].UpdatedAt)
			}
			return records[i].SortOrder < records[j].SortOrder
		}
		return records[i].Module < records[j].Module
	})
	return records, nil
}

func (h *AdminHandler) getContentByID(ctx context.Context, id string) (AdminContentRecord, error) {
	return scanAdminContent(h.db.QueryRowContext(ctx, `
		SELECT id, module, title, content_type, status, scope, owner, tags, summary, url,
		       priority, sort_order, payload, created_at, updated_at
		FROM admin_content_records
		WHERE id = ? AND deleted_at IS NULL
	`, id))
}

func filterAdminRecords(records []AdminContentRecord, module string, status string, keyword string) []AdminContentRecord {
	filtered := make([]AdminContentRecord, 0, len(records))
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	for _, record := range records {
		if module != "" && record.Module != module {
			continue
		}
		if status != "" && status != "全部" && record.Status != status {
			continue
		}
		if keyword != "" {
			text := strings.ToLower(strings.Join([]string{
				record.Title,
				record.Type,
				record.Status,
				record.Scope,
				record.Owner,
				record.Summary,
				strings.Join(record.Tags, ","),
			}, " "))
			if !strings.Contains(text, keyword) {
				continue
			}
		}
		filtered = append(filtered, record)
	}
	return filtered
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

func detectImageUpload(fileHeader *multipart.FileHeader) (string, string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", "", fmt.Errorf("could not read uploaded image")
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", "", fmt.Errorf("could not inspect uploaded image")
	}

	if n >= 12 && string(buffer[0:4]) == "RIFF" && string(buffer[8:12]) == "WEBP" {
		return "image/webp", ".webp", nil
	}

	switch contentType := http.DetectContentType(buffer[:n]); contentType {
	case "image/jpeg":
		return contentType, ".jpg", nil
	case "image/png":
		return contentType, ".png", nil
	case "image/gif":
		return contentType, ".gif", nil
	case "image/webp":
		return contentType, ".webp", nil
	default:
		return "", "", fmt.Errorf("only JPG, PNG, GIF, and WebP images are supported")
	}
}

func saveUploadedFile(fileHeader *multipart.FileHeader, targetPath string) error {
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer dst.Close()

	written, err := io.Copy(dst, io.LimitReader(src, maxAdminImageUploadBytes+1))
	if err != nil {
		return err
	}
	if written > maxAdminImageUploadBytes {
		return fmt.Errorf("uploaded image is too large")
	}
	return nil
}

func randomHex(byteCount int) string {
	bytes := make([]byte, byteCount)
	if _, err := rand.Read(bytes); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(bytes)
}

type adminContentScanner interface {
	Scan(dest ...any) error
}

func scanAdminContent(scanner adminContentScanner) (AdminContentRecord, error) {
	var record AdminContentRecord
	var tagsRaw string
	var payloadRaw string
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(
		&record.ID,
		&record.Module,
		&record.Title,
		&record.Type,
		&record.Status,
		&record.Scope,
		&record.Owner,
		&tagsRaw,
		&record.Summary,
		&record.URL,
		&record.Priority,
		&record.SortOrder,
		&payloadRaw,
		&createdAt,
		&updatedAt,
	); err != nil {
		return AdminContentRecord{}, err
	}
	record.Tags = parseJSONStringSlice(tagsRaw)
	if strings.TrimSpace(payloadRaw) == "" {
		payloadRaw = "{}"
	}
	record.Payload = json.RawMessage(payloadRaw)
	record.CreatedAt = parseSQLiteTime(createdAt)
	record.UpdatedAt = parseSQLiteTime(updatedAt)
	return record, nil
}

func (h *AdminHandler) logAudit(ctx context.Context, action string, recordID string, module string, detail string) {
	_, _ = h.db.ExecContext(ctx, `
		INSERT INTO admin_audit_logs (action, record_id, module, detail, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, action, recordID, module, detail, nowString())
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

func parseJSONStringSlice(raw string) []string {
	values := []string{}
	if strings.TrimSpace(raw) == "" {
		return values
	}
	_ = json.Unmarshal([]byte(raw), &values)
	return values
}

func normalizeJSON(raw json.RawMessage) string {
	if len(raw) == 0 || !json.Valid(raw) {
		return "{}"
	}
	return string(raw)
}

func marshalJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func parseSQLiteTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
