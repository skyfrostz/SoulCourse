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

func TestRecoveryLoggerConvertsPanicToRequestError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	router := gin.New()
	router.Use(RequestID(), RecoveryLogger(logx.NewJSON(&output, logx.LevelDebug)))
	router.GET("/panic", func(c *gin.Context) { panic("boom") })
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"internal_server_error"`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
	if !strings.Contains(output.String(), "boom") || !strings.Contains(output.String(), "堆栈") {
		t.Fatalf("log=%s", output.String())
	}
}
