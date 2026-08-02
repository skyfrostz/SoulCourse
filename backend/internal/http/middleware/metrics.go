package middleware

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var defaultLatencyBuckets = []float64{0.05, 0.1, 0.3, 0.5, 1, 2.5, 5}

type MetricsRecorder struct {
	mu        sync.Mutex
	routes    map[metricKey]*routeMetrics
	webVitals map[webVitalKey]*webVitalMetrics
	mobile    map[mobileTelemetryKey]uint64
	buckets   []float64
	started   time.Time
}

type webVitalKey struct{ Name, Rating string }
type mobileTelemetryKey struct{ Event, AppVersion string }
type webVitalMetrics struct {
	Count   uint64
	Sum     float64
	Buckets []uint64
}

var webVitalBuckets = []float64{0.05, 0.1, 0.2, 0.5, 1, 2.5, 5, 10, 50, 100, 200, 500, 1000, 2500, 5000, 10000}

type metricKey struct {
	Method string
	Route  string
	Status string
}

type routeMetrics struct {
	Count   uint64
	Sum     float64
	Buckets []uint64
}

func NewMetricsRecorder() *MetricsRecorder {
	buckets := append([]float64(nil), defaultLatencyBuckets...)
	return &MetricsRecorder{
		routes:    make(map[metricKey]*routeMetrics),
		webVitals: make(map[webVitalKey]*webVitalMetrics),
		mobile:    make(map[mobileTelemetryKey]uint64),
		buckets:   buckets,
		started:   time.Now().UTC(),
	}
}

func (m *MetricsRecorder) MobileTelemetryHandler(c *gin.Context) {
	var input struct {
		Event      string `json:"event" binding:"required"`
		AppVersion string `json:"appVersion" binding:"required"`
		AndroidAPI int    `json:"androidApi" binding:"required"`
		WebView    string `json:"webView" binding:"required"`
		Route      string `json:"route" binding:"required"`
		DurationMS int    `json:"durationMs"`
	}
	allowedEvents := map[string]bool{"boot": true, "network_error": true, "js_error": true, "native_error": true, "upload_error": true}
	if err := c.ShouldBindJSON(&input); err != nil || !allowedEvents[input.Event] || len(input.AppVersion) > 40 || len(input.WebView) > 40 || len(input.Route) > 120 || input.AndroidAPI < 26 || input.AndroidAPI > 100 || input.DurationMS < 0 || input.DurationMS > 60000 {
		AbortWithError(c, http.StatusBadRequest, "invalid_mobile_telemetry", "mobile telemetry payload is invalid")
		return
	}
	m.mu.Lock()
	m.mobile[mobileTelemetryKey{Event: input.Event, AppVersion: input.AppVersion}]++
	m.mu.Unlock()
	c.Status(http.StatusNoContent)
}

func (m *MetricsRecorder) WebVitalsHandler(c *gin.Context) {
	var input struct {
		Name   string  `json:"name" binding:"required"`
		Value  float64 `json:"value" binding:"required"`
		Rating string  `json:"rating" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || (input.Name != "LCP" && input.Name != "INP" && input.Name != "CLS") || input.Value < 0 || input.Value > 60000 || (input.Rating != "good" && input.Rating != "needs-improvement" && input.Rating != "poor") {
		AbortWithError(c, http.StatusBadRequest, "invalid_web_vital", "web vital payload is invalid")
		return
	}
	key := webVitalKey{Name: input.Name, Rating: input.Rating}
	m.mu.Lock()
	metric := m.webVitals[key]
	if metric == nil {
		metric = &webVitalMetrics{Buckets: make([]uint64, len(webVitalBuckets)+1)}
		m.webVitals[key] = metric
	}
	metric.Count++
	metric.Sum += input.Value
	for index, bucket := range webVitalBuckets {
		if input.Value <= bucket {
			metric.Buckets[index]++
		}
	}
	metric.Buckets[len(metric.Buckets)-1]++
	m.mu.Unlock()
	c.Status(http.StatusNoContent)
}

func RequireSameOriginTelemetry() gin.HandlerFunc {
	return func(c *gin.Context) {
		if site := strings.TrimSpace(c.GetHeader("Sec-Fetch-Site")); site != "" && site != "same-origin" {
			AbortWithError(c, http.StatusForbidden, "telemetry_origin_denied", "telemetry must be same-origin")
			return
		}
		if origin := strings.TrimSpace(c.GetHeader("Origin")); origin != "" {
			parsed, err := url.Parse(origin)
			if err != nil || !strings.EqualFold(parsed.Host, c.Request.Host) {
				AbortWithError(c, http.StatusForbidden, "telemetry_origin_denied", "telemetry must be same-origin")
				return
			}
		}
		c.Next()
	}
}

func (m *MetricsRecorder) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		m.Observe(c.Request.Method, metricRoute(c), c.Writer.Status(), time.Since(start))
	}
}

func (m *MetricsRecorder) Observe(method string, route string, status int, duration time.Duration) {
	if m == nil {
		return
	}
	key := metricKey{
		Method: strings.ToUpper(strings.TrimSpace(method)),
		Route:  route,
		Status: strconv.Itoa(status),
	}
	if key.Method == "" {
		key.Method = "UNKNOWN"
	}
	if key.Route == "" {
		key.Route = "unmatched"
	}
	seconds := duration.Seconds()

	m.mu.Lock()
	defer m.mu.Unlock()
	metrics := m.routes[key]
	if metrics == nil {
		metrics = &routeMetrics{Buckets: make([]uint64, len(m.buckets)+1)}
		m.routes[key] = metrics
	}
	metrics.Count++
	metrics.Sum += seconds
	for index, bucket := range m.buckets {
		if seconds <= bucket {
			metrics.Buckets[index]++
		}
	}
	metrics.Buckets[len(metrics.Buckets)-1]++
}

func (m *MetricsRecorder) Handler(c *gin.Context) {
	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.String(http.StatusOK, m.Render())
}

func RequireMetricsToken(token string) gin.HandlerFunc {
	expected := strings.TrimSpace(token)
	return func(c *gin.Context) {
		if expected == "" {
			c.Next()
			return
		}
		actual := strings.TrimSpace(c.GetHeader("X-Metrics-Token"))
		if actual == "" {
			actual = bearerToken(c.GetHeader("Authorization"))
		}
		if subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
			AbortWithError(c, http.StatusUnauthorized, "unauthorized", "invalid metrics token")
			return
		}
		c.Next()
	}
}

func (m *MetricsRecorder) Render() string {
	if m == nil {
		return ""
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	keys := make([]metricKey, 0, len(m.routes))
	for key := range m.routes {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i int, j int) bool {
		if keys[i].Route != keys[j].Route {
			return keys[i].Route < keys[j].Route
		}
		if keys[i].Method != keys[j].Method {
			return keys[i].Method < keys[j].Method
		}
		return keys[i].Status < keys[j].Status
	})

	var builder strings.Builder
	builder.WriteString("# HELP soulcourse_http_requests_total Total HTTP requests by method, route, and status.\n")
	builder.WriteString("# TYPE soulcourse_http_requests_total counter\n")
	for _, key := range keys {
		metrics := m.routes[key]
		builder.WriteString(fmt.Sprintf("soulcourse_http_requests_total{method=%q,route=%q,status=%q} %d\n", key.Method, key.Route, key.Status, metrics.Count))
	}
	builder.WriteString("# HELP soulcourse_http_request_duration_seconds HTTP request duration histogram by method, route, and status.\n")
	builder.WriteString("# TYPE soulcourse_http_request_duration_seconds histogram\n")
	for _, key := range keys {
		metrics := m.routes[key]
		for index, bucket := range m.buckets {
			builder.WriteString(fmt.Sprintf("soulcourse_http_request_duration_seconds_bucket{method=%q,route=%q,status=%q,le=%q} %d\n", key.Method, key.Route, key.Status, formatBucket(bucket), metrics.Buckets[index]))
		}
		builder.WriteString(fmt.Sprintf("soulcourse_http_request_duration_seconds_bucket{method=%q,route=%q,status=%q,le=\"+Inf\"} %d\n", key.Method, key.Route, key.Status, metrics.Buckets[len(metrics.Buckets)-1]))
		builder.WriteString(fmt.Sprintf("soulcourse_http_request_duration_seconds_sum{method=%q,route=%q,status=%q} %.6f\n", key.Method, key.Route, key.Status, metrics.Sum))
		builder.WriteString(fmt.Sprintf("soulcourse_http_request_duration_seconds_count{method=%q,route=%q,status=%q} %d\n", key.Method, key.Route, key.Status, metrics.Count))
	}
	builder.WriteString("# HELP soulcourse_process_start_time_seconds Unix timestamp for backend process start.\n")
	builder.WriteString("# TYPE soulcourse_process_start_time_seconds gauge\n")
	builder.WriteString(fmt.Sprintf("soulcourse_process_start_time_seconds %d\n", m.started.Unix()))
	builder.WriteString("# HELP soulcourse_web_vital_value Browser Web Vital value histogram (milliseconds for LCP/INP, unitless for CLS).\n")
	builder.WriteString("# TYPE soulcourse_web_vital_value histogram\n")
	webKeys := make([]webVitalKey, 0, len(m.webVitals))
	for key := range m.webVitals {
		webKeys = append(webKeys, key)
	}
	sort.Slice(webKeys, func(i, j int) bool {
		if webKeys[i].Name != webKeys[j].Name {
			return webKeys[i].Name < webKeys[j].Name
		}
		return webKeys[i].Rating < webKeys[j].Rating
	})
	for _, key := range webKeys {
		metric := m.webVitals[key]
		for index, bucket := range webVitalBuckets {
			builder.WriteString(fmt.Sprintf("soulcourse_web_vital_value_bucket{name=%q,rating=%q,le=%q} %d\n", key.Name, key.Rating, formatBucket(bucket), metric.Buckets[index]))
		}
		builder.WriteString(fmt.Sprintf("soulcourse_web_vital_value_bucket{name=%q,rating=%q,le=\"+Inf\"} %d\n", key.Name, key.Rating, metric.Buckets[len(metric.Buckets)-1]))
		builder.WriteString(fmt.Sprintf("soulcourse_web_vital_value_sum{name=%q,rating=%q} %.6f\n", key.Name, key.Rating, metric.Sum))
		builder.WriteString(fmt.Sprintf("soulcourse_web_vital_value_count{name=%q,rating=%q} %d\n", key.Name, key.Rating, metric.Count))
	}
	return builder.String()
}

func metricRoute(c *gin.Context) string {
	if route := c.FullPath(); route != "" {
		return route
	}
	return c.Request.URL.Path
}

func formatBucket(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func bearerToken(value string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(value, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(value, prefix))
}
