package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBodyLimitRejectsContentLengthOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.Use(BodyLimit(8))
	router.POST("/payload", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/payload", strings.NewReader("0123456789"))
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", recorder.Code)
	}
	if recorder.Header().Get(RequestIDHeader) == "" {
		t.Fatal("expected request id header")
	}
	if !strings.Contains(recorder.Body.String(), `"code":"request_body_too_large"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"requestId":"`+recorder.Header().Get(RequestIDHeader)+`"`) {
		t.Fatalf("response did not include request id: %s", recorder.Body.String())
	}
}

func TestBodyLimitAllowsRequestsWithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(BodyLimit(16))
	router.POST("/payload", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/payload", strings.NewReader("01234567"))
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}
