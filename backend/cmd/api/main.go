package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"subject-choice-forum/backend/internal/config"
	httpserver "subject-choice-forum/backend/internal/http"
	"subject-choice-forum/backend/internal/repository/postgres"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"

	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("load config", zap.Error(err))
	}

	if cfg.AppEnv == "local" || cfg.AppEnv == "development" {
		if devLogger, err := zap.NewDevelopment(); err == nil {
			logger = devLogger
			defer logger.Sync()
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := storage.NewPostgresPool(ctx, cfg)
	if err != nil {
		logger.Fatal("connect postgres", zap.Error(err))
	}
	defer db.Close()

	redisClient, err := storage.NewRedisClient(ctx, cfg)
	if err != nil {
		logger.Fatal("connect redis", zap.Error(err))
	}
	defer redisClient.Close()

	forumRepo := postgres.NewForumRepository(db)
	forumService := service.NewForumService(forumRepo, cfg)

	server := httpserver.NewServer(cfg, logger, db, redisClient, forumService)

	go func() {
		logger.Info("api listening", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("api server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}
}
