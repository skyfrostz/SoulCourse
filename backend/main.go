package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"subject-choice-forum/backend/internal/config"
	httpserver "subject-choice-forum/backend/internal/http"
	"subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
	if cfg.AppEnv == "local" || cfg.AppEnv == "development" {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := storage.NewSQLiteDB(cfg)
	if err != nil {
		logger.Error("open sqlite", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	forumRepo := sqlite.NewForumRepository(db)
	emailSender := service.NewSMTPEmailSender(cfg)
	forumService := service.NewForumService(forumRepo, cfg, emailSender)
	server := httpserver.NewServer(cfg, logger, db, forumService)

	go func() {
		logger.Info("api listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
