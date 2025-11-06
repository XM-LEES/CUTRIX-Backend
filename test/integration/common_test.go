package integration

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "net/url"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/db"
    "cutrix-backend/internal/handlers"
    "cutrix-backend/internal/middleware"
    "cutrix-backend/internal/repositories"
    "cutrix-backend/internal/services"
)

// Ensure target database exists, creating it if missing
func ensureDatabaseExists(t *testing.T, dsn string) {
    t.Helper()
    u, err := url.Parse(dsn)
    if err != nil { t.Fatalf("parse dsn: %v", err) }
    dbName := strings.TrimPrefix(u.Path, "/")
    u.Path = "/postgres"
    adminDSN := u.String()

    adminConn, err := db.Open(adminDSN)
    if err != nil { t.Fatalf("open admin db: %v", err) }
    defer adminConn.Close()

    createSQL := "CREATE DATABASE \"" + dbName + "\""
    _, execErr := adminConn.Exec(createSQL)
    if execErr != nil {
        if !strings.Contains(strings.ToLower(execErr.Error()), "already exists") {
            t.Fatalf("create database: %v", execErr)
        }
    }
}

// Locate migrations path relative to repo root
func migrationsPath(t *testing.T, file string) string {
    t.Helper()
    wd, err := os.Getwd()
    if err != nil { t.Fatalf("getwd: %v", err) }
    dir := wd
    for i := 0; i < 6; i++ {
        gm := filepath.Join(dir, "go.mod")
        if _, err := os.Stat(gm); err == nil {
            return filepath.Join(dir, "migrations", file)
        }
        nd := filepath.Dir(dir)
        if nd == dir { break }
        dir = nd
    }
    t.Fatalf("cannot locate repo root with go.mod from %s", wd)
    return ""
}

// Open DB and apply migrations
func openDBAndMigrate(t *testing.T) *sql.DB {
    t.Helper()
    gin.SetMode(gin.TestMode)
    dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
    if dsn == "" { t.Fatalf("env DATABASE_URL is empty") }
    ensureDatabaseExists(t, dsn)
    conn, err := db.Open(dsn)
    if err != nil { t.Fatalf("db open: %v", err) }
    mp := migrationsPath(t, "000001_initial_schema.up.sql")
    if err := db.RunMigrations(conn, mp); err != nil {
        t.Fatalf("migrate: %v", err)
    }
    return conn
}

// Build a router with all API routes registered
func buildRouter(conn *sql.DB) *gin.Engine {
    r := gin.New()
    r.Use(middleware.RequestID())
    api := r.Group("/api/v1")

    ordersRepo := repositories.NewSqlOrdersRepository(conn)
    plansRepo := repositories.NewSqlPlansRepository(conn)
    layoutsRepo := repositories.NewSqlLayoutsRepository(conn)
    tasksRepo := repositories.NewSqlTasksRepository(conn)
    logsRepo := repositories.NewSqlLogsRepository(conn)

    handlers.NewOrdersHandler(services.NewOrdersService(ordersRepo)).Register(api)
    handlers.NewPlansHandler(services.NewPlansService(plansRepo)).Register(api)
    handlers.NewLayoutsHandler(services.NewLayoutsService(layoutsRepo)).Register(api)
    handlers.NewTasksHandler(services.NewTasksService(tasksRepo)).Register(api)
    handlers.NewLogsHandler(services.NewLogsService(logsRepo)).Register(api)
    return r
}

// Small helper to perform JSON HTTP requests against the router
func doJSON(r *gin.Engine, method, path string, body string) (*httptest.ResponseRecorder, *http.Request) {
    var reader *strings.Reader
    if body != "" { reader = strings.NewReader(body) } else { reader = strings.NewReader("") }
    req, _ := http.NewRequest(method, path, reader)
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    return w, req
}

// Decode JSON helper
func decodeJSON[T any](t *testing.T, w *httptest.ResponseRecorder, out *T) {
    t.Helper()
    if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
        t.Fatalf("json decode: %v; body=%s", err, w.Body.String())
    }
}