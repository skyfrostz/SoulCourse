package middleware

import (
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/logx"

	"github.com/gin-gonic/gin"
)

func RequestLogger(logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.Writer.Status() < http.StatusBadRequest && isStaticAssetRequest(c) {
			return
		}

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

func isStaticAssetRequest(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return false
	}
	return path.Ext(c.Request.URL.Path) != ""
}

func requestPathWithQuery(c *gin.Context) string {
	if c.Request.URL.RawQuery == "" {
		return c.Request.URL.Path
	}
	return c.Request.URL.Path + "?" + redactQuery(c.Request.URL.Query()).Encode()
}

func redactQuery(values url.Values) url.Values {
	redacted := make(url.Values, len(values))
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if isSensitiveQueryKey(key) {
			redacted[key] = []string{"[REDACTED]"}
			continue
		}
		redacted[key] = append([]string(nil), values[key]...)
	}
	return redacted
}

func isSensitiveQueryKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
	return strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "code") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "email") ||
		strings.Contains(normalized, "phone")
}
