package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"subject-choice-forum/backend/internal/config"
	httpserver "subject-choice-forum/backend/internal/http"
	"subject-choice-forum/backend/internal/logx"
	"subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"
)

func main() {
	loadEnvCandidates()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[时间]%s [级别]错误 [模块]系统 [操作]加载配置失败 [错误]%v\n", time.Now().Format("2006-01-02 15:04:05"), err)
		os.Exit(1)
	}
	logger := logx.New(os.Stdout, logx.LevelInfo)
	if cfg.AppEnv == "local" || cfg.AppEnv == "development" {
		logger = logx.New(os.Stdout, logx.LevelDebug)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := storage.NewSQLiteDB(cfg)
	if err != nil {
		logger.Error("系统", "打开数据库失败", logx.F("错误", err))
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("系统", "数据库已连接", logx.F("路径", cfg.SQLitePath))

	forumRepo := sqlite.NewForumRepository(db)
	emailSender := service.NewSMTPEmailSender(cfg)
	forumService := service.NewForumService(forumRepo, cfg, emailSender)
	server := httpserver.NewServer(cfg, logger, db, forumService)

	go func() {
		logger.Info("系统", "后端服务启动", logx.F("地址", server.Addr), logx.F("环境", cfg.AppEnv))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("系统", "后端服务启动失败", logx.F("错误", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Warn("系统", "收到退出信号，开始优雅关闭")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("系统", "优雅关闭失败", logx.F("错误", err))
		return
	}
	logger.Info("系统", "后端服务已退出")
}
