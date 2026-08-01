package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDUsesTrustedHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"requestId": GetRequestID(c)})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(RequestIDHeader, "req-public-beta-001")
	router.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(RequestIDHeader); got != "req-public-beta-001" {
		t.Fatalf("request id header = %q", got)
	}
}

func TestRequestIDRejectsUnsafeHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(RequestIDHeader, "bad header\nvalue")
	router.ServeHTTP(recorder, request)

	got := recorder.Header().Get(RequestIDHeader)
	if got == "" || got == "bad header\nvalue" {
		t.Fatalf("request id header was not regenerated: %q", got)
	}
}
