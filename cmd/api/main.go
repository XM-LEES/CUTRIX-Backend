package main

import (
    "database/sql"
    "log"

    "github.com/gin-gonic/gin"

    _ "modernc.org/sqlite"

    "example-demo/internal/config"
    "example-demo/internal/handlers"
    "example-demo/internal/logger"
    "example-demo/internal/middleware"
    "example-demo/internal/repositories"
    "example-demo/internal/services"
)

func main() {
    // 加载最小配置
    cfg := config.Load()

    // 打开 SQLite 数据库（会在当前目录生成 db 文件）
    db, err := sql.Open("sqlite", "file:"+cfg.DBPath+"?cache=shared&mode=rwc")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // 启用外键（SQLite 默认关闭）
    if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
        log.Println("warn: enable foreign_keys failed:", err)
    }

    // 初始化表
    userRepo := repositories.NewUserRepository(db)
    if err := userRepo.InitSchema(); err != nil {
        log.Fatal(err)
    }
    todoRepo := repositories.NewTodoRepository(db)
    if err := todoRepo.InitSchema(); err != nil {
        log.Fatal(err)
    }

    // 组装服务
    todoSvc := services.NewTodoService(todoRepo)
    userSvc := services.NewUserService(userRepo)

    // 结构化日志（最小 JSON 行）
    _ = logger.New()

    // 启动 HTTP 路由
    r := gin.New()
    r.Use(gin.Logger())
    r.Use(gin.Recovery())
    r.Use(middleware.CORS())
    r.Use(middleware.RequestID())

    api := r.Group("/api/v1")
    // 健康检查
    api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
    // 注册 Users CRUD 与 Todos CRUD
    handlers.NewUserHandler(userSvc).Register(api)
    handlers.NewTodoHandler(todoSvc).Register(api)

    // 兜底
    r.NoRoute(func(c *gin.Context) { c.JSON(404, gin.H{"error": "route_not_found"}) })

    if err := r.Run(cfg.Port); err != nil {
        log.Fatal(err)
    }
}