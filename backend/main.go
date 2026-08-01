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
	"subject-choice-forum/backend/internal/observability"
	"subject-choice-forum/backend/internal/repository"
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
	logger := logx.NewJSON(os.Stdout, logx.LevelInfo)
	if cfg.AppEnv == "local" || cfg.AppEnv == "development" {
		logger = logx.New(os.Stdout, logx.LevelDebug)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	tracing, err := observability.InitTracing(ctx, cfg)
	if err != nil {
		logger.Error("可观测性", "OTLP tracing 初始化失败", logx.F("错误", err))
		os.Exit(1)
	}
	if tracing.Enabled() {
		logger.Info("可观测性", "OTLP tracing 已启用", logx.F("服务名", cfg.OTLPServiceName))
	}

	database, err := storage.NewDatabase(ctx, cfg)
	if err != nil {
		logger.Error("系统", "打开数据库失败", logx.F("错误", err))
		os.Exit(1)
	}
	db := database.DB
	defer db.Close()
	logger.Info("系统", "数据库已连接", logx.F("驱动", database.Driver))
	if cfg.SMTPEnabled() {
		logger.Info("邮件", "SMTP已启用", logx.F("服务器", fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)))
	} else {
		logger.Warn("邮件", "SMTP未完整配置，邮箱验证码不可用")
	}

	forumRepo, err := repository.NewForumRepository(database)
	if err != nil {
		logger.Error("系统", "数据库 repository 初始化失败，拒绝启动且不会回退 SQLite",
			logx.F("驱动", database.Driver), logx.F("错误", err))
		os.Exit(1)
	}
	emailSender := service.NewSMTPEmailSender(cfg, logger)
	if cfg.Production() {
		checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := emailSender.Check(checkCtx)
		cancel()
		if err != nil {
			logger.Error("邮件", "SMTP就绪检查失败，拒绝启动", logx.F("错误", err))
			os.Exit(1)
		}
		logger.Info("邮件", "SMTP传输与凭证检查通过")
	}
	forumService := service.NewForumService(forumRepo, cfg, emailSender)
	server := httpserver.NewServer(cfg, logger, db, forumService, tracing)
	if server == nil {
		logger.Error("系统", "对象存储初始化失败，服务不会启动")
		os.Exit(1)
	}

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
	serverErr := server.Shutdown(shutdownCtx)
	if serverErr != nil {
		logger.Error("系统", "优雅关闭失败", logx.F("错误", serverErr))
	}
	tracingErr := tracing.Shutdown(shutdownCtx)
	if tracingErr != nil {
		logger.Error("可观测性", "OTLP tracing 关闭失败", logx.F("错误", tracingErr))
	}
	if serverErr != nil || tracingErr != nil {
		return
	}
	logger.Info("系统", "后端服务已退出")
}
