package httpserver

import (
	"crypto/subtle"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/http/handler"
	"subject-choice-forum/backend/internal/http/middleware"
	"subject-choice-forum/backend/internal/logx"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewServer(
	cfg config.Config,
	logger *logx.Logger,
	db *sql.DB,
	forumService *service.ForumService,
) *http.Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(middleware.RecoveryLogger(logger))
	router.Use(middleware.RequestLogger(logger))
	router.Use(middleware.SecurityHeaders())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Admin-Token", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	healthHandler := handler.NewHealthHandler(db)
	aiService := service.NewAIService(cfg)
	forumHandler := handler.NewForumHandler(forumService, aiService)
	adminHandler := handler.NewAdminHandler(cfg, forumService, db)

	if cfg.AppBasePath != "" {
		redirectToBasePath := func(c *gin.Context) {
			target := cfg.AppBasePath + "/"
			if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
				target += "?" + rawQuery
			}
			c.Redirect(http.StatusPermanentRedirect, target)
		}
		router.GET("/", redirectToBasePath)
		router.HEAD("/", redirectToBasePath)
	}

	baseRouter := router.Group(cfg.AppBasePath)

	baseRouter.GET("/healthz", healthHandler.Live)
	baseRouter.GET("/readyz", healthHandler.Ready)
	baseRouter.Static("/uploads", cfg.MediaUploadDir)

	api := baseRouter.Group("/api/v1")
	api.Use(middleware.OptionalAuth(forumService))
	{
		api.POST("/auth/email-verification-code", forumHandler.SendEmailVerificationCode)
		api.POST("/auth/register", forumHandler.Register)
		api.POST("/auth/login", forumHandler.Login)
		api.GET("/me", middleware.RequireAuth(forumService), forumHandler.Me)
		api.GET("/taxonomy", forumHandler.Taxonomy)
		api.GET("/content", adminHandler.ListPublishedContent)
		api.GET("/insights", forumHandler.ListInsights)
		api.GET("/insights/:id", forumHandler.GetInsight)
		api.GET("/topics", forumHandler.ListTopics)
		api.GET("/topics/:slug", forumHandler.GetTopic)
		api.GET("/posts", forumHandler.ListPosts)
		api.POST("/posts", middleware.RequireAuth(forumService), forumHandler.CreatePost)
		api.GET("/posts/:id", forumHandler.GetPost)
		api.POST("/posts/:id/comments", middleware.RequireAuth(forumService), forumHandler.CreateComment)
		api.POST("/posts/:id/like", middleware.RequireAuth(forumService), forumHandler.TogglePostLike)
		api.POST("/posts/:id/favorite", middleware.RequireAuth(forumService), forumHandler.TogglePostFavorite)
		api.POST("/authors/:name/follow", middleware.RequireAuth(forumService), forumHandler.ToggleFollowAuthor)
		api.POST("/ai/choice-advice", middleware.RequireAuth(forumService), forumHandler.ChoiceAdvice)

		api.POST("/admin/login", adminHandler.Login)

		admin := api.Group("/admin")
		admin.Use(requireAdmin(cfg))
		{
			admin.GET("/email-config", adminHandler.EmailConfig)
			admin.POST("/email-test", adminHandler.SendTestEmail)
			admin.GET("/content", adminHandler.ListContent)
			admin.POST("/content", adminHandler.CreateContent)
			admin.PUT("/content/:id", adminHandler.UpdateContent)
			admin.POST("/content/:id/workflow", adminHandler.WorkflowContent)
			admin.DELETE("/content/:id", adminHandler.DeleteContent)
			admin.POST("/uploads/images", adminHandler.UploadImage)
			admin.GET("/content-summary", adminHandler.ContentSummary)
			admin.GET("/audit-logs", adminHandler.AuditLogs)
		}
	}

	registerSPA(router, logger, cfg.FrontendDistDir, cfg.AppBasePath)

	return &http.Server{
		Addr:              cfg.HTTPAddress(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func requireAdmin(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.AdminToken == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"code":    "admin_disabled",
					"message": "ADMIN_TOKEN is not configured",
				},
			})
			return
		}
		token := c.GetHeader("X-Admin-Token")
		if token == "" {
			token = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(cfg.AdminToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "unauthorized",
					"message": "invalid admin token",
				},
			})
			return
		}
		c.Next()
	}
}
