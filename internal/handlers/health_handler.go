package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

// HealthHandler 仅用于示例，真实项目可扩展为更多系统检查
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler { return &HealthHandler{} }

// Register 将健康检查挂载到给定路由组
func (h *HealthHandler) Register(r *gin.RouterGroup) {
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "ok"})
    })
}