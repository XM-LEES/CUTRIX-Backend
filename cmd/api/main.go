package main

import (
    "log"

    "github.com/gin-gonic/gin"

    "example/internal/handlers"
    "example/internal/middleware"
)

func main() {
    r := gin.New()
    r.Use(gin.Logger())
    r.Use(gin.Recovery())
    r.Use(middleware.CORS())

    // 路由分组示例（不包含业务）
    api := r.Group("/api/v1")

    // 健康检查挂载（示例）
    handlers.NewHealthHandler().Register(api)

    // 兜底路由
    r.NoRoute(func(c *gin.Context) {
        c.JSON(404, gin.H{"error": "route_not_found"})
    })

    // 端口可按需从环境变量或配置加载
    if err := r.Run(":8080"); err != nil {
        log.Fatal(err)
    }
}