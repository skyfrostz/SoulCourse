package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/http/middleware"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

type ForumHandler struct {
	service        *service.ForumService
	ai             *service.AIService
	secureCookies  bool
	mediaUploadDir string
	appBasePath    string
	objectStore    storage.ObjectStore
}

func NewForumHandler(forumService *service.ForumService, aiService *service.AIService, secureCookies bool, mediaUploadDir string, appBasePath string) *ForumHandler {
	store, _ := storage.NewLocalObjectStore(mediaUploadDir, appBasePath)
	return NewForumHandlerWithObjectStore(forumService, aiService, secureCookies, mediaUploadDir, appBasePath, store)
}

func NewForumHandlerWithObjectStore(forumService *service.ForumService, aiService *service.AIService, secureCookies bool, mediaUploadDir, appBasePath string, store storage.ObjectStore) *ForumHandler {
	return &ForumHandler{service: forumService, ai: aiService, secureCookies: secureCookies, mediaUploadDir: mediaUploadDir, appBasePath: appBasePath, objectStore: store}
}

func (h *ForumHandler) SendEmailVerificationCode(c *gin.Context) {
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
		fail(c, http.StatusInternalServerError, "email_send_failed", "could not send verification code")
		return
	}
	ok(c, result)
}

func (h *ForumHandler) ForgotPassword(c *gin.Context) {
	h.SendEmailVerificationCode(c)
}

func requestRemoteIP(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(request.RemoteAddr)
}

func (h *ForumHandler) setAuthCookies(c *gin.Context, session domain.AuthSession) {
	maxAge := int(time.Until(session.ExpiresAt).Seconds())
	if maxAge <= 0 {
		maxAge = 1
	}
	middleware.SetSessionCookie(c, session.Token, maxAge, h.secureCookies)
	csrfToken, err := generateCookieToken()
	if err == nil {
		middleware.SetCSRFCookie(c, csrfToken, maxAge, h.secureCookies)
	}
}

func generateCookieToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}

func (h *ForumHandler) routePath(relativePath string) string {
	if h.appBasePath == "" {
		return relativePath
	}
	return h.appBasePath + relativePath
}

func inspectStoredImage(path string) (int64, string, int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", 0, 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, "", 0, 0, err
	}
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return 0, "", 0, 0, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return 0, "", 0, 0, err
	}
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return 0, "", 0, 0, err
	}
	contentType := http.DetectContentType(buffer[:n])
	if format == "jpg" || format == "jpeg" {
		contentType = "image/jpeg"
	}
	return info.Size(), contentType, config.Width, config.Height, nil
}

func inspectStoredImageReader(reader io.Reader, size int64) (int64, string, int, int, error) {
	data, err := io.ReadAll(io.LimitReader(reader, service.MaxImageUploadBytes()+1))
	if err != nil || int64(len(data)) != size || int64(len(data)) > service.MaxImageUploadBytes() {
		return 0, "", 0, 0, errors.New("invalid object size")
	}
	config, format, err := image.DecodeConfig(strings.NewReader(string(data)))
	if err != nil {
		return 0, "", 0, 0, err
	}
	contentType := http.DetectContentType(data)
	if format == "jpg" || format == "jpeg" {
		contentType = "image/jpeg"
	}
	return size, contentType, config.Width, config.Height, nil
}

func handleEmailVerificationRateLimit(c *gin.Context, err error) bool {
	var rateLimitError *service.EmailVerificationRateLimitError
	if !errors.As(err, &rateLimitError) {
		return false
	}
	limit := rateLimitError.Limit
	c.Header("Retry-After", strconv.Itoa(limit.RetryAfterSeconds))
	failWithDetails(c, http.StatusTooManyRequests, "email_verification_rate_limited", "too many verification code requests", envelope{
		"retryAfterSeconds": limit.RetryAfterSeconds,
		"hourlyLimit":       limit.EmailHourlyLimit,
		"hourlyRemaining":   limit.EmailHourlyRemaining,
	})
	return true
}

func (h *ForumHandler) Register(c *gin.Context) {
	var input domain.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}

	session, err := h.service.Register(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidEmailVerificationCode) {
			fail(c, http.StatusBadRequest, "invalid_verification_code", "verification code is invalid or expired")
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			fail(c, http.StatusConflict, "email_exists", "email already registered")
			return
		}
		fail(c, http.StatusInternalServerError, "internal_error", "could not register")
		return
	}
	h.setAuthCookies(c, session)
	created(c, session)
}

func (h *ForumHandler) Login(c *gin.Context) {
	var input domain.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}

	session, err := h.service.Login(c.Request.Context(), input)
	if err != nil {
		fail(c, http.StatusUnauthorized, "invalid_credentials", "email or password is incorrect")
		return
	}
	h.setAuthCookies(c, session)
	ok(c, session)
}

func (h *ForumHandler) ResetPassword(c *gin.Context) {
	var input domain.ResetPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), input); err != nil {
		if errors.Is(err, service.ErrInvalidEmailVerificationCode) {
			fail(c, http.StatusBadRequest, "invalid_verification_code", "verification code is invalid or expired")
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			fail(c, http.StatusNotFound, "not_found", "account not found")
			return
		}
		fail(c, http.StatusInternalServerError, "internal_error", "could not reset password")
		return
	}
	middleware.ClearSessionCookie(c, h.secureCookies)
	middleware.ClearCSRFCookie(c, h.secureCookies)
	ok(c, gin.H{"reset": true})
}

func (h *ForumHandler) Logout(c *gin.Context) {
	token, _ := middleware.SessionToken(c)
	if err := h.service.Logout(c.Request.Context(), token); err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "could not logout")
		return
	}
	middleware.ClearSessionCookie(c, h.secureCookies)
	middleware.ClearCSRFCookie(c, h.secureCookies)
	ok(c, gin.H{"signedOut": true})
}

func (h *ForumHandler) Me(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	ok(c, user)
}

func (h *ForumHandler) DeleteMe(c *gin.Context) {
	var input domain.DeleteAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	user, _ := middleware.CurrentUser(c)
	if err := h.service.DeleteAccount(c.Request.Context(), user.ID, input); err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			fail(c, http.StatusUnauthorized, "invalid_credentials", "password is incorrect")
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			fail(c, http.StatusNotFound, "not_found", "account not found")
			return
		}
		fail(c, http.StatusInternalServerError, "internal_error", "could not delete account")
		return
	}
	middleware.ClearSessionCookie(c, h.secureCookies)
	middleware.ClearCSRFCookie(c, h.secureCookies)
	ok(c, gin.H{"deleted": true})
}

func (h *ForumHandler) PresignImageUpload(c *gin.Context) {
	var input domain.PresignImageUploadInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	user, _ := middleware.CurrentUser(c)
	uploadURLFactory := func(id string) string {
		return h.routePath("/api/v1/uploads/images/" + id + "/object")
	}
	result, err := h.service.PresignImageUpload(c.Request.Context(), user.ID, input, "")
	if err != nil {
		if errors.Is(err, service.ErrInvalidUpload) {
			fail(c, http.StatusBadRequest, "invalid_upload", "image upload metadata is invalid")
			return
		}
		fail(c, http.StatusInternalServerError, "internal_error", "could not create upload")
		return
	}
	if h.objectStore == nil {
		result.UploadURL = uploadURLFactory(result.ID)
	} else if _, local := h.objectStore.(*storage.LocalObjectStore); local {
		result.UploadURL = uploadURLFactory(result.ID)
	} else {
		result.UploadURL, err = h.objectStore.PresignPut(c.Request.Context(), result.AssetKey, result.ContentType, 15*time.Minute)
		if err != nil {
			fail(c, http.StatusInternalServerError, "storage_unavailable", "could not create upload URL")
			return
		}
	}
	if strings.TrimSpace(result.UploadURL) == "" {
		fail(c, http.StatusInternalServerError, "internal_error", "could not create upload URL")
		return
	}
	created(c, result)
}

func (h *ForumHandler) PutImageUploadObject(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	record, err := h.service.GetImageUpload(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		failNotFoundOrInternal(c, err, "upload")
		return
	}
	if c.GetHeader("Content-Type") != record.ContentType {
		fail(c, http.StatusBadRequest, "invalid_content_type", "content type does not match upload metadata")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxImageUploadBytes()+1)
	if err := h.objectStore.Put(c.Request.Context(), record.AssetKey, record.ContentType, c.Request.Body, record.SizeBytes); err != nil {
		fail(c, http.StatusInternalServerError, "upload_save_failed", "could not save uploaded image")
		return
	}
	ok(c, gin.H{"uploaded": true})
}

func (h *ForumHandler) CompleteImageUpload(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	record, err := h.service.GetImageUploadForCompletion(c.Request.Context(), user.ID, c.Param("id"))
	if err != nil {
		failNotFoundOrInternal(c, err, "upload")
		return
	}
	body, objectInfo, err := h.objectStore.Open(c.Request.Context(), record.AssetKey)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid_upload_object", "uploaded object is missing")
		return
	}
	defer body.Close()
	sizeBytes, contentType, width, height, err := inspectStoredImageReader(body, objectInfo.Size)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid_upload_object", "uploaded object is not a valid image")
		return
	}
	result, err := h.service.CompleteImageUpload(c.Request.Context(), user.ID, record.ID, sizeBytes, contentType, width, height)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUpload) {
			fail(c, http.StatusBadRequest, "invalid_upload", "uploaded object does not match upload metadata")
			return
		}
		fail(c, http.StatusInternalServerError, "internal_error", "could not complete upload")
		return
	}
	result.URL = h.objectStore.PublicURL(result.AssetKey)
	if result.URL == "" {
		result.URL = h.routePath(result.URL)
	}
	ok(c, result)
}

func (h *ForumHandler) GetProfile(c *gin.Context) {
	profile, err := h.service.GetAccountProfile(c.Request.Context(), middleware.CurrentUserID(c), c.Param("name"))
	if err != nil {
		failNotFoundOrInternal(c, err, "profile")
		return
	}
	ok(c, profile)
}

func (h *ForumHandler) GetMyProfile(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	profile, err := h.service.GetAccountProfileByUserID(c.Request.Context(), &user.ID, user.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "could not load profile")
		return
	}
	ok(c, profile)
}

func (h *ForumHandler) ListMySessions(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	token, _ := middleware.SessionToken(c)
	sessions, err := h.service.ListAuthSessions(c.Request.Context(), user.ID, token)
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "could not list sessions")
		return
	}
	ok(c, sessions)
}

func (h *ForumHandler) RevokeMySession(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	sessionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || sessionID <= 0 {
		fail(c, http.StatusBadRequest, "invalid_session", "session id is invalid")
		return
	}
	token, _ := middleware.SessionToken(c)
	sessions, err := h.service.ListAuthSessions(c.Request.Context(), user.ID, token)
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "could not load sessions")
		return
	}
	revokingCurrent := false
	for _, session := range sessions {
		if session.ID == sessionID {
			revokingCurrent = session.Current
			break
		}
	}
	if err := h.service.RevokeAuthSessionByID(c.Request.Context(), user.ID, sessionID); err != nil {
		failNotFoundOrInternal(c, err, "session")
		return
	}
	if revokingCurrent {
		middleware.ClearSessionCookie(c, h.secureCookies)
		middleware.ClearCSRFCookie(c, h.secureCookies)
	}
	ok(c, gin.H{"revoked": true})
}

func (h *ForumHandler) UpdateMyProfile(c *gin.Context) {
	var input domain.UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	user, _ := middleware.CurrentUser(c)
	profile, err := h.service.UpdateAccountProfile(c.Request.Context(), user.ID, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidElectives) {
			fail(c, http.StatusBadRequest, "invalid_electives", err.Error())
			return
		}
		fail(c, http.StatusInternalServerError, "internal_error", "could not update profile")
		return
	}
	ok(c, profile)
}

func (h *ForumHandler) ListNotifications(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	limit := parseLimit(c, 30, 100)
	page, err := h.service.ListNotifications(c.Request.Context(), user.ID, limit, c.Query("cursor"))
	if err != nil {
		if strings.Contains(err.Error(), "invalid notification cursor") {
			fail(c, http.StatusBadRequest, "invalid_cursor", "invalid notification cursor")
			return
		}
		fail(c, http.StatusInternalServerError, "internal_error", "could not list notifications")
		return
	}
	okWithMeta(c, page.Items, envelope{"nextCursor": page.NextCursor, "hasMore": page.HasMore})
}

func (h *ForumHandler) MarkNotificationRead(c *gin.Context) {
	id, okID := parseID(c, "id")
	if !okID {
		return
	}
	user, _ := middleware.CurrentUser(c)
	if err := h.service.MarkNotificationRead(c.Request.Context(), user.ID, &id); err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "could not update notification")
		return
	}
	ok(c, envelope{"read": true})
}

func (h *ForumHandler) MarkAllNotificationsRead(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	if err := h.service.MarkNotificationRead(c.Request.Context(), user.ID, nil); err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "could not update notifications")
		return
	}
	ok(c, envelope{"read": true})
}

func (h *ForumHandler) ListConversations(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	page, err := h.service.ListConversations(c.Request.Context(), user.ID, parseLimit(c, 100, 100), c.Query("cursor"))
	if err != nil {
		if strings.Contains(err.Error(), "invalid conversation cursor") {
			fail(c, http.StatusBadRequest, "invalid_cursor", "invalid conversation cursor")
			return
		}
		fail(c, http.StatusInternalServerError, "internal_error", "could not list conversations")
		return
	}
	okWithMeta(c, page.Items, envelope{"nextCursor": page.NextCursor, "hasMore": page.HasMore})
}

func (h *ForumHandler) ListDirectMessages(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	page, err := h.service.ListDirectMessages(c.Request.Context(), user.ID, c.Param("name"), parseLimit(c, 50, 100), c.Query("cursor"))
	if err != nil {
		if strings.Contains(err.Error(), "invalid notification cursor") {
			fail(c, http.StatusBadRequest, "invalid_cursor", "invalid message cursor")
			return
		}
		failNotFoundOrInternal(c, err, "user")
		return
	}
	okWithMeta(c, page.Items, envelope{"nextCursor": page.NextCursor, "hasMore": page.HasMore})
}

func (h *ForumHandler) SendDirectMessage(c *gin.Context) {
	var input domain.SendMessageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	user, _ := middleware.CurrentUser(c)
	item, err := h.service.SendDirectMessage(c.Request.Context(), user.ID, input)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(c, http.StatusNotFound, "not_found", "recipient not found")
			return
		}
		fail(c, http.StatusBadRequest, "message_failed", err.Error())
		return
	}
	created(c, item)
}

func (h *ForumHandler) Taxonomy(c *gin.Context) {
	ok(c, envelope{
		"tracks": []envelope{
			{"value": domain.TrackPhysics, "label": "物理方向"},
			{"value": domain.TrackHistory, "label": "历史方向"},
		},
		"subjects": []envelope{
			{"value": domain.SubjectChemistry, "label": "化学"},
			{"value": domain.SubjectBiology, "label": "生物"},
			{"value": domain.SubjectPolitics, "label": "政治"},
			{"value": domain.SubjectGeography, "label": "地理"},
		},
		"categories": []envelope{
			{"value": domain.CategoryExperience, "label": "经验帖"},
			{"value": domain.CategoryQuestion, "label": "家长提问"},
			{"value": domain.CategoryData, "label": "数据建议"},
		},
		"topicTags":   domain.TopicTags(),
		"subjectTags": domain.SubjectTags(),
	})
}

func (h *ForumHandler) ListInsights(c *gin.Context) {
	insights, err := h.service.ListInsights(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "could not list insights")
		return
	}
	ok(c, insights)
}

func (h *ForumHandler) GetInsight(c *gin.Context) {
	id, okID := parseID(c, "id")
	if !okID {
		return
	}
	insight, err := h.service.GetInsight(c.Request.Context(), id)
	if err != nil {
		failNotFoundOrInternal(c, err, "insight")
		return
	}
	ok(c, insight)
}

func (h *ForumHandler) ListTopics(c *gin.Context) {
	topics, err := h.service.ListTopics(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "could not list topics")
		return
	}
	ok(c, topics)
}

func (h *ForumHandler) GetTopic(c *gin.Context) {
	detail, err := h.service.GetTopic(c.Request.Context(), middleware.CurrentUserID(c), c.Param("slug"))
	if err != nil {
		failNotFoundOrInternal(c, err, "topic")
		return
	}
	ok(c, detail)
}

func (h *ForumHandler) ListPosts(c *gin.Context) {
	limit := parseLimit(c, 20, 50)

	postPage, err := h.service.ListPosts(c.Request.Context(), middleware.CurrentUserID(c), domain.FeedFilter{
		Track:    domain.SubjectTrack(c.Query("track")),
		Subject:  domain.Subject(c.Query("subject")),
		Subjects: parseSubjects(c.Query("subjects")),
		Tag:      c.Query("tag"),
		Category: domain.PostCategory(c.Query("category")),
		Province: c.Query("province"),
		Keyword:  c.Query("q"),
		Sort:     domain.FeedSort(c.DefaultQuery("sort", string(domain.SortRecommended))),
		Limit:    limit,
		Cursor:   c.Query("cursor"),
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "could not list posts")
		return
	}
	okWithMeta(c, postPage.Items, envelope{"nextCursor": postPage.NextCursor, "hasMore": postPage.HasMore})
}

func parseSubjects(value string) []domain.Subject {
	parts := strings.Split(value, ",")
	subjects := make([]domain.Subject, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			subjects = append(subjects, domain.Subject(item))
		}
	}
	return subjects
}

func (h *ForumHandler) GetPost(c *gin.Context) {
	id, okID := parseID(c, "id")
	if !okID {
		return
	}

	post, comments, err := h.service.GetPost(c.Request.Context(), middleware.CurrentUserID(c), id)
	if err != nil {
		failNotFoundOrInternal(c, err, "post")
		return
	}

	ok(c, envelope{
		"post":     post,
		"comments": comments,
	})
}

func (h *ForumHandler) CreatePost(c *gin.Context) {
	var input domain.CreatePostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}

	user, _ := middleware.CurrentUser(c)
	post, err := h.service.CreatePost(c.Request.Context(), user, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidElectives) {
			fail(c, http.StatusBadRequest, "invalid_electives", err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidPostImages) {
			fail(c, http.StatusBadRequest, "invalid_images", err.Error())
			return
		}
		fail(c, http.StatusInternalServerError, "internal_error", "could not create post")
		return
	}

	created(c, post)
}

func (h *ForumHandler) UpdatePost(c *gin.Context) {
	id, okID := parseID(c, "id")
	if !okID {
		return
	}
	var input domain.UpdatePostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	user, _ := middleware.CurrentUser(c)
	post, err := h.service.UpdatePost(c.Request.Context(), user.ID, id, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidElectives) {
			fail(c, http.StatusBadRequest, "invalid_electives", err.Error())
			return
		}
		failNotFoundOrInternal(c, err, "post")
		return
	}
	ok(c, post)
}

func (h *ForumHandler) DeletePost(c *gin.Context) {
	id, okID := parseID(c, "id")
	if !okID {
		return
	}
	user, _ := middleware.CurrentUser(c)
	if err := h.service.DeletePost(c.Request.Context(), user.ID, id); err != nil {
		failNotFoundOrInternal(c, err, "post")
		return
	}
	ok(c, envelope{"deleted": true})
}

func (h *ForumHandler) CreateComment(c *gin.Context) {
	id, okID := parseID(c, "id")
	if !okID {
		return
	}

	var input domain.CreateCommentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}

	user, _ := middleware.CurrentUser(c)
	comment, err := h.service.CreateComment(c.Request.Context(), user, id, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidMessage) {
			fail(c, http.StatusBadRequest, "invalid_comment", err.Error())
			return
		}
		failNotFoundOrInternal(c, err, "post")
		return
	}

	created(c, comment)
}

func (h *ForumHandler) ReportPost(c *gin.Context) {
	id, okID := parseID(c, "id")
	if !okID {
		return
	}
	var input domain.ReportPostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	user, _ := middleware.CurrentUser(c)
	report, err := h.service.ReportPost(c.Request.Context(), user, id, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidReport) {
			fail(c, http.StatusBadRequest, "invalid_report", err.Error())
			return
		}
		failNotFoundOrInternal(c, err, "post")
		return
	}
	created(c, report)
}

func (h *ForumHandler) TogglePostLike(c *gin.Context) {
	id, okID := parseID(c, "id")
	if !okID {
		return
	}
	user, _ := middleware.CurrentUser(c)
	result, err := h.service.TogglePostLike(c.Request.Context(), user.ID, id)
	if err != nil {
		failNotFoundOrInternal(c, err, "post")
		return
	}
	ok(c, result)
}

func (h *ForumHandler) TogglePostFavorite(c *gin.Context) {
	id, okID := parseID(c, "id")
	if !okID {
		return
	}
	user, _ := middleware.CurrentUser(c)
	result, err := h.service.TogglePostFavorite(c.Request.Context(), user.ID, id)
	if err != nil {
		failNotFoundOrInternal(c, err, "post")
		return
	}
	ok(c, result)
}

func (h *ForumHandler) ToggleFollowAuthor(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	active, err := h.service.ToggleFollowAuthor(c.Request.Context(), user.ID, c.Param("name"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fail(c, http.StatusNotFound, "not_found", "author not found")
			return
		}
		fail(c, http.StatusBadRequest, "follow_failed", err.Error())
		return
	}
	ok(c, envelope{"active": active})
}

func (h *ForumHandler) ChoiceAdvice(c *gin.Context) {
	var input domain.ChoiceAdviceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}

	advice, err := h.ai.ChoiceAdvice(c.Request.Context(), input)
	if err != nil {
		c.Header("X-AI-Fallback", "true")
		okWithMeta(c, advice, envelope{"degraded": true})
		return
	}
	ok(c, advice)
}

func parseID(c *gin.Context, param string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil || id <= 0 {
		fail(c, http.StatusBadRequest, "invalid_id", "id must be a positive integer")
		return 0, false
	}
	return id, true
}

func parseLimit(c *gin.Context, fallback int, maximum int) int {
	value, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(fallback)))
	if err != nil || value <= 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}
