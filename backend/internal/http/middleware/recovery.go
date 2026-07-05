package middleware

import (
	"net/http"
	"runtime/debug"

	"subject-choice-forum/backend/internal/logx"

	"github.com/gin-gonic/gin"
)

func RecoveryLogger(logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("HTTP", "请求发生异常",
					logx.F("方法", c.Request.Method),
					logx.F("路径", c.Request.URL.Path),
					logx.F("状态", http.StatusInternalServerError),
					logx.F("IP", c.ClientIP()),
					logx.F("异常", recovered),
					logx.F("堆栈", string(debug.Stack())),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "internal_server_error",
						"message": "server encountered an unexpected error",
					},
				})
			}
		}()

		c.Next()
	}
}
