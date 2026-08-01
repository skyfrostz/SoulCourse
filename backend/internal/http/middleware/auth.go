package middleware

import (
	"net/http"
	"strings"

	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-gonic/gin"
)

const CurrentUserKey = "currentUser"
const SessionCookieName = "scf_session"
const CSRFCookieName = "scf_csrf"
const CSRFHeaderName = "X-CSRF-Token"

func OptionalAuth(forumService *service.ForumService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := userFromAuthorization(c, forumService)
		if ok {
			c.Set(CurrentUserKey, user)
		}
		c.Next()
	}
}

func RequireAuth(forumService *service.ForumService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := userFromAuthorization(c, forumService)
		if !ok {
			AbortWithError(c, http.StatusUnauthorized, "unauthorized", "please login first")
			return
		}
		c.Set(CurrentUserKey, user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (domain.User, bool) {
	value, ok := c.Get(CurrentUserKey)
	if !ok {
		return domain.User{}, false
	}
	user, ok := value.(domain.User)
	return user, ok
}

func CurrentUserID(c *gin.Context) *int64 {
	user, ok := CurrentUser(c)
	if !ok {
		return nil
	}
	return &user.ID
}

func userFromAuthorization(c *gin.Context, forumService *service.ForumService) (domain.User, bool) {
	token, ok := SessionToken(c)
	if !ok {
		return domain.User{}, false
	}
	user, err := forumService.UserFromToken(c.Request.Context(), token)
	return user, err == nil
}

func CSRFProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		if csrfSafeMethod(c.Request.Method) {
			c.Next()
			return
		}
		if strings.HasSuffix(c.Request.URL.Path, "/admin/login") || strings.HasSuffix(c.Request.URL.Path, "/telemetry/web-vitals") {
			c.Next()
			return
		}
		_, userCookieErr := c.Request.Cookie(SessionCookieName)
		_, adminCookieErr := c.Request.Cookie(AdminSessionCookieName)
		hasUserCookie := userCookieErr == nil
		hasAdminCookie := adminCookieErr == nil
		if c.GetHeader("Authorization") != "" && !hasUserCookie && !hasAdminCookie {
			c.Next()
			return
		}
		cookieName := CSRFCookieName
		if protectedAdminWritePath(c.Request.URL.Path) {
			cookieName = AdminCSRFCookieName
		} else if _, ok := CurrentUser(c); !ok {
			c.Next()
			return
		}
		cookieToken, err := c.Cookie(cookieName)
		if err != nil || strings.TrimSpace(cookieToken) == "" {
			AbortWithError(c, http.StatusForbidden, "csrf_required", "CSRF token is required")
			return
		}
		headerToken := strings.TrimSpace(c.GetHeader(CSRFHeaderName))
		if headerToken == "" || headerToken != cookieToken {
			AbortWithError(c, http.StatusForbidden, "csrf_invalid", "CSRF token is invalid")
			return
		}
		c.Next()
	}
}

func protectedAdminWritePath(path string) bool {
	return strings.Contains(path, "/admin/") && !strings.HasSuffix(path, "/admin/login")
}

func csrfSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func BearerToken(c *gin.Context) (string, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	return token, token != ""
}

func SessionToken(c *gin.Context) (string, bool) {
	if token, err := c.Cookie(SessionCookieName); err == nil && strings.TrimSpace(token) != "" {
		return strings.TrimSpace(token), true
	}
	return BearerToken(c)
}

func SetSessionCookie(c *gin.Context, token string, maxAgeSeconds int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAgeSeconds,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookie(c *gin.Context, secure bool) {
	SetSessionCookie(c, "", -1, secure)
}

func SetCSRFCookie(c *gin.Context, token string, maxAgeSeconds int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAgeSeconds,
		Secure:   secure,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearCSRFCookie(c *gin.Context, secure bool) {
	SetCSRFCookie(c, "", -1, secure)
}
