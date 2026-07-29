package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func TestPostInteractionsRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	forum := service.NewForumService(nil, config.Config{JWTSecret: "test-secret"}, nil)
	router := gin.New()
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
		if recorder.Body.String() != `{"error":{"code":"unauthorized","message":"please login first"}}` {
			t.Fatalf("%s returned unexpected envelope: %s", path, recorder.Body.String())
		}
	}
}
