package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redisClient}
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
	if err := h.db.Ping(ctx); err != nil {
		checks["postgres"] = "down"
		fail(c, http.StatusServiceUnavailable, "dependency_unavailable", "postgres is unavailable")
		return
	}
	checks["postgres"] = "ok"

	if err := h.redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "down"
		fail(c, http.StatusServiceUnavailable, "dependency_unavailable", "redis is unavailable")
		return
	}
	checks["redis"] = "ok"

	c.JSON(http.StatusOK, envelope{
		"status": "ready",
		"checks": checks,
	})
}
