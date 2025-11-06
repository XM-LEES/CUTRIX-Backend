package main

import (
    "log"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/config"
    "cutrix-backend/internal/db"
    "cutrix-backend/internal/handlers"
    "cutrix-backend/internal/middleware"
    "cutrix-backend/internal/repositories"
    "cutrix-backend/internal/services"
)

func main() {
    cfg := config.Load()

    // Initialize services (nil by default if DB not configured)
    var ordersSvc services.OrdersService
    var plansSvc services.PlansService
    var layoutsSvc services.LayoutsService
    var tasksSvc services.TasksService
    var logsSvc services.LogsService

    if cfg.DatabaseURL != "" {
        conn, err := db.Open(cfg.DatabaseURL)
        if err != nil {
            log.Println("warn: database connect failed:", err)
        } else {
            defer conn.Close()
            if err := db.RunMigrations(conn, "migrations/000001_initial_schema.up.sql"); err != nil {
                log.Println("warn: migrations failed:", err)
            }

            // Wire repositories
            ordersRepo := repositories.NewSqlOrdersRepository(conn)
            plansRepo := repositories.NewSqlPlansRepository(conn)
            layoutsRepo := repositories.NewSqlLayoutsRepository(conn)
            tasksRepo := repositories.NewSqlTasksRepository(conn)
            logsRepo := repositories.NewSqlLogsRepository(conn)

            // Wire services
            ordersSvc = services.NewOrdersService(ordersRepo)
            plansSvc = services.NewPlansService(plansRepo)
            layoutsSvc = services.NewLayoutsService(layoutsRepo)
            tasksSvc = services.NewTasksService(tasksRepo)
            logsSvc = services.NewLogsService(logsRepo)
        }
    } else {
        log.Println("warn: DATABASE_URL not set; business endpoints will return db_not_configured")
    }

    r := gin.New()
    r.Use(gin.Logger())
    r.Use(gin.Recovery())
    r.Use(middleware.CORS())
    r.Use(middleware.RequestID())

    api := r.Group("/api/v1")
    handlers.NewHealthHandler().Register(api)
    handlers.NewOrdersHandler(ordersSvc).Register(api)
    handlers.NewPlansHandler(plansSvc).Register(api)
    handlers.NewLayoutsHandler(layoutsSvc).Register(api)
    handlers.NewTasksHandler(tasksSvc).Register(api)
    handlers.NewLogsHandler(logsSvc).Register(api)

    r.NoRoute(func(c *gin.Context) { c.JSON(404, gin.H{"error": "route_not_found"}) })

    if err := r.Run(cfg.Port); err != nil {
        log.Fatal(err)
    }
}