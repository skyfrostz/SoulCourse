package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"subject-choice-forum/backend/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

type envelope gin.H

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, successEnvelope(c, data))
}

func okWithMeta(c *gin.Context, data any, extraMeta envelope) {
	c.JSON(http.StatusOK, successEnvelopeWithMeta(c, data, extraMeta))
}

func created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, successEnvelope(c, data))
}

func fail(c *gin.Context, status int, code string, message string) {
	failWithDetails(c, status, code, message, nil)
}

func failWithDetails(c *gin.Context, status int, code string, message string, details envelope) {
	errorPayload := envelope{
		"code":      code,
		"message":   message,
		"requestId": middleware.GetRequestID(c),
	}
	for key, value := range details {
		errorPayload[key] = value
	}
	c.JSON(status, envelope{
		"error": errorPayload,
	})
}

func successEnvelope(c *gin.Context, data any) envelope {
	return successEnvelopeWithMeta(c, data, nil)
}

func successEnvelopeWithMeta(c *gin.Context, data any, extraMeta envelope) envelope {
	meta := envelope{}
	if requestID := middleware.GetRequestID(c); requestID != "" {
		meta["requestId"] = requestID
	}
	for key, value := range extraMeta {
		meta[key] = value
	}
	payload := envelope{"data": data}
	if len(meta) > 0 {
		payload["meta"] = meta
	}
	return payload
}

func failNotFoundOrInternal(c *gin.Context, err error, resource string) {
	if errors.Is(err, sql.ErrNoRows) {
		fail(c, http.StatusNotFound, "not_found", resource+" not found")
		return
	}
	fail(c, http.StatusInternalServerError, "internal_error", "request failed")
}
