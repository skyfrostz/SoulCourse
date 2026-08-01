package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"subject-choice-forum/backend/internal/logx"
)

func TestRequestPathWithQueryRedactsSensitiveValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/api/v1/auth/reset-password?email=a@example.com&code=123456&cursor=next&token=secret", nil)

	path := requestPathWithQuery(context)
	if strings.Contains(path, "a@example.com") || strings.Contains(path, "123456") || strings.Contains(path, "secret") {
		t.Fatalf("path leaked sensitive query values: %s", path)
	}
	if !strings.Contains(path, "cursor=next") {
		t.Fatalf("path removed non-sensitive query values: %s", path)
	}
}

func TestRequestLoggerSkipsSuccessfulStaticAssetsAndLogsFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := logx.NewJSON(&output, logx.LevelDebug)
	router := gin.New()
	router.Use(RequestID(), RequestLogger(logger))
	router.GET("/app.js", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/missing.js", func(c *gin.Context) { c.Status(http.StatusNotFound) })
	for _, path := range []string{"/app.js", "/missing.js"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path+"?token=secret", nil))
	}
	if strings.Count(output.String(), "请求完成") != 1 {
		t.Fatalf("log=%s", output.String())
	}
	if strings.Contains(output.String(), "secret") {
		t.Fatalf("sensitive query leaked: %s", output.String())
	}
}
