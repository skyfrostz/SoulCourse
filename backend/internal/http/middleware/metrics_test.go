package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestWebVitalsHandlerValidatesAndAggregatesAnonymousMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := NewMetricsRecorder()
	router := gin.New()
	router.POST("/api/v1/telemetry/web-vitals", RequireSameOriginTelemetry(), recorder.WebVitalsHandler)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/web-vitals", bytes.NewBufferString(`{"name":"LCP","value":1800,"rating":"good"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("valid metric status=%d body=%s", response.Code, response.Body.String())
	}
	body := recorder.Render()
	if !strings.Contains(body, `soulcourse_web_vital_value_count{name="LCP",rating="good"} 1`) {
		t.Fatalf("web vital metric missing:\n%s", body)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/web-vitals", bytes.NewBufferString(`{"name":"EMAIL","value":1,"rating":"good","user":"private"}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid metric status=%d want=400", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/web-vitals", bytes.NewBufferString(`{"name":"CLS","value":0.05,"rating":"good"}`))
	request.Host = "soulcourse.cn"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://attacker.example")
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-origin metric status=%d want=403", response.Code)
	}
}

func TestRequireSameOriginTelemetryBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID(), RequireSameOriginTelemetry())
	router.POST("/telemetry", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, tc := range []struct {
		name, site, origin, host string
		want                     int
	}{
		{"same origin", "same-origin", "https://example.test", "example.test", 204},
		{"cross site", "cross-site", "", "example.test", 403},
		{"origin mismatch", "", "https://evil.test", "example.test", 403},
		{"malformed origin", "", "://bad", "example.test", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/telemetry", nil)
			req.Host = tc.host
			req.Header.Set("Sec-Fetch-Site", tc.site)
			req.Header.Set("Origin", tc.origin)
			router.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestMetricsRecorderConcurrentObserveAndRender(t *testing.T) {
	m := NewMetricsRecorder()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); m.Observe("get", "/items/:id", 200, time.Millisecond); _ = m.Render() }()
	}
	wg.Wait()
	if !strings.Contains(m.Render(), `soulcourse_http_requests_total{method="GET",route="/items/:id",status="200"} 50`) {
		t.Fatal(m.Render())
	}
}

func TestMetricsRecorderRendersPrometheusText(t *testing.T) {
	recorder := NewMetricsRecorder()

	recorder.Observe(http.MethodGet, "/api/v1/posts", http.StatusOK, 75*time.Millisecond)
	recorder.Observe(http.MethodGet, "/api/v1/posts", http.StatusOK, 350*time.Millisecond)
	recorder.Observe(http.MethodPost, "/api/v1/posts", http.StatusCreated, 520*time.Millisecond)

	body := recorder.Render()
	for _, want := range []string{
		"# TYPE soulcourse_http_requests_total counter",
		`soulcourse_http_requests_total{method="GET",route="/api/v1/posts",status="200"} 2`,
		`soulcourse_http_request_duration_seconds_bucket{method="GET",route="/api/v1/posts",status="200",le="0.1"} 1`,
		`soulcourse_http_request_duration_seconds_bucket{method="GET",route="/api/v1/posts",status="200",le="+Inf"} 2`,
		`soulcourse_http_request_duration_seconds_count{method="POST",route="/api/v1/posts",status="201"} 1`,
		"soulcourse_process_start_time_seconds",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q:\n%s", want, body)
		}
	}
}

func TestMetricsMiddlewareRecordsGinRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := NewMetricsRecorder()
	router := gin.New()
	router.Use(recorder.Middleware())
	router.GET("/api/v1/posts/:id", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})
	router.GET("/metrics", recorder.Handler)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/posts/42", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("route status = %d, want 404", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	body := response.Body.String()
	if !strings.Contains(body, `route="/api/v1/posts/:id"`) || !strings.Contains(body, `status="404"`) {
		t.Fatalf("metrics did not record templated route/status:\n%s", body)
	}
	if strings.Contains(body, `route="/metrics"`) {
		t.Fatalf("metrics endpoint should not record itself:\n%s", body)
	}
}

func TestRequireMetricsToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := NewMetricsRecorder()
	router := gin.New()
	router.Use(RequestID())
	router.GET("/metrics", RequireMetricsToken("metrics-token-0123456789abcdef"), recorder.Handler)

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("metrics without token status = %d, want 401", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	request.Header.Set("Authorization", "Bearer metrics-token-0123456789abcdef")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("metrics with bearer token status = %d, want 200 body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "soulcourse_process_start_time_seconds") {
		t.Fatalf("metrics body missing process metric: %s", response.Body.String())
	}
}
