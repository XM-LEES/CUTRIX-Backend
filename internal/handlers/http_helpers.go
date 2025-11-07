package handlers

import (
    "database/sql"
    "errors"
    "net/http"
    "strings"
    "log/slog"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/services"
    "cutrix-backend/internal/logger"
)

// bearerToken extracts the Bearer token from Authorization header.
func bearerToken(c *gin.Context) string {
    auth := c.GetHeader("Authorization")
    if strings.HasPrefix(auth, "Bearer ") { return strings.TrimSpace(auth[7:]) }
    return ""
}

// writeSvcError maps service-layer errors to HTTP responses and logs accordingly.
// Logging policy:
// - 404: info (normal not-found); 4xx: warn (user/action issue); 5xx: error (server fault)
// Logged fields: method, path, status_code, error, request_id, user_id.
func writeSvcError(c *gin.Context, err error) {
    var status int
    switch {
    case errors.Is(err, services.ErrUnauthorized):
        status = http.StatusUnauthorized
        c.JSON(status, gin.H{"error":"unauthorized"})
    case errors.Is(err, services.ErrForbidden):
        status = http.StatusForbidden
        c.JSON(status, gin.H{"error":"forbidden"})
    case errors.Is(err, services.ErrConflict):
        status = http.StatusConflict
        c.JSON(status, gin.H{"error":"conflict"})
    case errors.Is(err, services.ErrNotFound) || errors.Is(err, sql.ErrNoRows):
        status = http.StatusNotFound
        c.JSON(status, gin.H{"error":"not_found"})
    case errors.Is(err, services.ErrValidation):
        status = http.StatusBadRequest
        c.JSON(status, gin.H{"error":"validation_error"})
    default:
        status = http.StatusInternalServerError
        c.JSON(status, gin.H{"error":"internal_error", "message": err.Error()})
    }
    rid, _ := c.Get("request_id")
    uid, _ := c.Get("user_id")
    attrs := []slog.Attr{
        slog.String("method", c.Request.Method),
        slog.String("path", c.FullPath()),
        slog.Int("status_code", status),
        slog.String("error", err.Error()),
        slog.Any("request_id", rid),
        slog.Any("user_id", uid),
    }
    // convert []slog.Attr to []any for slog methods
    args := make([]any, 0, len(attrs))
    for _, a := range attrs { args = append(args, a) }
    switch {
    case status == http.StatusNotFound:
        logger.L.Info("http_error", args...)
    case status >= 400 && status < 500:
        logger.L.Warn("http_error", args...)
    default:
        logger.L.Error("http_error", args...)
    }
}