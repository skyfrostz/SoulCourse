package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"subject-choice-forum/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	db      *sql.DB
	driver  string
	timeout time.Duration
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return NewHealthHandlerWithTimeout(db, 2*time.Second)
}

func NewHealthHandlerWithTimeout(db *sql.DB, timeout time.Duration) *HealthHandler {
	return NewHealthHandlerWithDatabase(db, "sqlite", timeout)
}

func NewHealthHandlerWithDatabase(db *sql.DB, driver string, timeout time.Duration) *HealthHandler {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &HealthHandler{db: db, driver: driver, timeout: timeout}
}

func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, envelope{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	checks := envelope{}
	if err := h.db.PingContext(ctx); err != nil {
		checks["database"] = "down"
		fail(c, http.StatusServiceUnavailable, "dependency_unavailable", "database is unavailable")
		return
	}
	checks["database"] = "ok"
	if h.driver == "postgres" {
		if err := storage.VerifyPostgresSchema(ctx, h.db); err != nil {
			fail(c, http.StatusServiceUnavailable, "schema_unavailable", "database schema is unavailable")
			return
		}
		checks["schema"] = "ok"
	}

	payload := envelope{"status": "ready", "checks": checks}
	if h.driver == "postgres" {
		payload["schemaVersion"] = storage.RequiredPostgresSchemaVersion
	}
	c.JSON(http.StatusOK, payload)
}
