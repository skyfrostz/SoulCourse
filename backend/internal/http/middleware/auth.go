package middleware

import (
	"net/http"
	"strings"

	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-gonic/gin"
)

const CurrentUserKey = "currentUser"

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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "unauthorized",
					"message": "please login first",
				},
			})
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
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return domain.User{}, false
	}
	user, err := forumService.UserFromToken(c.Request.Context(), strings.TrimPrefix(header, "Bearer "))
	return user, err == nil
}
