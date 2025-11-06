package integration

import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "os"
    "strings"
    "testing"
    "time"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/db"
    "cutrix-backend/internal/handlers"
    "cutrix-backend/internal/middleware"
    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
    "cutrix-backend/internal/services"
)

func TestOrders_Create_And_Get(t *testing.T) {
    gin.SetMode(gin.TestMode)

    dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
    if dsn == "" { t.Fatalf("env DATABASE_URL is empty") }

    // Ensure target database exists (connect to admin DB and create if missing)
    ensureDatabaseExists(t, dsn)

    conn, err := db.Open(dsn)
    if err != nil { t.Fatalf("db open: %v", err) }
    t.Cleanup(func(){ conn.Close() })

    // Apply migrations (idempotent)
    mp := migrationsPath(t, "000001_initial_schema.up.sql")
    if err := db.RunMigrations(conn, mp); err != nil {
        t.Fatalf("migrate: %v", err)
    }

    // Wire repository + service
    ordersRepo := repositories.NewSqlOrdersRepository(conn)
    ordersSvc := services.NewOrdersService(ordersRepo)

    // Build router
    r := gin.New()
    r.Use(middleware.RequestID())
    api := r.Group("/api/v1")
    handlers.NewOrdersHandler(ordersSvc).Register(api)

    // Prepare a unique order_number to avoid conflicts
    orderNumber := fmt.Sprintf("ORD-IT-%d", time.Now().UnixNano())

    // Create order
    reqBody := strings.NewReader(fmt.Sprintf(`{"order_number":"%s","style_number":"STY-001","customer_name":"ACME"}`, orderNumber))
    req, _ := http.NewRequest(http.MethodPost, "/api/v1/orders", reqBody)
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusCreated { t.Fatalf("create code=%d body=%s", w.Code, w.Body.String()) }

    var created models.ProductionOrder
    if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
        t.Fatalf("json: %v", err)
    }
    if created.OrderID == 0 { t.Fatalf("expected non-zero order_id") }

    // Get order
    getReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/orders/%d", created.OrderID), nil)
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, getReq)
    if w2.Code != http.StatusOK { t.Fatalf("get code=%d body=%s", w2.Code, w2.Body.String()) }

    var fetched models.ProductionOrder
    if err := json.Unmarshal(w2.Body.Bytes(), &fetched); err != nil {
        t.Fatalf("json get: %v", err)
    }
    if fetched.OrderNumber != orderNumber { t.Fatalf("order_number mismatch: got=%s want=%s", fetched.OrderNumber, orderNumber) }
}