package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAdminSessionIssueResolveAndRevoke(t *testing.T) {
	store := NewAdminSessionStore(time.Hour)
	principal, err := NewAdminPrincipal(" Editor@Example.com ", AdminRoleContentEditor)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := store.Issue(principal)
	if err != nil {
		t.Fatal(err)
	}

	resolved, ok := store.Resolve(token)
	if !ok || resolved.Email != "editor@example.com" || resolved.Role != AdminRoleContentEditor {
		t.Fatalf("resolved principal = %#v, ok=%t", resolved, ok)
	}
	resolved.Permissions[0] = "forged.permission"
	again, ok := store.Resolve(token)
	if !ok || containsPermission(again.Permissions, "forged.permission") {
		t.Fatal("resolved permissions mutated the stored principal")
	}

	store.Revoke(token)
	if _, ok := store.Resolve(token); ok || store.Valid(token) {
		t.Fatal("revoked session remained valid")
	}
}

func TestAdminCookiesTokensAndContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	SetAdminSessionCookie(c, "session", 60, true)
	SetAdminCSRFCookie(c, "csrf", 60, true)
	cookies := rec.Result().Cookies()
	if len(cookies) != 2 || !cookies[0].HttpOnly || cookies[1].HttpOnly {
		t.Fatalf("cookies=%+v", cookies)
	}
	for _, cookie := range cookies {
		if !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
			t.Fatalf("cookie=%+v", cookie)
		}
	}
	token, err := GenerateCSRFToken()
	if err != nil || len(token) < 40 {
		t.Fatalf("token=%q err=%v", token, err)
	}
	p := AdminPrincipal{Email: "admin@example.test", Role: AdminRoleModerator, Permissions: []string{"reports:read"}}
	ctx := ContextWithAdminPrincipal(context.Background(), p)
	got, ok := AdminPrincipalFromContext(ctx)
	if !ok || got.Email != p.Email {
		t.Fatalf("principal=%+v ok=%v", got, ok)
	}
}

func TestAdminSessionRejectsUnknownRole(t *testing.T) {
	store := NewAdminSessionStore(time.Hour)
	if _, _, err := store.Issue(AdminPrincipal{Email: "admin@example.com", Role: "unknown"}); err == nil {
		t.Fatal("unknown role was accepted")
	}
}

func TestRequireAdminPermissionRoleMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		role       string
		permission string
		wantStatus int
	}{
		{"super admin content delete", AdminRoleSuperAdmin, AdminPermissionContentDelete, http.StatusNoContent},
		{"super admin password reset", AdminRoleSuperAdmin, AdminPermissionUsersPasswordReset, http.StatusNoContent},
		{"content editor publish", AdminRoleContentEditor, AdminPermissionContentPublish, http.StatusNoContent},
		{"content editor cannot moderate", AdminRoleContentEditor, AdminPermissionModerationAct, http.StatusForbidden},
		{"moderator can ban", AdminRoleModerator, AdminPermissionUsersBan, http.StatusNoContent},
		{"moderator cannot reset password", AdminRoleModerator, AdminPermissionUsersPasswordReset, http.StatusForbidden},
		{"moderator cannot write content", AdminRoleModerator, AdminPermissionContentWrite, http.StatusForbidden},
		{"empty permission denies", AdminRoleSuperAdmin, "", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			principal, err := NewAdminPrincipal("admin@example.com", tt.role)
			if err != nil {
				t.Fatal(err)
			}
			router := gin.New()
			router.Use(func(c *gin.Context) {
				if !SetAdminPrincipal(c, principal) {
					t.Fatal("could not set principal")
				}
				c.Next()
			})
			router.POST("/admin/action", RequireAdminPermission(tt.permission), func(c *gin.Context) { c.Status(http.StatusNoContent) })

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/admin/action", nil))
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
		})
	}
}

func TestRequireAdminPermissionDefaultsToDeny(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/missing", RequireAdminPermission(AdminPermissionAuditRead), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.POST("/forged", func(c *gin.Context) {
		c.Set(AdminPrincipalContextKey, AdminPrincipal{
			Email:       "attacker@example.com",
			Role:        "unknown",
			Permissions: []string{AdminPermissionAuditRead},
		})
		c.Next()
	}, RequireAdminPermission(AdminPermissionAuditRead), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for path, want := range map[string]int{"/missing": http.StatusUnauthorized, "/forged": http.StatusForbidden} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, nil))
		if recorder.Code != want {
			t.Fatalf("%s status = %d, want %d", path, recorder.Code, want)
		}
	}
}
