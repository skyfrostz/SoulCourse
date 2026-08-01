package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

func TestHealthLiveAndDefaultTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandlerWithDatabase(nil, "sqlite", 0)
	if handler.timeout <= 0 {
		t.Fatal("health handler did not apply a positive default timeout")
	}
	router := gin.New()
	router.GET("/livez", handler.Live)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("live status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Status string `json:"status"`
		Time   string `json:"time"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != "ok" || payload.Time == "" {
		t.Fatalf("unexpected live payload: %+v", payload)
	}
}

func TestHealthReadyUsesGenericDatabaseCheck(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/readyz", NewHealthHandler(db).Ready)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("ready status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if body := recorder.Body.String(); !strings.Contains(body, `"database":"ok"`) {
		t.Fatalf("ready response does not expose generic database check: %s", body)
	}
}

func TestHealthReadyReportsClosedDatabase(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/readyz", NewHealthHandler(db).Ready)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), `"code":"dependency_unavailable"`) {
		t.Fatalf("ready status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
