package middleware

import (
    "crypto/rand"
    "encoding/hex"
    "github.com/gin-gonic/gin"
)

// RequestID injects a request ID into context and response headers for log correlation.
func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        rid := c.GetHeader("X-Request-ID")
        if rid == "" {
            b := make([]byte, 16)
            if _, err := rand.Read(b); err == nil {
                rid = hex.EncodeToString(b)
            } else {
                rid = "rid-unknown"
            }
        }
        c.Set("request_id", rid)
        c.Writer.Header().Set("X-Request-ID", rid)
        c.Next()
    }
}