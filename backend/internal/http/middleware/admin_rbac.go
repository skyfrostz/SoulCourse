package middleware

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

const AdminPrincipalContextKey = "adminPrincipal"

type adminPrincipalRequestContextKey struct{}

const (
	AdminRoleSuperAdmin    = "super_admin"
	AdminRoleContentEditor = "content_editor"
	AdminRoleModerator     = "moderator"
)

const (
	AdminPermissionDashboardRead      = "dashboard.read"
	AdminPermissionContentRead        = "content.read"
	AdminPermissionContentWrite       = "content.write"
	AdminPermissionContentPublish     = "content.publish"
	AdminPermissionContentDelete      = "content.delete"
	AdminPermissionMediaUpload        = "media.upload"
	AdminPermissionModerationRead     = "moderation.read"
	AdminPermissionModerationAct      = "moderation.act"
	AdminPermissionUsersRead          = "users.read"
	AdminPermissionUsersBan           = "users.ban"
	AdminPermissionUsersPasswordReset = "users.password_reset"
	AdminPermissionSystemEmailRead    = "system.email.read"
	AdminPermissionSystemEmailTest    = "system.email.test"
	AdminPermissionAuditRead          = "audit.read"
)

var ErrInvalidAdminPrincipal = errors.New("invalid admin principal")

type AdminPrincipal struct {
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

var fixedAdminRolePermissions = map[string][]string{
	AdminRoleSuperAdmin: {
		AdminPermissionDashboardRead,
		AdminPermissionContentRead,
		AdminPermissionContentWrite,
		AdminPermissionContentPublish,
		AdminPermissionContentDelete,
		AdminPermissionMediaUpload,
		AdminPermissionModerationRead,
		AdminPermissionModerationAct,
		AdminPermissionUsersRead,
		AdminPermissionUsersBan,
		AdminPermissionUsersPasswordReset,
		AdminPermissionSystemEmailRead,
		AdminPermissionSystemEmailTest,
		AdminPermissionAuditRead,
	},
	AdminRoleContentEditor: {
		AdminPermissionDashboardRead,
		AdminPermissionContentRead,
		AdminPermissionContentWrite,
		AdminPermissionContentPublish,
		AdminPermissionContentDelete,
		AdminPermissionMediaUpload,
	},
	AdminRoleModerator: {
		AdminPermissionDashboardRead,
		AdminPermissionContentRead,
		AdminPermissionModerationRead,
		AdminPermissionModerationAct,
		AdminPermissionUsersRead,
		AdminPermissionUsersBan,
	},
}

func NewAdminPrincipal(email, role string) (AdminPrincipal, error) {
	return normalizeAdminPrincipal(AdminPrincipal{Email: email, Role: role})
}

func AdminRolePermissions(role string) ([]string, bool) {
	permissions, ok := fixedAdminRolePermissions[strings.TrimSpace(role)]
	if !ok {
		return nil, false
	}
	result := append([]string(nil), permissions...)
	sort.Strings(result)
	return result, true
}

func SetAdminPrincipal(c *gin.Context, principal AdminPrincipal) bool {
	normalized, err := normalizeAdminPrincipal(principal)
	if err != nil {
		return false
	}
	c.Set(AdminPrincipalContextKey, normalized)
	return true
}

func CurrentAdminPrincipal(c *gin.Context) (AdminPrincipal, bool) {
	value, ok := c.Get(AdminPrincipalContextKey)
	if !ok {
		return AdminPrincipal{}, false
	}
	principal, ok := value.(AdminPrincipal)
	if !ok {
		return AdminPrincipal{}, false
	}
	return cloneAdminPrincipal(principal), true
}

func ContextWithAdminPrincipal(ctx context.Context, principal AdminPrincipal) context.Context {
	return context.WithValue(ctx, adminPrincipalRequestContextKey{}, cloneAdminPrincipal(principal))
}

func AdminPrincipalFromContext(ctx context.Context) (AdminPrincipal, bool) {
	principal, ok := ctx.Value(adminPrincipalRequestContextKey{}).(AdminPrincipal)
	if !ok {
		return AdminPrincipal{}, false
	}
	return cloneAdminPrincipal(principal), true
}

func RequireAdminPermission(permission string) gin.HandlerFunc {
	required := strings.TrimSpace(permission)
	return func(c *gin.Context) {
		principal, ok := CurrentAdminPrincipal(c)
		if !ok {
			AbortWithError(c, http.StatusUnauthorized, "unauthorized", "invalid admin session")
			return
		}
		allowed, knownRole := AdminRolePermissions(principal.Role)
		if required == "" || !knownRole || !containsPermission(allowed, required) || !containsPermission(principal.Permissions, required) {
			AbortWithError(c, http.StatusForbidden, "admin_permission_denied", "admin permission is required")
			return
		}
		c.Next()
	}
}

func normalizeAdminPrincipal(principal AdminPrincipal) (AdminPrincipal, error) {
	email := strings.ToLower(strings.TrimSpace(principal.Email))
	role := strings.TrimSpace(principal.Role)
	permissions, ok := AdminRolePermissions(role)
	if email == "" || !ok {
		return AdminPrincipal{}, ErrInvalidAdminPrincipal
	}
	return AdminPrincipal{Email: email, Role: role, Permissions: permissions}, nil
}

func cloneAdminPrincipal(principal AdminPrincipal) AdminPrincipal {
	principal.Permissions = append([]string(nil), principal.Permissions...)
	return principal
}

func containsPermission(permissions []string, required string) bool {
	for _, permission := range permissions {
		if permission == required {
			return true
		}
	}
	return false
}
