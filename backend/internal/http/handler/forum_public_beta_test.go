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
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/http/middleware"
	"subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

type forumHandlerHarness struct {
	db      *sql.DB
	router  *gin.Engine
	users   map[string]domain.User
	handler *ForumHandler
}

func newForumHandlerHarness(t *testing.T) *forumHandlerHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tempDir := t.TempDir()
	cfg := config.Config{
		SQLitePath:     filepath.Join(tempDir, "forum-handler.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
		JWTSecret:      "handler-test-secret",
	}
	db, err := storage.NewSQLiteDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	users := map[string]domain.User{
		"owner": seedForumHandlerUser(t, db, "owner@example.com", "发帖学生"),
		"peer":  seedForumHandlerUser(t, db, "peer@example.com", "同伴学生"),
	}
	forumService := service.NewForumService(sqlite.NewForumRepository(db), cfg, nil)
	forumHandler := NewForumHandler(forumService, nil, false, cfg.MediaUploadDir, "")
	router := gin.New()
	router.Use(func(c *gin.Context) {
		name := c.GetHeader("X-Test-User")
		if user, ok := users[name]; ok {
			c.Set(middleware.CurrentUserKey, user)
		}
		c.Next()
	})
	router.POST("/posts", forumHandler.CreatePost)
	router.PUT("/posts/:id", forumHandler.UpdatePost)
	router.DELETE("/posts/:id", forumHandler.DeletePost)
	router.POST("/posts/:id/reports", forumHandler.ReportPost)
	router.POST("/messages", forumHandler.SendDirectMessage)
	router.GET("/messages/:name", forumHandler.ListDirectMessages)
	router.GET("/conversations", forumHandler.ListConversations)
	router.GET("/notifications", forumHandler.ListNotifications)
	router.POST("/notifications/:id/read", forumHandler.MarkNotificationRead)
	router.POST("/notifications/read-all", forumHandler.MarkAllNotificationsRead)
	router.GET("/taxonomy", forumHandler.Taxonomy)
	router.GET("/insights", forumHandler.ListInsights)
	router.GET("/insights/:id", forumHandler.GetInsight)
	router.GET("/topics", forumHandler.ListTopics)
	router.GET("/topics/:slug", forumHandler.GetTopic)
	router.GET("/posts", forumHandler.ListPosts)
	router.GET("/posts/:id", forumHandler.GetPost)
	router.POST("/posts/:id/comments", forumHandler.CreateComment)
	router.POST("/posts/:id/like", forumHandler.TogglePostLike)
	router.POST("/posts/:id/favorite", forumHandler.TogglePostFavorite)
	router.POST("/authors/:name/follow", forumHandler.ToggleFollowAuthor)

	return &forumHandlerHarness{db: db, router: router, users: users, handler: forumHandler}
}

func TestForumHandlerAuthenticatedPostLifecycleAndAuthorization(t *testing.T) {
	h := newForumHandlerHarness(t)
	initialPosts := queryRowCount(t, h.db, "SELECT COUNT(*) FROM posts")

	invalidJSON := h.request(http.MethodPost, "/posts", "owner", `{`)
	assertHandlerError(t, invalidJSON, http.StatusBadRequest, "invalid_payload")
	invalidElectives := h.request(http.MethodPost, "/posts", "owner", postPayload("无效选科帖子", "chemistry", "chemistry"))
	assertHandlerError(t, invalidElectives, http.StatusBadRequest, "invalid_electives")
	invalidImages := h.request(http.MethodPost, "/posts", "owner", strings.Replace(postPayload("伪造图片帖子", "chemistry", "biology"), `"imageUrls":[]`, `"imageUrls":["https://evil.example/image.png"]`, 1))
	assertHandlerError(t, invalidImages, http.StatusBadRequest, "invalid_images")
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM posts", initialPosts)

	created := h.request(http.MethodPost, "/posts", "owner", postPayload("公测发帖流程", "chemistry", "biology"))
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	postID := responseDataID(t, created)
	var title, content, electives string
	var ownerID int64
	if err := h.db.QueryRow("SELECT user_id, title, content, electives FROM posts WHERE id = ?", postID).Scan(&ownerID, &title, &content, &electives); err != nil {
		t.Fatal(err)
	}
	if ownerID != h.users["owner"].ID || title != "公测发帖流程" || !strings.Contains(content, "数据库副作用") || !strings.Contains(electives, "biology") {
		t.Fatalf("unexpected stored post owner=%d title=%q content=%q electives=%s", ownerID, title, content, electives)
	}

	unauthorizedUpdate := h.request(http.MethodPut, pathID("/posts/", postID), "peer", updatePostPayload("越权篡改"))
	assertHandlerError(t, unauthorizedUpdate, http.StatusNotFound, "not_found")
	assertScalarString(t, h.db, "SELECT title FROM posts WHERE id = ?", postID, "公测发帖流程")

	updated := h.request(http.MethodPut, pathID("/posts/", postID), "owner", updatePostPayload("编辑后的帖子"))
	if updated.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", updated.Code, updated.Body.String())
	}
	assertScalarString(t, h.db, "SELECT title FROM posts WHERE id = ?", postID, "编辑后的帖子")
	invalidUpdate := h.request(http.MethodPut, pathID("/posts/", postID), "owner", strings.Replace(updatePostPayload("不应保存的编辑"), `["politics","geography"]`, `["politics","politics"]`, 1))
	assertHandlerError(t, invalidUpdate, http.StatusBadRequest, "invalid_electives")
	assertScalarString(t, h.db, "SELECT title FROM posts WHERE id = ?", postID, "编辑后的帖子")

	invalidID := h.request(http.MethodDelete, "/posts/nope", "owner", "")
	assertHandlerError(t, invalidID, http.StatusBadRequest, "invalid_id")
	unauthorizedDelete := h.request(http.MethodDelete, pathID("/posts/", postID), "peer", "")
	assertHandlerError(t, unauthorizedDelete, http.StatusNotFound, "not_found")
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM posts WHERE id = ? AND deleted_at IS NULL", 1, postID)

	deleted := h.request(http.MethodDelete, pathID("/posts/", postID), "owner", "")
	if deleted.Code != http.StatusOK || !strings.Contains(deleted.Body.String(), `"deleted":true`) {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM posts WHERE id = ? AND deleted_at IS NOT NULL", 1, postID)
}

func TestForumHandlerReportPersistsAndRejectsInvalidRequests(t *testing.T) {
	h := newForumHandlerHarness(t)
	postID := createForumHandlerPost(t, h, "owner")

	invalid := h.request(http.MethodPost, pathID("/posts/", postID)+"/reports", "peer", `{"reason":"   ","detail":"ignored"}`)
	assertHandlerError(t, invalid, http.StatusBadRequest, "invalid_report")
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM content_reports", 0)

	reported := h.request(http.MethodPost, pathID("/posts/", postID)+"/reports", "peer", `{"reason":"spam","detail":"重复推广内容"}`)
	if reported.Code != http.StatusCreated {
		t.Fatalf("report status=%d body=%s", reported.Code, reported.Body.String())
	}
	var reporterID, targetID int64
	var reason, detail, status string
	if err := h.db.QueryRow("SELECT reporter_user_id, target_id, reason, detail, status FROM content_reports").Scan(&reporterID, &targetID, &reason, &detail, &status); err != nil {
		t.Fatal(err)
	}
	if reporterID != h.users["peer"].ID || targetID != postID || reason != "spam" || detail != "重复推广内容" || status != "open" {
		t.Fatalf("unexpected report reporter=%d target=%d reason=%q detail=%q status=%q", reporterID, targetID, reason, detail, status)
	}

	missing := h.request(http.MethodPost, "/posts/999999/reports", "peer", `{"reason":"spam"}`)
	assertHandlerError(t, missing, http.StatusNotFound, "not_found")
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM content_reports", 1)
}

func TestForumHandlerMessagesPersistAndNotificationReadFlows(t *testing.T) {
	h := newForumHandlerHarness(t)
	initialMessages := queryRowCount(t, h.db, "SELECT COUNT(*) FROM direct_messages")

	blank := h.request(http.MethodPost, "/messages", "owner", `{"recipientName":"同伴学生","content":"   "}`)
	assertHandlerError(t, blank, http.StatusBadRequest, "message_failed")
	missing := h.request(http.MethodPost, "/messages", "owner", `{"recipientName":"不存在用户","content":"你好"}`)
	assertHandlerError(t, missing, http.StatusNotFound, "not_found")
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM direct_messages", initialMessages)

	sent := h.request(http.MethodPost, "/messages", "owner", `{"recipientName":"同伴学生","content":"  明天公测见  "}`)
	if sent.Code != http.StatusCreated {
		t.Fatalf("send status=%d body=%s", sent.Code, sent.Body.String())
	}
	var messageID int64
	var senderID, recipientID int64
	var content string
	if err := h.db.QueryRow("SELECT id, sender_user_id, recipient_user_id, content FROM direct_messages ORDER BY id DESC LIMIT 1").Scan(&messageID, &senderID, &recipientID, &content); err != nil {
		t.Fatal(err)
	}
	if senderID != h.users["owner"].ID || recipientID != h.users["peer"].ID || content != "明天公测见" {
		t.Fatalf("unexpected message sender=%d recipient=%d content=%q", senderID, recipientID, content)
	}
	var notificationID int64
	var notificationReadAt sql.NullString
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := h.db.Exec(`INSERT INTO notifications (recipient_user_id, actor_user_id, type, title, summary, target_url, created_at) VALUES (?, ?, 'message', '新私信', '明天公测见', '/messages', ?)`, recipientID, senderID, now)
	if err != nil {
		t.Fatal(err)
	}
	notificationID, err = result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if err := h.db.QueryRow("SELECT read_at FROM notifications WHERE id = ?", notificationID).Scan(&notificationReadAt); err != nil {
		t.Fatal(err)
	}
	if notificationReadAt.Valid {
		t.Fatal("new message notification unexpectedly marked read")
	}

	messages := h.request(http.MethodGet, "/messages/发帖学生?limit=1", "peer", "")
	if messages.Code != http.StatusOK || !strings.Contains(messages.Body.String(), "明天公测见") {
		t.Fatalf("messages status=%d body=%s", messages.Code, messages.Body.String())
	}
	badMessageCursor := h.request(http.MethodGet, "/messages/发帖学生?cursor=broken", "peer", "")
	assertHandlerError(t, badMessageCursor, http.StatusBadRequest, "invalid_cursor")
	conversations := h.request(http.MethodGet, "/conversations", "peer", "")
	if conversations.Code != http.StatusOK || !strings.Contains(conversations.Body.String(), "发帖学生") {
		t.Fatalf("conversations status=%d body=%s", conversations.Code, conversations.Body.String())
	}
	badConversationCursor := h.request(http.MethodGet, "/conversations?cursor=broken", "peer", "")
	assertHandlerError(t, badConversationCursor, http.StatusBadRequest, "invalid_cursor")

	notifications := h.request(http.MethodGet, "/notifications?limit=1", "peer", "")
	if notifications.Code != http.StatusOK || !strings.Contains(notifications.Body.String(), "明天公测见") {
		t.Fatalf("notifications status=%d body=%s", notifications.Code, notifications.Body.String())
	}
	badNotificationCursor := h.request(http.MethodGet, "/notifications?cursor=broken", "peer", "")
	assertHandlerError(t, badNotificationCursor, http.StatusBadRequest, "invalid_cursor")

	marked := h.request(http.MethodPost, pathID("/notifications/", notificationID)+"/read", "peer", "")
	if marked.Code != http.StatusOK {
		t.Fatalf("mark notification status=%d body=%s", marked.Code, marked.Body.String())
	}
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM notifications WHERE id = ? AND read_at IS NOT NULL", 1, notificationID)

	if _, err := h.db.Exec("UPDATE notifications SET read_at = NULL WHERE id = ?", notificationID); err != nil {
		t.Fatal(err)
	}
	markedAll := h.request(http.MethodPost, "/notifications/read-all", "peer", "")
	if markedAll.Code != http.StatusOK {
		t.Fatalf("mark all status=%d body=%s", markedAll.Code, markedAll.Body.String())
	}
	assertRowCount(t, h.db, "SELECT COUNT(*) FROM notifications WHERE recipient_user_id = ? AND read_at IS NULL", 0, recipientID)

	invalidID := h.request(http.MethodPost, "/notifications/zero/read", "peer", "")
	assertHandlerError(t, invalidID, http.StatusBadRequest, "invalid_id")
	_ = messageID
}

func (h *forumHandlerHarness) request(method, path, user, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("X-Test-User", user)
	h.router.ServeHTTP(recorder, request)
	return recorder
}

func seedForumHandlerUser(t *testing.T, db *sql.DB, email, nickname string) domain.User {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := db.Exec(`INSERT INTO users (email, password_hash, nickname, role, province, grade, created_at, updated_at) VALUES (?, ?, ?, 'student', '广东', '高一', ?, ?)`, email, "test-hash", nickname, now, now)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return domain.User{ID: id, Email: email, Nickname: nickname, Role: "student", Province: "广东", Grade: "高一"}
}

func createForumHandlerPost(t *testing.T, h *forumHandlerHarness, user string) int64 {
	t.Helper()
	response := h.request(http.MethodPost, "/posts", user, postPayload("待举报公测帖子", "chemistry", "biology"))
	if response.Code != http.StatusCreated {
		t.Fatalf("seed post status=%d body=%s", response.Code, response.Body.String())
	}
	return responseDataID(t, response)
}

func postPayload(title, first, second string) string {
	return `{"title":"` + title + `","content":"用于验证 handler 数据库副作用的完整正文","track":"physics","electives":["` + first + `","` + second + `"],"category":"question","grade":"高一","province":"广东","imageUrls":[]}`
}

func updatePostPayload(title string) string {
	return `{"title":"` + title + `","content":"编辑后仍然满足长度要求的正文内容","track":"history","electives":["politics","geography"],"category":"experience"}`
}

func responseDataID(t *testing.T, recorder *httptest.ResponseRecorder) int64 {
	t.Helper()
	var body struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.ID <= 0 {
		t.Fatalf("missing response data id: %s", recorder.Body.String())
	}
	return body.Data.ID
}

func assertHandlerError(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status || !strings.Contains(recorder.Body.String(), `"code":"`+code+`"`) {
		t.Fatalf("status=%d want=%d code=%q body=%s", recorder.Code, status, code, recorder.Body.String())
	}
}

func assertRowCount(t *testing.T, db *sql.DB, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := db.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("row count=%d want=%d query=%q", got, want, query)
	}
}

func queryRowCount(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var count int
	if err := db.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func assertScalarString(t *testing.T, db *sql.DB, query string, arg any, want string) {
	t.Helper()
	var got string
	if err := db.QueryRow(query, arg).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("value=%q want=%q query=%q", got, want, query)
	}
}

func pathID(prefix string, id int64) string {
	return prefix + stringInt64ForHandlerTest(id)
}
