package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/domain"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterRejectsRequestsOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewRateLimiter()
	router := gin.New()
	router.Use(RequestID())
	router.POST("/login", limiter.Limit(RateLimitRule{
		Name:   "login",
		Limit:  2,
		Window: time.Minute,
	}), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("{}"))
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("request %d status = %d, want 204", i+1, recorder.Code)
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("{}"))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", recorder.Code)
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
	if !strings.Contains(recorder.Body.String(), `"code":"rate_limited"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"requestId":"`+recorder.Header().Get(RequestIDHeader)+`"`) {
		t.Fatalf("response did not include request id: %s", recorder.Body.String())
	}
}

func TestRateLimiterConcurrentBoundary(t *testing.T) {
	limiter := NewRateLimiter()
	rule := RateLimitRule{Name: "comment", Limit: 25, Window: time.Minute}
	var allowed atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ok, _ := limiter.allow(rule, "same-client"); ok {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := allowed.Load(); got != 25 {
		t.Fatalf("allowed=%d, want 25", got)
	}
}

func TestRateLimitKeysSeparateUsersAndAnonymousClients(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "203.0.113.10:1234"
	if got := ClientIPAndUserKey(c); got != "203.0.113.10" {
		t.Fatalf("anonymous key=%q", got)
	}
	c.Set(CurrentUserKey, domain.User{ID: 42})
	if got := ClientIPAndUserKey(c); got != "203.0.113.10:user:42" {
		t.Fatalf("user key=%q", got)
	}
}

func TestRateLimiterResetsAfterWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	start := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter()
	limiter.now = func() time.Time { return start }

	if allowed, _ := limiter.allow(RateLimitRule{Name: "test", Limit: 1, Window: time.Second}, "client"); !allowed {
		t.Fatal("first request should be allowed")
	}
	if allowed, _ := limiter.allow(RateLimitRule{Name: "test", Limit: 1, Window: time.Second}, "client"); allowed {
		t.Fatal("second request in the same window should be rejected")
	}

	start = start.Add(time.Second)
	if allowed, _ := limiter.allow(RateLimitRule{Name: "test", Limit: 1, Window: time.Second}, "client"); !allowed {
		t.Fatal("request after window should be allowed")
	}
}

func TestRateLimiterInvalidRuleAndUserKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, rule := range []RateLimitRule{{Name: "zero", Limit: 0, Window: time.Minute}, {Name: "window", Limit: 1, Window: 0}} {
		router := gin.New()
		router.Use(NewRateLimiter().Limit(rule))
		router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("invalid rule status=%d", recorder.Code)
		}
	}

	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		c.Set(CurrentUserKey, domain.User{ID: 42})
		c.String(http.StatusOK, ClientIPAndUserKey(c))
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(recorder.Body.String(), ":user:42") {
		t.Fatalf("user key=%q", recorder.Body.String())
	}
}
