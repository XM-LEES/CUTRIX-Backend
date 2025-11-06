package middleware

import (
    "crypto/rand"
    "encoding/hex"
    "github.com/gin-gonic/gin"
)

// RequestID 将请求ID注入上下文与响应头，便于日志关联
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