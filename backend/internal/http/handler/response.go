package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type envelope gin.H

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, envelope{"data": data})
}

func created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, envelope{"data": data})
}

func fail(c *gin.Context, status int, code string, message string) {
	failWithDetails(c, status, code, message, nil)
}

func failWithDetails(c *gin.Context, status int, code string, message string, details envelope) {
	errorPayload := envelope{
		"code":    code,
		"message": message,
	}
	for key, value := range details {
		errorPayload[key] = value
	}
	c.JSON(status, envelope{
		"error": errorPayload,
	})
}

func failNotFoundOrInternal(c *gin.Context, err error, resource string) {
	if errors.Is(err, sql.ErrNoRows) {
		fail(c, http.StatusNotFound, "not_found", resource+" not found")
		return
	}
	fail(c, http.StatusInternalServerError, "internal_error", "request failed")
}
