package middleware

import (
	"time"

	"subject-choice-forum/backend/internal/logx"

	"github.com/gin-gonic/gin"
)

func RequestLogger(logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		level := logx.LevelInfo
		if status >= 500 {
			level = logx.LevelError
		} else if status >= 400 {
			level = logx.LevelWarn
		}

		fields := []logx.Field{
			logx.F("方法", c.Request.Method),
			logx.F("路径", requestPathWithQuery(c)),
			logx.F("状态", status),
			logx.F("耗时", time.Since(start)),
			logx.F("IP", c.ClientIP()),
		}
		if userID := CurrentUserID(c); userID != nil {
			fields = append(fields, logx.F("用户ID", *userID))
		}
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			fields = append(fields, logx.F("请求ID", requestID))
		}

		logger.Log(level, "HTTP", "请求完成", fields...)
	}
}

func requestPathWithQuery(c *gin.Context) string {
	if c.Request.URL.RawQuery == "" {
		return c.Request.URL.Path
	}
	return c.Request.URL.Path + "?" + c.Request.URL.RawQuery
}
