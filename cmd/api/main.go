package main

import (
    "log"
    "os"
    "time"
    
    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/config"
    "cutrix-backend/internal/db"
    "cutrix-backend/internal/handlers"
    "cutrix-backend/internal/middleware"
    "cutrix-backend/internal/repositories"
    "cutrix-backend/internal/services"
    "cutrix-backend/internal/logger"
)

func main() {
    // Structured JSON logger initialized via package init (LOG_LEVEL controls verbosity)

    cfg := config.Load()

    // Initialize services (nil by default if DB not configured)
    var ordersSvc services.OrdersService
    var plansSvc services.PlansService
    var layoutsSvc services.LayoutsService
    var tasksSvc services.TasksService
    var logsSvc services.LogsService
    var usersSvc services.UsersService
    var authSvc services.AuthService

    if cfg.DatabaseURL != "" {
        conn, err := db.Open(cfg.DatabaseURL)
        if err != nil {
            logger.L.Warn("db_connect_failed", "error", err)
        } else {
            defer conn.Close()
            if err := db.RunMigrations(conn, "migrations/000001_initial_schema.up.sql"); err != nil {
                logger.L.Warn("migrations_failed", "error", err)
            } else {
                logger.L.Info("migrations_applied", "script", "000001_initial_schema.up.sql")
            }

            // Wire repositories
            ordersRepo := repositories.NewSqlOrdersRepository(conn)
            plansRepo := repositories.NewSqlPlansRepository(conn)
            layoutsRepo := repositories.NewSqlLayoutsRepository(conn)
            tasksRepo := repositories.NewSqlTasksRepository(conn)
            logsRepo := repositories.NewSqlLogsRepository(conn)
            usersRepo := repositories.NewSqlUsersRepository(conn)

            // Wire services
            ordersSvc = services.NewOrdersService(ordersRepo)
            plansSvc = services.NewPlansService(plansRepo)
            layoutsSvc = services.NewLayoutsService(layoutsRepo)
            tasksSvc = services.NewTasksService(tasksRepo)
            logsSvc = services.NewLogsService(logsRepo)
            usersSvc = services.NewUsersService(usersRepo)

            // Auth service with env-secret and default TTLs
            secret := os.Getenv("AUTH_SECRET")
            if secret == "" { secret = "dev-secret" }
            accessTTL := 15 * time.Minute
            refreshTTL := 7 * 24 * time.Hour
            authSvc = services.NewAuthService(usersRepo, secret, accessTTL, refreshTTL)
            logger.L.Info("startup", "port", cfg.Port, "db", "connected")
        }
    } else {
        logger.L.Warn("db_not_configured")
    }

    r := gin.New()
    // Replace gin.Logger with structured RequestLogger
    // r.Use(gin.Logger())
    r.Use(gin.Recovery())
    r.Use(middleware.CORS())
    r.Use(middleware.RequestID())
    r.Use(middleware.RequestLogger())

    api := r.Group("/api/v1")
    // Register auth and users routes (internally attaches necessary guards)
    handlers.RegisterRoutes(api, authSvc, usersSvc)

    // Mount domain routes under authenticated group when auth is configured
    if authSvc != nil {
        protected := api.Group("")
        protected.Use(middleware.RequireAuth(authSvc))
        handlers.NewOrdersHandler(ordersSvc).RegisterProtected(protected)
        handlers.NewPlansHandler(plansSvc).RegisterProtected(protected)
        handlers.NewLayoutsHandler(layoutsSvc).RegisterProtected(protected)
        handlers.NewTasksHandler(tasksSvc).RegisterProtected(protected)
        handlers.NewLogsHandler(logsSvc).RegisterProtected(protected)
    } else {
        // Fallback for environments without auth (e.g., local dev without DB)
        handlers.NewOrdersHandler(ordersSvc).Register(api)
        handlers.NewPlansHandler(plansSvc).Register(api)
        handlers.NewLayoutsHandler(layoutsSvc).Register(api)
        handlers.NewTasksHandler(tasksSvc).Register(api)
        handlers.NewLogsHandler(logsSvc).Register(api)
    }

    r.NoRoute(func(c *gin.Context) { c.JSON(404, gin.H{"error": "route_not_found"}) })

    logger.L.Info("api_listen", "addr", cfg.Port)
    if err := r.Run(cfg.Port); err != nil {
        logger.L.Error("server_run_failed", "error", err)
        log.Fatal(err)
    }
}