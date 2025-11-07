package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/services"
)

// RequireAuth validates the access token and injects claims into the context.
// - Parse token from `Authorization: Bearer <token>`
// - Verify signature and expiry
// - Reject if the account is inactive (claims.IsActive=false)
// - Set `claims`, `user_id`, and `role` into gin.Context
func RequireAuth(auth services.AuthService) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := bearerToken(c)
        if token == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
            return
        }
        claims, err := auth.ParseToken(c.Request.Context(), token)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
            return
        }
        if !claims.IsActive {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error":"forbidden"})
            return
        }
        c.Set("claims", claims)
        c.Set("user_id", claims.UserID)
        c.Set("role", claims.Role)
        c.Next()
    }
}

// RequireRoles verifies the current user has at least one of the given roles.
// Must be used after RequireAuth.
func RequireRoles(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        v, ok := c.Get("claims")
        if !ok || v == nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
            return
        }
        claims, ok := v.(*services.Claims)
        if !ok || claims == nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
            return
        }
        // If no roles specified, no extra check is required.
        if len(roles) == 0 {
            c.Next()
            return
        }
        curr := strings.ToLower(claims.Role)
        for _, r := range roles {
            if curr == strings.ToLower(r) {
                c.Next()
                return
            }
        }
        c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error":"forbidden"})
    }
}

// bearerToken extracts the Bearer token from the Authorization header.
func bearerToken(c *gin.Context) string {
    auth := c.GetHeader("Authorization")
    if strings.HasPrefix(auth, "Bearer ") {
        return strings.TrimSpace(auth[7:])
    }
    return ""
}