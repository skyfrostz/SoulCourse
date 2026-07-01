package httpserver

import (
	"net/http"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/http/handler"
	"subject-choice-forum/backend/internal/http/middleware"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func NewServer(
	cfg config.Config,
	logger *zap.Logger,
	db *pgxpool.Pool,
	redisClient *redis.Client,
	forumService *service.ForumService,
) *http.Server {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger(logger))
	router.Use(middleware.SecurityHeaders())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	healthHandler := handler.NewHealthHandler(db, redisClient)
	aiService := service.NewAIService(cfg)
	forumHandler := handler.NewForumHandler(forumService, aiService)

	router.GET("/healthz", healthHandler.Live)
	router.GET("/readyz", healthHandler.Ready)

	api := router.Group("/api/v1")
	api.Use(middleware.OptionalAuth(forumService))
	{
		api.POST("/auth/register", forumHandler.Register)
		api.POST("/auth/login", forumHandler.Login)
		api.GET("/me", middleware.RequireAuth(forumService), forumHandler.Me)
		api.GET("/taxonomy", forumHandler.Taxonomy)
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
	}

	return &http.Server{
		Addr:              cfg.HTTPAddress(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
