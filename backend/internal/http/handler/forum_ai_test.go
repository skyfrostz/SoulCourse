package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func TestChoiceAdviceReturnsDegradedMetaWhenAIDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	forumHandler := NewForumHandler(nil, service.NewAIService(config.Config{}), false, "", "")
	router := gin.New()
	router.POST("/api/v1/ai/choice-advice", forumHandler.ChoiceAdvice)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/ai/choice-advice",
		bytes.NewBufferString(`{"profile":{"preferredTrack":"physics","preferredSubjects":["chemistry","biology"]},"question":"给我下一步建议"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("X-AI-Fallback"); got != "true" {
		t.Fatalf("expected X-AI-Fallback=true, got %q", got)
	}
	var body struct {
		Data struct {
			Source string `json:"source"`
		} `json:"data"`
		Meta struct {
			Degraded bool `json:"degraded"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Source != "fallback" || !body.Meta.Degraded {
		t.Fatalf("unexpected degraded response: %+v", body)
	}
}
