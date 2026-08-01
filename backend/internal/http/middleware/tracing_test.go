package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTracingUsesRouteTemplateAndSafeAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	router := gin.New()
	router.Use(RequestID())
	router.Use(Tracing(provider.Tracer("test")))
	router.GET("/users/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "/users/secret-user?email=hidden@example.com&token=secret", nil)
	request.Header.Set(RequestIDHeader, "request-12345678")
	request.Header.Set("Authorization", "Bearer secret-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected one span, got %d", len(spans))
	}
	span := spans[0]
	if span.Name() != "GET /users/:id" {
		t.Fatalf("span name = %q", span.Name())
	}
	attributes := map[string]string{}
	for _, item := range span.Attributes() {
		attributes[string(item.Key)] = item.Value.Emit()
	}
	if attributes["http.route"] != "/users/:id" || attributes["http.request.method"] != "GET" || attributes["http.response.status_code"] != "204" || attributes["request.id"] != "request-12345678" {
		t.Fatalf("unexpected span attributes: %#v", attributes)
	}
	for key, value := range attributes {
		if key == "url.full" || key == "http.target" || value == "/users/secret-user" || value == "hidden@example.com" || value == "secret-token" {
			t.Fatalf("sensitive attribute recorded: %s=%q", key, value)
		}
	}
}
