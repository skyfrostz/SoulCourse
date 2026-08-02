package httpserver

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/http/handler"
	"subject-choice-forum/backend/internal/http/middleware"
	"subject-choice-forum/backend/internal/logx"
	"subject-choice-forum/backend/internal/observability"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewServer(
	cfg config.Config,
	logger *logx.Logger,
	db *sql.DB,
	forumService *service.ForumService,
	tracing ...*observability.Tracing,
) *http.Server {
	ctx := context.Background()
	cancel := func() {}
	if cfg.Production() {
		timeout := cfg.DatabaseConnectTimeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	objectStore, err := storage.NewObjectStore(ctx, cfg)
	if err != nil {
		logger.Error("存储", "对象存储初始化失败", logx.F("错误", err))
		return nil
	}
	if cfg.Production() {
		if err := objectStore.Check(ctx); err != nil {
			logger.Error("存储", "对象存储就绪检查失败", logx.F("错误", err))
			return nil
		}
	}
	return NewServerWithObjectStore(cfg, logger, db, forumService, objectStore, tracing...)
}

func NewServerWithObjectStore(
	cfg config.Config,
	logger *logx.Logger,
	db *sql.DB,
	forumService *service.ForumService,
	objectStore storage.ObjectStore,
	tracing ...*observability.Tracing,
) *http.Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		logger.Error("HTTP", "可信代理配置无效", logx.F("错误", err))
	}
	router.Use(middleware.RequestID())
	if len(tracing) > 0 && tracing[0] != nil && tracing[0].Enabled() {
		router.Use(middleware.Tracing(tracing[0].Tracer()))
	}
	router.Use(middleware.RecoveryLogger(logger))
	router.Use(middleware.RequestLogger(logger))
	metricsRecorder := middleware.NewMetricsRecorder()
	router.Use(metricsRecorder.Middleware())
	router.Use(middleware.SecurityHeaders(cfg.Production()))
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	healthHandler := handler.NewHealthHandlerWithDatabase(db, cfg.DatabaseDriver, cfg.DatabaseHealthTimeout)
	aiService := service.NewAIService(cfg)
	forumService.ConfigurePostTagger(aiService, func(err error) {
		logger.Warn("AI", "帖子自动打标失败，保留手动标签", logx.F("错误", err))
	})
	forumHandler := handler.NewForumHandlerWithObjectStore(forumService, aiService, cfg.Production(), cfg.MediaUploadDir, cfg.AppBasePath, objectStore)
	adminSessionStore := middleware.NewAdminSessionStore(8 * time.Hour)
	adminHandler := handler.NewAdminHandler(cfg, forumService, db, adminSessionStore)
	rateLimiter := middleware.NewRateLimiter()
	authRateLimit := rateLimiter.Limit(middleware.RateLimitRule{Name: "auth", Limit: 20, Window: time.Minute, KeyFunc: middleware.ClientIPKey})
	writeRateLimit := rateLimiter.Limit(middleware.RateLimitRule{Name: "write", Limit: 30, Window: time.Minute, KeyFunc: middleware.ClientIPAndUserKey})
	aiRateLimit := rateLimiter.Limit(middleware.RateLimitRule{Name: "ai", Limit: 10, Window: time.Minute, KeyFunc: middleware.ClientIPAndUserKey})
	telemetryRateLimit := rateLimiter.Limit(middleware.RateLimitRule{Name: "telemetry", Limit: 120, Window: time.Minute, KeyFunc: middleware.ClientIPKey})

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
	} else {
		router.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/welcome")
		})
		router.HEAD("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/welcome")
		})
	}

	baseRouter := router.Group(cfg.AppBasePath)

	baseRouter.GET("/healthz", healthHandler.Live)
	baseRouter.HEAD("/healthz", healthHandler.Live)
	baseRouter.GET("/readyz", healthHandler.Ready)
	baseRouter.HEAD("/readyz", healthHandler.Ready)
	baseRouter.GET("/metrics", middleware.RequireMetricsToken(cfg.MetricsToken), metricsRecorder.Handler)
	baseRouter.Static("/uploads", cfg.MediaUploadDir)

	api := baseRouter.Group("/api/v1")
	api.Use(middleware.OptionalAuth(forumService))
	api.Use(middleware.BodyLimit(cfg.HTTPMaxBodyBytes))
	api.Use(middleware.CSRFProtection())
	{
		api.POST("/telemetry/web-vitals", middleware.RequireSameOriginTelemetry(), telemetryRateLimit, metricsRecorder.WebVitalsHandler)
		api.POST("/auth/email-verification-code", authRateLimit, forumHandler.SendEmailVerificationCode)
		api.POST("/auth/forgot-password", authRateLimit, forumHandler.ForgotPassword)
		api.POST("/auth/register", authRateLimit, forumHandler.Register)
		api.POST("/auth/login", authRateLimit, forumHandler.Login)
		api.POST("/auth/reset-password", authRateLimit, forumHandler.ResetPassword)
		api.POST("/auth/logout", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.Logout)
		api.GET("/me", middleware.RequireAuth(forumService), forumHandler.Me)
		api.DELETE("/me", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.DeleteMe)
		api.GET("/me/profile", middleware.RequireAuth(forumService), forumHandler.GetMyProfile)
		api.PUT("/me/profile", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.UpdateMyProfile)
		api.GET("/me/sessions", middleware.RequireAuth(forumService), forumHandler.ListMySessions)
		api.DELETE("/me/sessions/:id", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.RevokeMySession)
		api.POST("/uploads/images/presign", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.PresignImageUpload)
		api.PUT("/uploads/images/:id/object", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.PutImageUploadObject)
		api.POST("/uploads/images/:id/complete", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.CompleteImageUpload)
		api.GET("/profiles/:name", forumHandler.GetProfile)
		api.GET("/notifications", middleware.RequireAuth(forumService), forumHandler.ListNotifications)
		api.POST("/notifications/read-all", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.MarkAllNotificationsRead)
		api.POST("/notifications/:id/read", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.MarkNotificationRead)
		api.GET("/messages", middleware.RequireAuth(forumService), forumHandler.ListConversations)
		api.GET("/messages/:name", middleware.RequireAuth(forumService), forumHandler.ListDirectMessages)
		api.POST("/messages", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.SendDirectMessage)
		api.GET("/taxonomy", forumHandler.Taxonomy)
		api.GET("/content", adminHandler.ListPublishedContent)
		api.GET("/provinces", adminHandler.ListProvinces)
		api.GET("/policies", adminHandler.ListPolicies)
		api.GET("/requirements", adminHandler.ListRequirements)
		api.GET("/sources/:id", adminHandler.GetSource)
		api.GET("/insights", forumHandler.ListInsights)
		api.GET("/insights/:id", forumHandler.GetInsight)
		api.GET("/topics", forumHandler.ListTopics)
		api.GET("/topics/:slug", forumHandler.GetTopic)
		api.GET("/posts", forumHandler.ListPosts)
		api.POST("/posts", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.CreatePost)
		api.GET("/posts/:id", forumHandler.GetPost)
		api.PUT("/posts/:id", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.UpdatePost)
		api.DELETE("/posts/:id", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.DeletePost)
		api.POST("/posts/:id/comments", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.CreateComment)
		api.POST("/posts/:id/report", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.ReportPost)
		api.POST("/posts/:id/like", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.TogglePostLike)
		api.POST("/posts/:id/favorite", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.TogglePostFavorite)
		api.POST("/authors/:name/follow", middleware.RequireAuth(forumService), writeRateLimit, forumHandler.ToggleFollowAuthor)
		api.POST("/ai/choice-advice", middleware.RequireAuth(forumService), aiRateLimit, forumHandler.ChoiceAdvice)

		api.POST("/admin/login", authRateLimit, adminHandler.Login)

		admin := api.Group("/admin")
		admin.Use(requireAdmin(cfg, adminSessionStore))
		{
			admin.POST("/logout", writeRateLimit, adminHandler.Logout)
			admin.GET("/email-config", middleware.RequireAdminPermission(middleware.AdminPermissionSystemEmailRead), adminHandler.EmailConfig)
			admin.POST("/email-test", middleware.RequireAdminPermission(middleware.AdminPermissionSystemEmailTest), writeRateLimit, adminHandler.SendTestEmail)
			admin.GET("/content", middleware.RequireAdminPermission(middleware.AdminPermissionContentRead), adminHandler.ListContent)
			admin.POST("/content", middleware.RequireAdminPermission(middleware.AdminPermissionContentWrite), writeRateLimit, adminHandler.CreateContent)
			admin.PUT("/content/:id", middleware.RequireAdminPermission(middleware.AdminPermissionContentWrite), writeRateLimit, adminHandler.UpdateContent)
			admin.POST("/content/:id/workflow", middleware.RequireAdminPermission(middleware.AdminPermissionContentPublish), writeRateLimit, adminHandler.WorkflowContent)
			admin.DELETE("/content/:id", middleware.RequireAdminPermission(middleware.AdminPermissionContentDelete), writeRateLimit, adminHandler.DeleteContent)
			admin.POST("/uploads/images", middleware.RequireAdminPermission(middleware.AdminPermissionMediaUpload), writeRateLimit, adminHandler.UploadImage)
			admin.GET("/content-summary", middleware.RequireAdminPermission(middleware.AdminPermissionDashboardRead), adminHandler.ContentSummary)
			admin.GET("/audit-logs", middleware.RequireAdminPermission(middleware.AdminPermissionAuditRead), adminHandler.AuditLogs)
			admin.GET("/reports", middleware.RequireAdminPermission(middleware.AdminPermissionModerationRead), adminHandler.ListReports)
			admin.POST("/reports/:id/moderate", middleware.RequireAdminPermission(middleware.AdminPermissionModerationAct), writeRateLimit, adminHandler.ModerateReport)
			admin.POST("/users/:id/ban", middleware.RequireAdminPermission(middleware.AdminPermissionUsersBan), writeRateLimit, adminHandler.BanUser)
			admin.POST("/users/:id/restore", middleware.RequireAdminPermission(middleware.AdminPermissionUsersBan), writeRateLimit, adminHandler.RestoreUser)
			admin.PUT("/users/:id/password", middleware.RequireAdminPermission(middleware.AdminPermissionUsersPasswordReset), writeRateLimit, adminHandler.ResetUserPassword)
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

func requireAdmin(cfg config.Config, sessionStore *middleware.AdminSessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(middleware.AdminSessionCookieName)
		principal, valid := sessionStore.Resolve(token)
		if err != nil || !valid || !middleware.SetAdminPrincipal(c, principal) {
			middleware.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "invalid admin session")
			return
		}
		c.Request = c.Request.WithContext(middleware.ContextWithAdminPrincipal(c.Request.Context(), principal))
		c.Next()
	}
}
