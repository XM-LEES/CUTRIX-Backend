package handlers

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/services"
)

// AuthHandler provides auth endpoints: login, refresh, change password.
type AuthHandler struct{ svc services.AuthService }

func NewAuthHandler(svc services.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

func (h *AuthHandler) Register(r *gin.RouterGroup) {
    r.POST("/auth/login", h.login)
    r.POST("/auth/refresh", h.refresh)
    r.PUT("/auth/password", h.changePassword)
}

// login authenticates by name/password and returns tokens + user view.
func (h *AuthHandler) login(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    var body struct{
        Name     string `json:"name"`
        Password string `json:"password"`
    }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.Password) == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error":"validation_error", "message":"name and password required"}); return
    }
    tokens, user, err := h.svc.Login(c.Request.Context(), body.Name, body.Password)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, gin.H{"access_token": tokens.AccessToken, "refresh_token": tokens.RefreshToken, "expires_at": tokens.ExpiresAt, "user": user})
}

// refresh issues new tokens from a valid refresh token.
func (h *AuthHandler) refresh(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    var body struct{ RefreshToken string `json:"refresh_token"` }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    tokens, err := h.svc.Refresh(c.Request.Context(), body.RefreshToken)
    if err != nil { writeSvcError(c, err); return }
    c.JSON(http.StatusOK, gin.H{"access_token": tokens.AccessToken, "refresh_token": tokens.RefreshToken, "expires_at": tokens.ExpiresAt})
}

// changePassword updates current user's password after old password check.
func (h *AuthHandler) changePassword(c *gin.Context) {
    if h.svc == nil { c.JSON(http.StatusServiceUnavailable, gin.H{"error":"db_not_configured"}); return }
    token := bearerToken(c)
    if token == "" { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
    claims, err := h.svc.ParseToken(c.Request.Context(), token)
    if err != nil { writeSvcError(c, err); return }

    var body struct{
        OldPassword string `json:"old_password"`
        NewPassword string `json:"new_password"`
    }
    if err := c.ShouldBindJSON(&body); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid_json"}); return }
    if strings.TrimSpace(body.NewPassword) == "" { c.JSON(http.StatusBadRequest, gin.H{"error":"validation_error", "message":"new_password required"}); return }

    if err := h.svc.ChangePassword(c.Request.Context(), claims.UserID, body.OldPassword, body.NewPassword); err != nil { writeSvcError(c, err); return }
    c.Status(http.StatusNoContent)
}