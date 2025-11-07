package middleware

import (
    "time"
    "log/slog"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/logger"
)

// RequestLogger is a Gin middleware that logs structured HTTP request/response details.
// Fields: method, path, status_code, duration_ms, client_ip, request_id, user_id.
// Level policy: <400: info, 4xx: warn, >=500: error.
func RequestLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        dur := time.Since(start)

        rid, _ := c.Get("request_id")
        uid, _ := c.Get("user_id")
        status := c.Writer.Status()
        attrs := []slog.Attr{
            slog.String("method", c.Request.Method),
            slog.String("path", c.FullPath()),
            slog.Int("status_code", status),
            slog.Int64("duration_ms", dur.Milliseconds()),
            slog.String("client_ip", c.ClientIP()),
            slog.Any("request_id", rid),
            slog.Any("user_id", uid),
        }
        // convert []slog.Attr to []any for slog methods
        args := make([]any, 0, len(attrs))
        for _, a := range attrs { args = append(args, a) }
        switch {
        case status < 400:
            logger.L.Info("http_request", args...)
        case status >= 400 && status < 500:
            logger.L.Warn("http_request", args...)
        default:
            logger.L.Error("http_request", args...)
        }
    }
}