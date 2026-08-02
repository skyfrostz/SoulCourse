package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersSetHSTSForProductionAndForwardedHTTPS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		production bool
		proto      string
		wantHSTS   bool
	}{
		{name: "local plain http", wantHSTS: false},
		{name: "production behind proxy", production: true, wantHSTS: true},
		{name: "forwarded https", proto: "https", wantHSTS: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(SecurityHeaders(tt.production))
			router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.proto != "" {
				request.Header.Set("X-Forwarded-Proto", tt.proto)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			got := response.Header().Get("Strict-Transport-Security") != ""
			if got != tt.wantHSTS {
				t.Fatalf("HSTS present = %v, want %v", got, tt.wantHSTS)
			}
		})
	}
}

func TestSecurityHeadersRestrictBrowserCapabilities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders(true))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	csp := response.Header().Get("Content-Security-Policy")
	for _, directive := range []string{
		"connect-src 'self'",
		"font-src 'self' data: https://static.figma.com",
		"media-src 'self' https://d8j0ntlcm91z4.cloudfront.net",
		"form-action 'self'",
		"frame-ancestors 'none'",
		"upgrade-insecure-requests",
	} {
		if !strings.Contains(csp, directive) {
			t.Fatalf("CSP missing %q: %s", directive, csp)
		}
	}
	if strings.Contains(csp, "connect-src 'self' https:") {
		t.Fatalf("CSP permits arbitrary HTTPS connections: %s", csp)
	}
	if got := response.Header().Get("X-Permitted-Cross-Domain-Policies"); got != "none" {
		t.Fatalf("X-Permitted-Cross-Domain-Policies = %q", got)
	}
}
