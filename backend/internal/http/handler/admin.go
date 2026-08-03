package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/http/middleware"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	cfg           config.Config
	service       *service.ForumService
	db            *sql.DB
	sessionStore  *middleware.AdminSessionStore
	secureCookies bool
}

func NewAdminHandler(cfg config.Config, forumService *service.ForumService, db *sql.DB, sessionStore *middleware.AdminSessionStore) *AdminHandler {
	return &AdminHandler{cfg: cfg, service: forumService, db: db, sessionStore: sessionStore, secureCookies: cfg.Production()}
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

type AdminUserPasswordInput struct {
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type AdminUserModerationInput struct {
	Reason string `json:"reason" binding:"omitempty,max=1200"`
}

type AdminWorkflowInput struct {
	Action      string          `json:"action" binding:"required,max=80"`
	ActionLabel string          `json:"actionLabel" binding:"required,max=80"`
	NextStatus  string          `json:"nextStatus" binding:"required,max=30"`
	Note        string          `json:"note" binding:"omitempty,max=1200"`
	Priority    string          `json:"priority" binding:"omitempty,max=20"`
	Payload     json.RawMessage `json:"payload"`
}

type adminWorkflowTransition struct {
	From  string
	To    string
	Label string
}

var adminContentWorkflowTransitions = map[string]adminWorkflowTransition{
	"submit-content-review:草稿":   {From: "草稿", To: "待审核", Label: "提交审核"},
	"submit-content-review:退回修改": {From: "退回修改", To: "待审核", Label: "重新提交审核"},
	"submit-content-review:下架":   {From: "下架", To: "待审核", Label: "整改后提交审核"},
	"approve-content:待审核":        {From: "待审核", To: "已上架", Label: "审核通过"},
	"send-to-review:待审核":         {From: "待审核", To: "需复核", Label: "转入复核"},
	"return-content:待审核":         {From: "待审核", To: "退回修改", Label: "退回修改"},
	"pass-review:需复核":            {From: "需复核", To: "已上架", Label: "复核通过并上架"},
	"keep-review:需复核":            {From: "需复核", To: "需复核", Label: "继续复核"},
	"reject-after-review:需复核":    {From: "需复核", To: "下架", Label: "复核不通过下架"},
	"start-review:已上架":           {From: "已上架", To: "需复核", Label: "发起复核"},
	"unpublish-content:已上架":      {From: "已上架", To: "下架", Label: "确认下架"},
}

var adminContentWorkflowActions = map[string]struct{}{
	"submit-content-review": {}, "approve-content": {}, "send-to-review": {},
	"return-content": {}, "pass-review": {}, "keep-review": {},
	"reject-after-review": {}, "start-review": {}, "unpublish-content": {},
}

type AdminModerationInput struct {
	Action string `json:"action" binding:"required,oneof=hide restore dismiss"`
	Note   string `json:"note" binding:"omitempty,max=1200"`
}

type AdminContentSummary struct {
	Module    string `json:"module"`
	Total     int    `json:"total"`
	Published int    `json:"published"`
	Pending   int    `json:"pending"`
	Review    int    `json:"review"`
}

type LocalPolicyDocument struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
	Size int64  `json:"sizeBytes"`
}

const (
	maxAdminImageUploadBytes int64 = 8 * 1024 * 1024
	maxAdminImageDimension         = 12000
	maxAdminImagePixels            = 80_000_000
)

func (h *AdminHandler) Login(c *gin.Context) {
	var input AdminLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if h.cfg.AdminEmail == "" || (h.cfg.AdminPassword == "" && h.cfg.AdminPasswordHash == "") {
		fail(c, http.StatusServiceUnavailable, "admin_login_disabled", "admin login is not configured")
		return
	}
	if email != h.cfg.AdminEmail || !h.validAdminPassword(input.Password) {
		fail(c, http.StatusUnauthorized, "invalid_admin_credentials", "invalid admin email or password")
		return
	}
	role := h.cfg.AdminRole
	if role == "" {
		role = middleware.AdminRoleSuperAdmin
	}
	principal, err := middleware.NewAdminPrincipal(email, role)
	if err != nil {
		fail(c, http.StatusServiceUnavailable, "admin_role_invalid", "admin role is not configured")
		return
	}
	token, expiresAt, err := h.sessionStore.Issue(principal)
	if err != nil {
		fail(c, http.StatusInternalServerError, "admin_session_failed", "could not create admin session")
		return
	}
	maxAge := int(time.Until(expiresAt).Seconds())
	middleware.SetAdminSessionCookie(c, token, maxAge, h.secureCookies)
	csrfToken, err := middleware.GenerateCSRFToken()
	if err == nil {
		middleware.SetAdminCSRFCookie(c, csrfToken, maxAge, h.secureCookies)
	}
	ok(c, envelope{
		"email": email, "role": principal.Role, "permissions": principal.Permissions,
		"expiresAt": expiresAt,
	})
}

func (h *AdminHandler) Logout(c *gin.Context) {
	if token, err := c.Cookie(middleware.AdminSessionCookieName); err == nil {
		h.sessionStore.Revoke(token)
	}
	middleware.ClearAdminSessionCookie(c, h.secureCookies)
	middleware.ClearAdminCSRFCookie(c, h.secureCookies)
	ok(c, envelope{"signedOut": true})
}

func (h *AdminHandler) validAdminPassword(password string) bool {
	if h.cfg.AdminPasswordHash != "" {
		return bcrypt.CompareHashAndPassword([]byte(h.cfg.AdminPasswordHash), []byte(password)) == nil
	}
	return h.cfg.AdminPassword != "" && h.cfg.AdminPassword == password
}

func (h *AdminHandler) ResetUserPassword(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		fail(c, http.StatusBadRequest, "invalid_user_id", "user id is invalid")
		return
	}
	var input AdminUserPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, "password_hash_failed", "could not secure the new password")
		return
	}
	result, err := h.exec(c.Request.Context(), `
		UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL
	`, string(hash), nowString(), userID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "password_update_failed", "could not update user password")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		fail(c, http.StatusNotFound, "user_not_found", "user was not found")
		return
	}
	recordID := fmt.Sprintf("user-%d", userID)
	h.logAudit(c.Request.Context(), "reset_password", recordID, "users", "管理员已重置用户密码")
	ok(c, envelope{"updated": true, "userId": userID})
}

func (h *AdminHandler) BanUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		fail(c, http.StatusBadRequest, "invalid_user_id", "user id is invalid")
		return
	}
	var input AdminUserModerationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "违反社区规则"
	}
	now := nowString()
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		fail(c, http.StatusInternalServerError, "ban_user_failed", "could not ban user")
		return
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(c.Request.Context(), bindDatabaseQuery(h.cfg.DatabaseDriver, `
		UPDATE users
		SET banned_at = COALESCE(banned_at, ?), banned_reason = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL AND is_shadow = false
	`), now, reason, now, userID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "ban_user_failed", "could not ban user")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		fail(c, http.StatusNotFound, "user_not_found", "user was not found")
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), bindDatabaseQuery(h.cfg.DatabaseDriver, `UPDATE auth_sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`), now, userID); err != nil {
		fail(c, http.StatusInternalServerError, "ban_user_failed", "could not revoke user sessions")
		return
	}
	if err := tx.Commit(); err != nil {
		fail(c, http.StatusInternalServerError, "ban_user_failed", "could not ban user")
		return
	}
	h.logAudit(c.Request.Context(), "ban_user", fmt.Sprintf("user-%d", userID), "users", "管理员已封禁用户："+reason)
	ok(c, envelope{"banned": true, "userId": userID})
}

func (h *AdminHandler) RestoreUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		fail(c, http.StatusBadRequest, "invalid_user_id", "user id is invalid")
		return
	}
	var input AdminUserModerationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	note := strings.TrimSpace(input.Reason)
	now := nowString()
	result, err := h.exec(c.Request.Context(), `
		UPDATE users
		SET banned_at = NULL, banned_reason = '', updated_at = ?
		WHERE id = ? AND deleted_at IS NULL AND is_shadow = false
	`, now, userID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "restore_user_failed", "could not restore user")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		fail(c, http.StatusNotFound, "user_not_found", "user was not found")
		return
	}
	detail := "管理员已恢复用户"
	if note != "" {
		detail += "：" + note
	}
	h.logAudit(c.Request.Context(), "restore_user", fmt.Sprintf("user-%d", userID), "users", detail)
	ok(c, envelope{"restored": true, "userId": userID})
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
		"enabled":                                h.cfg.SMTPEnabled(),
		"host":                                   h.cfg.SMTPHost,
		"port":                                   h.cfg.SMTPPort,
		"usernameConfigured":                     h.cfg.SMTPUsername != "",
		"passwordConfigured":                     h.cfg.SMTPPassword != "",
		"fromEmail":                              h.cfg.SMTPFromEmail,
		"replyTo":                                h.cfg.SMTPReplyTo,
		"fromName":                               h.cfg.SMTPFromName,
		"useTLS":                                 h.cfg.SMTPUseTLS,
		"startTLS":                               h.cfg.SMTPStartTLS,
		"emailVerificationTTLMinutes":            h.cfg.EmailVerificationTTLMinutes,
		"emailVerificationCooldownSeconds":       h.cfg.EmailVerificationCooldownSeconds,
		"emailVerificationEmailHourlyLimit":      h.cfg.EmailVerificationEmailHourlyLimit,
		"emailVerificationIPHourlyLimit":         h.cfg.EmailVerificationIPHourlyLimit,
		"emailVerificationMaxValidationAttempts": h.cfg.EmailVerificationMaxValidationAttempts,
		"missing":                                missing,
	})
}

func (h *AdminHandler) SendTestEmail(c *gin.Context) {
	var input domain.EmailVerificationCodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	input.ClientIP = requestRemoteIP(c.Request)
	result, err := h.service.SendEmailVerificationCode(c.Request.Context(), input)
	if err != nil {
		if handleEmailVerificationRateLimit(c, err) {
			return
		}
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

	contentType, ext, width, height, err := detectImageUpload(fileHeader)
	if err != nil {
		fail(c, http.StatusBadRequest, "unsupported_file_type", err.Error())
		return
	}
	if width <= 0 || height <= 0 || width > maxAdminImageDimension || height > maxAdminImageDimension || width*height > maxAdminImagePixels {
		fail(c, http.StatusBadRequest, "image_dimensions_too_large", "image dimensions must be within 12000px per side and 80MP")
		return
	}

	dateDir := time.Now().UTC().Format("20060102")
	fileName := randomHex(16) + ext
	targetDir := filepath.Join(h.cfg.MediaUploadDir, "images", dateDir)
	if err := os.MkdirAll(targetDir, 0750); err != nil {
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
		"width":       width,
		"height":      height,
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
	if !adminHasPermission(c, middleware.AdminPermissionUsersRead) {
		filtered := records[:0]
		for _, record := range records {
			if record.Module != "users" {
				filtered = append(filtered, record)
			}
		}
		records = filtered
	}
	ok(c, envelope{"records": records})
}

func adminHasPermission(c *gin.Context, permission string) bool {
	principal, ok := middleware.CurrentAdminPrincipal(c)
	if !ok {
		return false
	}
	for _, candidate := range principal.Permissions {
		if candidate == permission {
			return true
		}
	}
	return false
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

func (h *AdminHandler) ListProvinces(c *gin.Context) {
	if h.cfg.DatabaseDriver == "postgres" {
		h.listTypedProvinces(c)
		return
	}
	rows, err := h.query(c.Request.Context(), `
		SELECT province, SUM(records_count), MAX(updated_at)
		FROM (
			SELECT scope AS province, COUNT(*) AS records_count, MAX(updated_at) AS updated_at
			FROM admin_content_records
			WHERE deleted_at IS NULL AND status IN ('已上架', '正常') AND scope <> ''
			GROUP BY scope
			UNION ALL
			SELECT province, COUNT(*) AS records_count, MAX(updated_at) AS updated_at
			FROM subject_insights
			WHERE province <> ''
			GROUP BY province
		)
		GROUP BY province
		ORDER BY CASE province WHEN '广东' THEN 0 WHEN '全国' THEN 1 ELSE 2 END, province ASC
	`)
	if err != nil {
		fail(c, http.StatusInternalServerError, "provinces_query_failed", "could not load provinces")
		return
	}
	defer rows.Close()
	items := make([]envelope, 0)
	seen := make(map[string]int)
	for rows.Next() {
		var province, updatedAt string
		var count int
		if err := rows.Scan(&province, &count, &updatedAt); err != nil {
			fail(c, http.StatusInternalServerError, "provinces_scan_failed", "could not read provinces")
			return
		}
		if province == "全站" || province == "全国" {
			continue
		}
		status := "verified"
		methodology := "已完成来源核对与结构化记录复核。"
		items = append(items, envelope{
			"province":       province,
			"coverageStatus": status,
			"recordsCount":   count,
			"dataYear":       2025,
			"capturedAt":     parseSQLiteTime(updatedAt),
			"methodology":    methodology,
		})
		seen[province] = len(items) - 1
	}
	if err := rows.Err(); err != nil {
		fail(c, http.StatusInternalServerError, "provinces_scan_failed", "could not read provinces")
		return
	}
	// SQLite deployments also expose structured 2026 source records. Keep
	// province coverage honest while allowing the knowledge page to show the
	// newest captured year and its review state.
	records, recordsErr := h.listContentRecords(c.Request.Context(), true)
	if recordsErr == nil {
		for _, record := range records {
			if record.Module != "policies" && record.Module != "requirements" || record.Scope == "" || record.Scope == "全国" {
				continue
			}
			meta := recordMetadata(record)
			item := envelope{"province": record.Scope, "coverageStatus": meta.status, "recordsCount": 1, "dataYear": meta.year, "capturedAt": record.UpdatedAt, "methodology": meta.methodology}
			if index, ok := seen[record.Scope]; ok {
				currentYear, _ := items[index]["dataYear"].(int)
				if meta.year > currentYear {
					items[index] = item
				}
				continue
			}
			items = append(items, item)
			seen[record.Scope] = len(items) - 1
		}
	}
	ok(c, envelope{"provinces": items})
}

func (h *AdminHandler) ListPolicies(c *gin.Context) {
	if h.cfg.DatabaseDriver == "postgres" {
		h.listTypedRealData(c, "policies")
		return
	}
	h.listPublishedModule(c, "policies", "policies")
}

func (h *AdminHandler) DownloadPolicyDocument(c *gin.Context) {
	root := h.policyDocumentsRoot()
	scope := strings.TrimSpace(c.Param("scope"))
	name := filepath.Base(strings.TrimSpace(c.Param("filename")))
	if scope == "" || name == "" || name == "." || name == ".." {
		c.Status(http.StatusBadRequest)
		return
	}
	target := filepath.Join(root, scope, name)
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		c.Status(http.StatusNotFound)
		return
	}
	info, err := os.Stat(target)
	if err != nil || !info.Mode().IsRegular() {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(name)))
	c.Header("Cache-Control", "public, max-age=86400")
	c.File(target)
}

func (h *AdminHandler) ListRequirements(c *gin.Context) {
	if h.cfg.DatabaseDriver == "postgres" {
		h.listTypedRealData(c, "requirements")
		return
	}
	h.listPublishedModule(c, "requirements", "requirements")
}

func (h *AdminHandler) GetSource(c *gin.Context) {
	if h.cfg.DatabaseDriver == "postgres" {
		h.getTypedSource(c)
		return
	}
	sourceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || sourceID <= 0 {
		fail(c, http.StatusBadRequest, "invalid_source_id", "source id is invalid")
		return
	}
	var capturedAt string
	var id, postID int64
	var sourcePlatform, sourceURL, sourceNoteID, sourceTitle, sourceAuthor, sourceFormat, methodology, title, scope string
	err = h.queryRow(c.Request.Context(), `
		SELECT cs.id, cs.post_id, cs.source_platform, cs.source_url, cs.source_note_id, cs.source_title,
		       cs.source_author, cs.source_format, cs.transformation_note, cs.captured_at,
		       p.title, p.province
		FROM content_sources cs
		JOIN posts p ON p.id = cs.post_id
		WHERE cs.id = ?
	`, sourceID).Scan(
		&id, &postID, &sourcePlatform, &sourceURL, &sourceNoteID, &sourceTitle,
		&sourceAuthor, &sourceFormat, &methodology, &capturedAt, &title, &scope,
	)
	if errors.Is(err, sql.ErrNoRows) {
		fail(c, http.StatusNotFound, "not_found", "source not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, "source_query_failed", "could not load source")
		return
	}
	item := envelope{
		"id":             id,
		"postId":         postID,
		"sourcePlatform": sourcePlatform,
		"sourceUrl":      sourceURL,
		"sourceNoteId":   sourceNoteID,
		"sourceTitle":    sourceTitle,
		"sourceAuthor":   sourceAuthor,
		"sourceFormat":   sourceFormat,
		"methodology":    methodology,
		"title":          title,
		"scope":          scope,
	}
	item["coverageStatus"] = "unverified"
	item["capturedAt"] = parseSQLiteTime(capturedAt)
	item["fileHash"] = ""
	ok(c, item)
}

func (h *AdminHandler) listTypedProvinces(c *gin.Context) {
	rows, err := h.query(c.Request.Context(), `
		SELECT name, coverage_status, records_count, data_year, captured_at, methodology
		FROM provinces
		ORDER BY CASE name WHEN '广东' THEN 0 ELSE 1 END, name`)
	if err != nil {
		fail(c, http.StatusInternalServerError, "provinces_query_failed", "could not load provinces")
		return
	}
	defer rows.Close()
	items := make([]envelope, 0)
	for rows.Next() {
		var name, status, methodology string
		var recordsCount, dataYear int
		var capturedAt time.Time
		if err := rows.Scan(&name, &status, &recordsCount, &dataYear, &capturedAt, &methodology); err != nil {
			fail(c, http.StatusInternalServerError, "provinces_scan_failed", "could not read provinces")
			return
		}
		items = append(items, envelope{"province": name, "coverageStatus": status, "recordsCount": recordsCount, "dataYear": dataYear, "capturedAt": capturedAt, "methodology": methodology})
	}
	if err := rows.Err(); err != nil {
		fail(c, http.StatusInternalServerError, "provinces_scan_failed", "could not read provinces")
		return
	}
	ok(c, envelope{"provinces": items})
}

func (h *AdminHandler) listTypedRealData(c *gin.Context, table string) {
	if table != "policies" && table != "requirements" {
		fail(c, http.StatusInternalServerError, "content_query_failed", "invalid real data module")
		return
	}
	requiredSubjectsExpression := `'[]'::jsonb`
	if table == "requirements" {
		requiredSubjectsExpression = "r.required_subjects"
	}
	query := fmt.Sprintf(`
		SELECT r.id::text, r.title, r.type, r.scope, r.coverage_status, r.data_year,
		       r.captured_at, s.name, s.url, s.file_hash, r.methodology, r.summary,
		       r.tags, r.url, s.id::text, %s
		FROM %s r
		JOIN sources s ON s.id = r.source_id
		WHERE r.deleted_at IS NULL
		ORDER BY r.data_year DESC, r.captured_at DESC, r.id DESC`, requiredSubjectsExpression, table)
	rows, err := h.query(c.Request.Context(), query)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_query_failed", "could not load "+table)
		return
	}
	defer rows.Close()
	items := make([]envelope, 0)
	for rows.Next() {
		var id, title, recordType, scope, status, sourceName, sourceURL, fileHash, methodology, summary, recordURL, sourceID string
		var dataYear int
		var capturedAt time.Time
		var tagsJSON, requiredSubjectsJSON []byte
		if err := rows.Scan(&id, &title, &recordType, &scope, &status, &dataYear, &capturedAt, &sourceName, &sourceURL, &fileHash, &methodology, &summary, &tagsJSON, &recordURL, &sourceID, &requiredSubjectsJSON); err != nil {
			fail(c, http.StatusInternalServerError, "content_scan_failed", "could not read "+table)
			return
		}
		var tags []string
		if err := json.Unmarshal(tagsJSON, &tags); err != nil {
			fail(c, http.StatusInternalServerError, "content_scan_failed", "could not decode "+table+" tags")
			return
		}
		var requiredSubjects []string
		if err := json.Unmarshal(requiredSubjectsJSON, &requiredSubjects); err != nil {
			fail(c, http.StatusInternalServerError, "content_scan_failed", "could not decode "+table+" required subjects")
			return
		}
		items = append(items, envelope{
			"id": id, "title": title, "type": recordType, "scope": scope,
			"coverageStatus": status, "dataYear": dataYear, "capturedAt": capturedAt,
			"source":   envelope{"id": sourceID, "name": sourceName, "url": sourceURL},
			"fileHash": fileHash, "methodology": methodology, "summary": summary,
			"tags": tags, "url": recordURL, "requiredSubjects": requiredSubjects,
		})
	}
	if err := rows.Err(); err != nil {
		fail(c, http.StatusInternalServerError, "content_scan_failed", "could not read "+table)
		return
	}
	ok(c, envelope{table: items})
}

func (h *AdminHandler) getTypedSource(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		fail(c, http.StatusBadRequest, "invalid_source_id", "source id is invalid")
		return
	}
	var sourceID, name, sourceURL, assetKey, fileHash, scope, methodology, status string
	var dataYear int
	var capturedAt time.Time
	err := h.queryRow(c.Request.Context(), `
		SELECT id::text, name, url, COALESCE(asset_key, ''), file_hash, data_year,
		       scope, captured_at, methodology, coverage_status
		FROM sources WHERE id::text = ?`, id).Scan(&sourceID, &name, &sourceURL, &assetKey, &fileHash, &dataYear, &scope, &capturedAt, &methodology, &status)
	if errors.Is(err, sql.ErrNoRows) {
		fail(c, http.StatusNotFound, "not_found", "source not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, "source_query_failed", "could not load source")
		return
	}
	ok(c, envelope{
		"id": sourceID, "name": name, "sourceUrl": sourceURL,
		"fileHash": fileHash, "dataYear": dataYear, "scope": scope,
		"capturedAt": capturedAt, "methodology": methodology, "coverageStatus": status,
	})
}

func (h *AdminHandler) listPublishedModule(c *gin.Context, module string, key string) {
	records, err := h.listContentRecords(c.Request.Context(), true)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_query_failed", "could not load "+key)
		return
	}
	records = filterAdminRecords(records, module, "", strings.TrimSpace(c.Query("q")))
	items := make([]envelope, 0, len(records))
	for _, record := range records {
		meta := recordMetadata(record)
		var payload map[string]any
		_ = json.Unmarshal(record.Payload, &payload)
		requiredSubjects := []string{}
		if values, ok := payload["requiredSubjects"].([]any); ok {
			for _, value := range values {
				if subject, ok := value.(string); ok && subject != "" {
					requiredSubjects = append(requiredSubjects, subject)
				}
			}
		}
		scope := record.Scope
		items = append(items, envelope{
			"id":             record.ID,
			"title":          record.Title,
			"type":           record.Type,
			"scope":          scope,
			"coverageStatus": meta.status,
			"dataYear":       meta.year,
			"capturedAt":     record.UpdatedAt,
			"source": envelope{
				"name": "阳光高考 / 各省教育考试院官方目录",
				"url":  "https://gaokao.chsi.com.cn/",
			},
			"fileHash":         "",
			"methodology":      meta.methodology,
			"summary":          record.Summary,
			"tags":             record.Tags,
			"url":              record.URL,
			"requiredSubjects": requiredSubjects,
			"localDocuments":   h.localPolicyDocuments(record.Scope),
		})
	}
	ok(c, envelope{key: items})
}

func (h *AdminHandler) policyDocumentsRoot() string {
	if h.cfg.SQLitePath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(h.cfg.SQLitePath), "policy_documents", "2026")
}

func (h *AdminHandler) localPolicyDocuments(scope string) []LocalPolicyDocument {
	root := h.policyDocumentsRoot()
	if root == "" || strings.TrimSpace(scope) == "" {
		return []LocalPolicyDocument{}
	}
	directory := filepath.Join(root, scope)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return []LocalPolicyDocument{}
	}
	documents := make([]LocalPolicyDocument, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "manifest.json" || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		docType := "文件"
		switch ext {
		case ".pdf":
			docType = "PDF"
		case ".doc", ".docx":
			docType = "Word"
		case ".xls", ".xlsx":
			docType = "Excel"
		case ".zip", ".rar", ".7z":
			docType = "压缩包"
		case ".html", ".htm":
			docType = "来源页"
		}
		documents = append(documents, LocalPolicyDocument{
			Name: entry.Name(),
			URL:  "/api/v1/policy-documents/" + url.PathEscape(scope) + "/" + url.PathEscape(entry.Name()),
			Type: docType,
			Size: info.Size(),
		})
	}
	sort.Slice(documents, func(i, j int) bool { return documents[i].Name < documents[j].Name })
	return documents
}

type contentMetadata struct {
	year        int
	status      string
	methodology string
}

func recordMetadata(record AdminContentRecord) contentMetadata {
	meta := contentMetadata{year: 2025, status: "unverified", methodology: "官方入口已收录，结构化结论待复核。"}
	var payload map[string]any
	if json.Unmarshal(record.Payload, &payload) == nil {
		if year, ok := payload["dataYear"].(float64); ok && int(year) > 0 {
			meta.year = int(year)
		}
		if status, ok := payload["coverageStatus"].(string); ok && (status == "verified" || status == "unverified") {
			meta.status = status
		}
		if methodology, ok := payload["methodology"].(string); ok && strings.TrimSpace(methodology) != "" {
			meta.methodology = methodology
		}
	}
	return meta
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

	record, err := h.getContentByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			fail(c, http.StatusNotFound, "content_not_found", "admin content record was not found")
			return
		}
		fail(c, http.StatusInternalServerError, "workflow_update_failed", "could not load admin workflow")
		return
	}
	if record.Module == "users" {
		fail(c, http.StatusBadRequest, "invalid_workflow_action", "user records must use user moderation endpoints")
		return
	}
	if _, ok := adminContentWorkflowActions[input.Action]; !ok {
		fail(c, http.StatusBadRequest, "invalid_workflow_action", "action is not allowed for admin content")
		return
	}
	transition, transitionAllowed := adminContentWorkflowTransitions[input.Action+":"+record.Status]
	if !transitionAllowed {
		fail(c, http.StatusConflict, "invalid_workflow_transition", "action is not allowed from the current content status")
		return
	}
	if input.NextStatus != transition.To {
		fail(c, http.StatusBadRequest, "invalid_workflow_status", "nextStatus does not match the server workflow")
		return
	}

	now := nowString()
	payload := appendAdminWorkflowEntry(record.Payload, transition, input.Note, now, adminWorkflowActor(c.Request.Context()))
	priority := adminWorkflowPriority(transition.To)
	result, err := h.exec(c.Request.Context(), `
		UPDATE admin_content_records
		SET status = ?, priority = ?, payload = ?, updated_at = ?
		WHERE id = ? AND status = ? AND deleted_at IS NULL
	`, transition.To, priority, payload, now, record.ID, transition.From)
	if err != nil {
		fail(c, http.StatusInternalServerError, "workflow_update_failed", "could not update admin workflow")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		fail(c, http.StatusConflict, "workflow_conflict", "content status changed while applying workflow action")
		return
	}

	record, err = h.getContentByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "workflow_update_failed", "could not load admin workflow")
		return
	}
	record, err = h.syncContentRecord(c.Request.Context(), record)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content_sync_failed", "could not sync admin workflow")
		return
	}

	detail := transition.Label + "：" + record.Title
	if input.Note != "" {
		detail += "；意见：" + input.Note
	}
	h.logAudit(c.Request.Context(), "workflow:"+input.Action, record.ID, record.Module, detail)
	ok(c, record)
}

func appendAdminWorkflowEntry(raw json.RawMessage, transition adminWorkflowTransition, note string, at string, actor string) string {
	payload := decodePayloadMap(raw)
	workflow, _ := payload["workflow"].([]any)
	payload["workflow"] = append(workflow, map[string]any{
		"time": at, "action": transition.Label, "from": transition.From, "to": transition.To,
		"note": note, "actor": actor,
	})
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func adminWorkflowPriority(status string) string {
	switch status {
	case "需复核", "下架":
		return "高"
	case "待审核", "退回修改":
		return "中"
	default:
		return "常规"
	}
}

func adminWorkflowActor(ctx context.Context) string {
	if principal, ok := middleware.AdminPrincipalFromContext(ctx); ok {
		return principal.Email + " (" + principal.Role + ")"
	}
	return "admin"
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
	if _, err := h.exec(c.Request.Context(), `
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
	ownerUserID, err := h.resolvePostOwnerID(ctx, postID, payloadMap)
	if err != nil {
		return AdminContentRecord{}, err
	}
	var ownerUserIDValue any
	if ownerUserID != nil {
		ownerUserIDValue = *ownerUserID
	}
	var deletedAt any
	if !isPublishedContentStatus(record.Status) {
		deletedAt = nowString()
	}

	now := nowString()
	if postID > 0 {
		result, err := h.exec(ctx, `
			UPDATE posts
			SET user_id = COALESCE(user_id, ?), author_name = ?, author_role = ?, title = ?, content = ?, image_urls = ?, tags = ?, track = ?, electives = ?, category = ?, grade = ?, province = ?, deleted_at = ?, updated_at = ?
			WHERE id = ?
		`,
			ownerUserIDValue,
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
			deletedAt,
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
		err := h.queryRow(ctx, `
			INSERT INTO posts (user_id, author_name, author_role, title, content, image_urls, tags, track, electives, category, grade, province, created_at, updated_at, deleted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			RETURNING id
		`,
			ownerUserIDValue,
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
			deletedAt,
		).Scan(&postID)
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
	if ownerUserID != nil {
		payloadMap["createdByUserId"] = strconv.FormatInt(*ownerUserID, 10)
	}

	if _, err := h.exec(ctx, `
		UPDATE admin_content_records
		SET payload = ?, url = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, marshalJSON(payloadMap), h.cfg.RoutePath(fmt.Sprintf("/posts/%d", postID)), now, record.ID); err != nil {
		return AdminContentRecord{}, err
	}
	return h.getContentByID(ctx, record.ID)
}

func (h *AdminHandler) resolvePostOwnerID(ctx context.Context, postID int64, payload map[string]any) (*int64, error) {
	if postID > 0 {
		var currentUserID sql.NullInt64
		err := h.queryRow(ctx, `SELECT user_id FROM posts WHERE id = ?`, postID).Scan(&currentUserID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if currentUserID.Valid {
			return &currentUserID.Int64, nil
		}
	}
	userID := payloadInt64(payload, "createdByUserId")
	if userID == 0 {
		userID = payloadInt64(payload, "userId")
	}
	if userID == 0 {
		return nil, nil
	}
	var existingID int64
	err := h.queryRow(ctx, `SELECT id FROM users WHERE id = ?`, userID).Scan(&existingID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &existingID, nil
}

func (h *AdminHandler) softDeleteSyncedPost(ctx context.Context, payload []byte) error {
	payloadMap := decodePayloadMap(payload)
	postID := payloadInt64(payloadMap, "postId")
	if postID == 0 {
		return nil
	}
	_, err := h.exec(ctx, `
		UPDATE posts
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, nowString(), nowString(), postID)
	return err
}

func (h *AdminHandler) AuditLogs(c *gin.Context) {
	rows, err := h.query(c.Request.Context(), `
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

func (h *AdminHandler) ListReports(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	query := `
		SELECT cr.id, cr.reporter_user_id, COALESCE(u.nickname, ''), cr.target_type, cr.target_id,
		       COALESCE(p.title, ''), COALESCE(p.author_name, ''),
		       CASE WHEN p.deleted_at IS NOT NULL THEN 1 ELSE 0 END,
		       cr.reason, cr.detail, cr.status,
		       cr.resolution_note, cr.resolved_at, cr.created_at, cr.updated_at
		FROM content_reports cr
		LEFT JOIN users u ON u.id = cr.reporter_user_id
		LEFT JOIN posts p ON cr.target_type = 'post' AND p.id = cr.target_id`
	args := []any{}
	if status != "" {
		query += " WHERE cr.status = ?"
		args = append(args, status)
	}
	query += ` ORDER BY CASE cr.status WHEN 'open' THEN 0 ELSE 1 END, cr.created_at DESC LIMIT 100`
	rows, err := h.query(c.Request.Context(), query, args...)
	if err != nil {
		fail(c, http.StatusInternalServerError, "reports_query_failed", "could not load content reports")
		return
	}
	defer rows.Close()
	reports := make([]domain.ContentReport, 0)
	for rows.Next() {
		report, err := scanAdminContentReport(rows)
		if err != nil {
			fail(c, http.StatusInternalServerError, "reports_scan_failed", "could not read content reports")
			return
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		fail(c, http.StatusInternalServerError, "reports_scan_failed", "could not read content reports")
		return
	}
	ok(c, envelope{"reports": reports})
}

func (h *AdminHandler) ModerateReport(c *gin.Context) {
	reportID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || reportID <= 0 {
		fail(c, http.StatusBadRequest, "invalid_report_id", "report id is invalid")
		return
	}
	var input AdminModerationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	report, err := h.getContentReport(c.Request.Context(), reportID)
	if err != nil {
		failNotFoundOrInternal(c, err, "report")
		return
	}
	if report.TargetType != "post" {
		fail(c, http.StatusBadRequest, "unsupported_report_target", "report target is not supported")
		return
	}

	now := nowString()
	status := "dismissed"
	auditAction := "dismiss_report"
	detail := fmt.Sprintf("管理员驳回举报 #%d：%s", report.ID, strings.TrimSpace(input.Note))
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		fail(c, http.StatusInternalServerError, "moderation_failed", "could not start moderation")
		return
	}
	defer func() { _ = tx.Rollback() }()
	if input.Action == "hide" {
		status = "actioned"
		auditAction = "hide_post"
		detail = fmt.Sprintf("管理员因举报 #%d 隐藏帖子 #%d：%s", report.ID, report.TargetID, strings.TrimSpace(input.Note))
		if _, err := tx.ExecContext(c.Request.Context(), bindDatabaseQuery(h.cfg.DatabaseDriver, `UPDATE posts SET deleted_at = COALESCE(deleted_at, ?), updated_at = ? WHERE id = ?`), now, now, report.TargetID); err != nil {
			fail(c, http.StatusInternalServerError, "moderation_failed", "could not hide post")
			return
		}
	} else if input.Action == "restore" {
		status = "actioned"
		auditAction = "restore_post"
		detail = fmt.Sprintf("管理员因举报 #%d 恢复帖子 #%d：%s", report.ID, report.TargetID, strings.TrimSpace(input.Note))
		if _, err := tx.ExecContext(c.Request.Context(), bindDatabaseQuery(h.cfg.DatabaseDriver, `UPDATE posts SET deleted_at = NULL, updated_at = ? WHERE id = ?`), now, report.TargetID); err != nil {
			fail(c, http.StatusInternalServerError, "moderation_failed", "could not restore post")
			return
		}
	}
	result, err := tx.ExecContext(c.Request.Context(), bindDatabaseQuery(h.cfg.DatabaseDriver, `
		UPDATE content_reports
		SET status = ?, resolution_note = ?, resolved_at = ?, updated_at = ?
		WHERE id = ?
	`), status, strings.TrimSpace(input.Note), now, now, report.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "moderation_failed", "could not update report")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		fail(c, http.StatusNotFound, "report_not_found", "report was not found")
		return
	}
	if err := tx.Commit(); err != nil {
		fail(c, http.StatusInternalServerError, "moderation_failed", "could not commit moderation")
		return
	}
	h.logAudit(c.Request.Context(), auditAction, fmt.Sprintf("report-%d", report.ID), "moderation", detail)
	updated, err := h.getContentReport(c.Request.Context(), report.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "moderation_failed", "could not reload report")
		return
	}
	ok(c, updated)
}

func (h *AdminHandler) upsertContent(ctx context.Context, input AdminContentInput, updateOnly bool) (AdminContentRecord, error) {
	now := nowString()
	if updateOnly {
		result, err := h.exec(ctx, `
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

	if _, err := h.exec(ctx, `
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
	rows, err := h.query(ctx, `
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
		if record.Module == "users" {
			continue
		}
		if publishedOnly && !isPublishedContentStatus(record.Status) {
			continue
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if !publishedOnly {
		users, err := h.listUserRecords(ctx)
		if err != nil {
			return nil, err
		}
		records = append(records, users...)
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

func isPublishedContentStatus(status string) bool {
	return status == "已上架" || status == "正常"
}

func (h *AdminHandler) listUserRecords(ctx context.Context) ([]AdminContentRecord, error) {
	rows, err := h.query(ctx, `
		SELECT u.id, COALESCE(u.email, ''), u.nickname, u.role, u.province, u.grade,
		       COALESCE(u.password_hash, ''), COALESCE(CAST(u.banned_at AS TEXT), ''), COALESCE(u.banned_reason, ''),
		       CAST(u.created_at AS TEXT), CAST(u.updated_at AS TEXT), COUNT(p.id)
		FROM users u
		LEFT JOIN posts p ON p.user_id = u.id AND p.deleted_at IS NULL
		WHERE u.deleted_at IS NULL
		GROUP BY u.id, u.email, u.nickname, u.role, u.province, u.grade, u.password_hash, u.banned_at, u.banned_reason, u.created_at, u.updated_at
		ORDER BY u.created_at DESC, u.id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]AdminContentRecord, 0)
	for rows.Next() {
		var id int64
		var email, nickname, role, province, grade, passwordHash, bannedAt, bannedReason, createdAt, updatedAt string
		var postCount int
		if err := rows.Scan(&id, &email, &nickname, &role, &province, &grade, &passwordHash, &bannedAt, &bannedReason, &createdAt, &updatedAt, &postCount); err != nil {
			return nil, err
		}
		status := "正常"
		if bannedAt != "" {
			status = "已封禁"
		}
		payload := marshalJSON(envelope{
			"userId":             id,
			"email":              email,
			"grade":              grade,
			"postCount":          postCount,
			"passwordConfigured": passwordHash != "",
			"bannedAt":           bannedAt,
			"bannedReason":       bannedReason,
		})
		records = append(records, AdminContentRecord{
			ID:        fmt.Sprintf("user-%d", id),
			Module:    "users",
			Title:     nickname,
			Type:      adminRoleLabel(role),
			Status:    status,
			Scope:     province,
			Owner:     "真实站内账号",
			Tags:      cleanStringSlice([]string{grade, status, "已设置密码"}),
			Summary:   fmt.Sprintf("%s · 已发布 %d 篇帖子", email, postCount),
			Priority:  "常规",
			Payload:   json.RawMessage(payload),
			CreatedAt: parseSQLiteTime(createdAt),
			UpdatedAt: parseSQLiteTime(updatedAt),
		})
	}
	return records, rows.Err()
}

func adminRoleLabel(role string) string {
	switch role {
	case "student":
		return "学生"
	case "parent":
		return "家长"
	case "teacher":
		return "老师"
	case "counselor":
		return "规划师"
	default:
		return role
	}
}

func (h *AdminHandler) getContentByID(ctx context.Context, id string) (AdminContentRecord, error) {
	return scanAdminContent(h.queryRow(ctx, `
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

func detectImageUpload(fileHeader *multipart.FileHeader) (string, string, int, int, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("could not read uploaded image")
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", "", 0, 0, fmt.Errorf("could not inspect uploaded image")
	}
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return "", "", 0, 0, fmt.Errorf("could not inspect uploaded image")
		}
	}
	imageConfig, _, err := image.DecodeConfig(file)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("could not decode image dimensions")
	}

	if n >= 12 && string(buffer[0:4]) == "RIFF" && string(buffer[8:12]) == "WEBP" {
		return "image/webp", ".webp", imageConfig.Width, imageConfig.Height, nil
	}

	switch contentType := http.DetectContentType(buffer[:n]); contentType {
	case "image/jpeg":
		return contentType, ".jpg", imageConfig.Width, imageConfig.Height, nil
	case "image/png":
		return contentType, ".png", imageConfig.Width, imageConfig.Height, nil
	case "image/gif":
		return contentType, ".gif", imageConfig.Width, imageConfig.Height, nil
	case "image/webp":
		return contentType, ".webp", imageConfig.Width, imageConfig.Height, nil
	default:
		return "", "", 0, 0, fmt.Errorf("only JPG, PNG, GIF, and WebP images are supported")
	}
}

func saveUploadedFile(fileHeader *multipart.FileHeader, targetPath string) error {
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
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

func (h *AdminHandler) getContentReport(ctx context.Context, id int64) (domain.ContentReport, error) {
	return scanAdminContentReport(h.queryRow(ctx, `
		SELECT cr.id, cr.reporter_user_id, COALESCE(u.nickname, ''), cr.target_type, cr.target_id,
		       COALESCE(p.title, ''), COALESCE(p.author_name, ''),
		       CASE WHEN p.deleted_at IS NOT NULL THEN 1 ELSE 0 END,
		       cr.reason, cr.detail, cr.status,
		       cr.resolution_note, cr.resolved_at, cr.created_at, cr.updated_at
		FROM content_reports cr
		LEFT JOIN users u ON u.id = cr.reporter_user_id
		LEFT JOIN posts p ON cr.target_type = 'post' AND p.id = cr.target_id
		WHERE cr.id = ?
	`, id))
}

func scanAdminContentReport(scanner adminContentScanner) (domain.ContentReport, error) {
	var report domain.ContentReport
	var resolvedAt sql.NullString
	var createdAt string
	var updatedAt string
	if err := scanner.Scan(
		&report.ID,
		&report.ReporterID,
		&report.ReporterName,
		&report.TargetType,
		&report.TargetID,
		&report.TargetTitle,
		&report.TargetAuthor,
		&report.TargetHidden,
		&report.Reason,
		&report.Detail,
		&report.Status,
		&report.ResolutionNote,
		&resolvedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.ContentReport{}, err
	}
	if resolvedAt.Valid {
		parsed := parseSQLiteTime(resolvedAt.String)
		report.ResolvedAt = &parsed
	}
	report.CreatedAt = parseSQLiteTime(createdAt)
	report.UpdatedAt = parseSQLiteTime(updatedAt)
	return report, nil
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
	actor := "admin"
	if principal, ok := middleware.AdminPrincipalFromContext(ctx); ok {
		actor = principal.Email + " (" + principal.Role + ")"
	}
	_, _ = h.exec(ctx, `
		INSERT INTO admin_audit_logs (action, record_id, module, detail, actor, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, action, recordID, module, detail, actor, nowString())
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
