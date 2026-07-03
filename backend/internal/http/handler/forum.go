package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/http/middleware"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

type ForumHandler struct {
	service *service.ForumService
	ai      *service.AIService
}

func NewForumHandler(forumService *service.ForumService, aiService *service.AIService) *ForumHandler {
	return &ForumHandler{service: forumService, ai: aiService}
}

func (h *ForumHandler) SendEmailVerificationCode(c *gin.Context) {
	var input domain.EmailVerificationCodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}

	result, err := h.service.SendEmailVerificationCode(c.Request.Context(), input)
	if err != nil {
		fail(c, http.StatusInternalServerError, "email_send_failed", "could not send verification code")
		return
	}
	ok(c, result)
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			fail(c, http.StatusConflict, "email_exists", "email already registered")
			return
		}
		fail(c, http.StatusInternalServerError, "internal_error", "could not register")
		return
	}
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
	ok(c, session)
}

func (h *ForumHandler) Me(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	ok(c, user)
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
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	posts, err := h.service.ListPosts(c.Request.Context(), middleware.CurrentUserID(c), domain.FeedFilter{
		Track:    domain.SubjectTrack(c.Query("track")),
		Subject:  domain.Subject(c.Query("subject")),
		Subjects: parseSubjects(c.Query("subjects")),
		Category: domain.PostCategory(c.Query("category")),
		Province: c.Query("province"),
		Keyword:  c.Query("q"),
		Sort:     domain.FeedSort(c.DefaultQuery("sort", string(domain.SortRecommended))),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "could not list posts")
		return
	}
	ok(c, posts)
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
		fail(c, http.StatusInternalServerError, "internal_error", "could not create post")
		return
	}

	created(c, post)
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
		failNotFoundOrInternal(c, err, "post")
		return
	}

	created(c, comment)
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
	if err != nil && !errors.Is(err, service.ErrAIDisabled) {
		c.Header("X-AI-Fallback", "true")
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
