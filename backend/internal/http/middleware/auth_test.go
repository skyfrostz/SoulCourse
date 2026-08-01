package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func TestPostInteractionsRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	forum := service.NewForumService(nil, config.Config{JWTSecret: "test-secret"}, nil)
	router := gin.New()
	router.Use(RequestID())
	protected := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	router.POST("/posts/:id/like", RequireAuth(forum), protected)
	router.POST("/posts/:id/comments", RequireAuth(forum), protected)

	for _, path := range []string{"/posts/1/like", "/posts/1/comments"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, path, nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s returned %d, want 401", path, recorder.Code)
		}
		if recorder.Header().Get(RequestIDHeader) == "" {
			t.Fatalf("%s did not return request id header", path)
		}
		if recorder.Body.String() != `{"error":{"code":"unauthorized","message":"please login first","requestId":"`+recorder.Header().Get(RequestIDHeader)+`"}}` {
			t.Fatalf("%s returned unexpected envelope: %s", path, recorder.Body.String())
		}
	}
}

func TestCSRFProtectionRequiresTokenForCookieAuthenticatedWrites(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.Use(CSRFProtection())
	router.POST("/public-write", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.POST("/api/v1/admin/login", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.POST("/api/v1/admin/content", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.POST("/api/v1/admin/reports/:id/moderate", func(c *gin.Context) {
		c.Set(CurrentUserKey, domain.User{ID: 1, Nickname: "普通用户"})
		CSRFProtection()(c)
	}, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.POST("/user-write", func(c *gin.Context) {
		c.Set(CurrentUserKey, domain.User{ID: 1, Nickname: "测试用户"})
		CSRFProtection()(c)
	}, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.POST("/write", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	t.Run("allows public writes with stale session cookie", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/public-write", nil)
		request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "expired-or-revoked"})
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", recorder.Code)
		}
	})

	t.Run("allows admin login with stale admin session cookie", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", nil)
		request.AddCookie(&http.Cookie{Name: AdminSessionCookieName, Value: "expired-admin-session"})
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", recorder.Code)
		}
	})

	t.Run("requires admin csrf for protected admin writes", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/content", nil)
		request.AddCookie(&http.Cookie{Name: AdminSessionCookieName, Value: "admin-session"})
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", recorder.Code)
		}
	})

	t.Run("authorization header cannot bypass admin cookie csrf", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/content", nil)
		request.AddCookie(&http.Cookie{Name: AdminSessionCookieName, Value: "admin-session"})
		request.Header.Set("Authorization", "Bearer arbitrary-value")
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", recorder.Code)
		}
	})

	t.Run("prefers admin csrf for admin writes when user session is also present", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/reports/1/moderate", nil)
		request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "user-session"})
		request.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "user-csrf"})
		request.AddCookie(&http.Cookie{Name: AdminSessionCookieName, Value: "admin-session"})
		request.AddCookie(&http.Cookie{Name: AdminCSRFCookieName, Value: "admin-csrf"})
		request.Header.Set(CSRFHeaderName, "admin-csrf")
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", recorder.Code)
		}
	})

	t.Run("rejects missing csrf header for authenticated user writes", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/user-write", nil)
		request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "session"})
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", recorder.Code)
		}
	})

	t.Run("allows matching csrf cookie and header", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/user-write", nil)
		request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "session"})
		request.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "csrf"})
		request.Header.Set(CSRFHeaderName, "csrf")
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", recorder.Code)
		}
	})

	t.Run("allows bearer compatibility during migration", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/write", nil)
		request.Header.Set("Authorization", "Bearer legacy-token")
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", recorder.Code)
		}
	})
}

func TestAuthTokenAndCookieBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"token": func() string { v, _ := SessionToken(c); return v }()})
	})
	for _, tc := range []struct {
		name, authorization, cookie, want string
	}{
		{"missing", "", "", ""},
		{"wrong scheme", "Basic abc", "", ""},
		{"empty bearer", "Bearer   ", "", ""},
		{"bearer trims", "Bearer  abc  ", "", "abc"},
		{"cookie wins", "Bearer bearer", " cookie ", "cookie"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", io.Reader(nil))
			req.Header.Set("Authorization", tc.authorization)
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: tc.cookie})
			}
			router.ServeHTTP(rec, req)
			if !strings.Contains(rec.Body.String(), `"token":"`+tc.want+`"`) {
				t.Fatalf("body=%s", rec.Body.String())
			}
		})
	}
}

func TestSessionAndCSRFCookieAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, secure := range []bool{false, true} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		SetSessionCookie(c, "session", 3600, secure)
		SetCSRFCookie(c, "csrf", 3600, secure)
		cookies := rec.Result().Cookies()
		if len(cookies) != 2 {
			t.Fatalf("secure=%v cookie count=%d", secure, len(cookies))
		}
		if !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].Secure != secure {
			t.Fatalf("session cookie attrs=%+v", cookies[0])
		}
		if cookies[1].HttpOnly || cookies[1].SameSite != http.SameSiteLaxMode || cookies[1].Secure != secure {
			t.Fatalf("csrf cookie attrs=%+v", cookies[1])
		}
	}
}
