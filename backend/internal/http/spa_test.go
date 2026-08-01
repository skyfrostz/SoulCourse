package httpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"subject-choice-forum/backend/internal/logx"
)

func TestSPAFallsBackToIndexAndRejectsMissingAssets(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<main>spa</main>"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0600); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	registerSPA(router, logx.New(io.Discard, logx.LevelError), dir, "/app")
	cases := []struct {
		name, method, path string
		status             int
		body               string
		cache              string
	}{
		{"root", http.MethodGet, "/app/", 200, "spa", "no-cache"},
		{"deep link", http.MethodGet, "/app/requirements", 200, "spa", "no-cache"},
		{"asset", http.MethodGet, "/app/assets/app.js", 200, "console.log", "public, max-age=31536000, immutable"},
		{"missing asset", http.MethodGet, "/app/assets/missing.js", 404, "", ""},
		{"api", http.MethodGet, "/app/api/v1/unknown", 404, "", ""},
		{"wrong base", http.MethodGet, "/other/requirements", 404, "", ""},
		{"post fallback", http.MethodPost, "/app/requirements", 404, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
			if rec.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%q", rec.Code, tc.status, rec.Body.String())
			}
			if tc.body != "" && !strings.Contains(rec.Body.String(), tc.body) {
				t.Fatalf("body=%q missing %q", rec.Body.String(), tc.body)
			}
			if tc.cache != "" && rec.Header().Get("Cache-Control") != tc.cache {
				t.Fatalf("cache=%q want=%q", rec.Header().Get("Cache-Control"), tc.cache)
			}
		})
	}
}

func TestSPACleansAndStripsBasePath(t *testing.T) {
	for _, tc := range []struct {
		in, base, want string
		ok             bool
	}{
		{"", "", "/", true}, {"/a/../", "", "/", true}, {"/app", "/app", "/", true},
		{"/app/x", "/app", "/x", true}, {"/application/x", "/app", "", false},
	} {
		clean := cleanRequestPath(tc.in)
		got, ok := stripBasePath(clean, tc.base)
		if got != tc.want || ok != tc.ok {
			t.Errorf("in=%q base=%q got=%q,%v want=%q,%v", tc.in, tc.base, got, ok, tc.want, tc.ok)
		}
	}
	for _, p := range []string{"/healthz", "/readyz", "/api/v1/posts", "/uploads/x"} {
		if !skipSPAPath(p) {
			t.Errorf("%s should be skipped", p)
		}
	}
	if skipSPAPath("/requirements") {
		t.Error("ordinary app path should not be skipped")
	}
}
