package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, envelope{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := envelope{}
	if err := h.db.PingContext(ctx); err != nil {
		checks["sqlite"] = "down"
		fail(c, http.StatusServiceUnavailable, "dependency_unavailable", "sqlite is unavailable")
		return
	}
	checks["sqlite"] = "ok"

	c.JSON(http.StatusOK, envelope{
		"status": "ready",
		"checks": checks,
	})
}
