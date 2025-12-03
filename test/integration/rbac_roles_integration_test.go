package integration

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "testing"
    "time"

    "github.com/gin-gonic/gin"

    "cutrix-backend/internal/handlers"
    "cutrix-backend/internal/middleware"
    "cutrix-backend/internal/repositories"
    "cutrix-backend/internal/services"
)

// buildProtectedRouter registers auth/users and then mounts domain routes under RequireAuth.
func buildProtectedRouter(conn *sql.DB) *gin.Engine {
    r := gin.New()
    r.Use(middleware.RequestID())
    api := r.Group("/api/v1")

    usersRepo := repositories.NewSqlUsersRepository(conn)
    authSvc := services.NewAuthService(usersRepo, "test-secret", time.Minute, 24*time.Hour)
    usersSvc := services.NewUsersService(usersRepo)
    handlers.RegisterRoutes(api, authSvc, usersSvc)

    ordersRepo := repositories.NewSqlOrdersRepository(conn)
    plansRepo := repositories.NewSqlPlansRepository(conn)
    layoutsRepo := repositories.NewSqlLayoutsRepository(conn)
    tasksRepo := repositories.NewSqlTasksRepository(conn)
    logsRepo := repositories.NewSqlLogsRepository(conn)

    protected := api.Group("")
    protected.Use(middleware.RequireAuth(authSvc))
    handlers.NewOrdersHandler(services.NewOrdersService(ordersRepo)).RegisterProtected(protected)
    handlers.NewPlansHandler(services.NewPlansService(plansRepo)).RegisterProtected(protected)
    handlers.NewLayoutsHandler(services.NewLayoutsService(layoutsRepo)).RegisterProtected(protected)
    handlers.NewTasksHandler(services.NewTasksService(tasksRepo)).RegisterProtected(protected)
    handlers.NewLogsHandler(services.NewLogsService(logsRepo)).RegisterProtected(protected)
    return r
}

// login returns access token and user object.
func login(t *testing.T, r *gin.Engine, name, password string) (string, struct{ UserID int; Name string; Role string }) {
    t.Helper()
    w, _ := doJSONAuth(r, http.MethodPost, "/api/v1/auth/login", fmt.Sprintf(`{"name":"%s","password":"%s"}`, name, password), "")
    if w.Code != http.StatusOK { t.Fatalf("login code=%d body=%s", w.Code, w.Body.String()) }
    var resp struct{
        AccessToken string `json:"access_token"`
        RefreshToken string `json:"refresh_token"`
        ExpiresAt string `json:"expires_at"`
        User struct{ UserID int; Name string; Role string } `json:"user"`
    }
    if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil { t.Fatalf("json login: %v", err) }
    return resp.AccessToken, resp.User
}

// seedOrder creates an order; returns order_id.
func seedOrder(t *testing.T, r *gin.Engine, token string) int {
    t.Helper()
    orderNumber := fmt.Sprintf("ORD-%d", time.Now().UTC().UnixNano())
    start := time.Now().UTC().Format(time.RFC3339)
    body := fmt.Sprintf(`{
        "order_number": "%s",
        "style_number": "STYLE-RBAC",
        "customer_name": "ACME",
        "order_start_date": "%s",
        "items": [ {"color":"Red","size":"M","quantity":3} ]
    }`, orderNumber, start)
    w, _ := doJSONAuth(r, http.MethodPost, "/api/v1/orders", body, token)
    if w.Code != http.StatusCreated { t.Fatalf("create order code=%d body=%s", w.Code, w.Body.String()) }
    var created struct{ OrderID int `json:"order_id"` }
    decodeJSON(t, w, &created)
    return created.OrderID
}

// seedPlanLayoutTask creates plan, layout, task and publishes the plan; returns ids.
func seedPlanLayoutTask(t *testing.T, r *gin.Engine, token string, orderID int) (int, int, int) {
    t.Helper()
    // plan
    w, _ := doJSONAuth(r, http.MethodPost, "/api/v1/plans", fmt.Sprintf(`{"plan_name":"Plan-RBAC","order_id":%d}`, orderID), token)
    if w.Code != http.StatusCreated { t.Fatalf("create plan code=%d body=%s", w.Code, w.Body.String()) }
    var plan struct{ PlanID int `json:"plan_id"` }
    decodeJSON(t, w, &plan)

    // layout
    w, _ = doJSONAuth(r, http.MethodPost, "/api/v1/layouts", fmt.Sprintf(`{"layout_name":"L-RBAC","plan_id":%d}`, plan.PlanID), token)
    if w.Code != http.StatusCreated { t.Fatalf("create layout code=%d body=%s", w.Code, w.Body.String()) }
    var layout struct{ LayoutID int `json:"layout_id"` }
    decodeJSON(t, w, &layout)

    // task
    w, _ = doJSONAuth(r, http.MethodPost, "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Red","planned_layers":3}`, layout.LayoutID), token)
    if w.Code != http.StatusCreated { t.Fatalf("create task code=%d body=%s", w.Code, w.Body.String()) }
    var task struct{ TaskID int `json:"task_id"` }
    decodeJSON(t, w, &task)

    // publish plan to mark tasks in_progress
    w, _ = doJSONAuth(r, http.MethodPost, fmt.Sprintf("/api/v1/plans/%d/publish", plan.PlanID), "{}", token)
    if w.Code != http.StatusNoContent { t.Fatalf("publish plan code=%d body=%s", w.Code, w.Body.String()) }

    return plan.PlanID, layout.LayoutID, task.TaskID
}

func TestRBAC_Manager_AllAccess(t *testing.T) {
    conn := openDBAndMigrate(t)
    t.Cleanup(func(){ conn.Close() })
    r := buildProtectedRouter(conn)

    // Seed manager
    mgrName := fmt.Sprintf("manager_%d", time.Now().UnixNano())
    _ = createUser(t, conn, mgrName, "manager", "Mgr123!")
    mgrToken, _ := login(t, r, mgrName, "Mgr123!")

    // Manager can create full chain and list logs
    orderID := seedOrder(t, r, mgrToken)
    _, _, taskID := seedPlanLayoutTask(t, r, mgrToken, orderID)

    // create log
    logBody := fmt.Sprintf(`{"task_id":%d,"layers_completed":1,"log_time":"%s"}`, taskID, time.Now().UTC().Format(time.RFC3339))
    w, _ := doJSONAuth(r, http.MethodPost, "/api/v1/logs", logBody, mgrToken)
    if w.Code != http.StatusCreated { t.Fatalf("manager create log code=%d body=%s", w.Code, w.Body.String()) }
    var log struct{ LogID int `json:"log_id"` }
    decodeJSON(t, w, &log)

    // list logs by task (manager only)
    w, _ = doJSONAuth(r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d/logs", taskID), "", mgrToken)
    if w.Code != http.StatusOK { t.Fatalf("manager list logs code=%d body=%s", w.Code, w.Body.String()) }
}

func TestRBAC_Worker_LogAndTaskScopes(t *testing.T) {
    conn := openDBAndMigrate(t)
    t.Cleanup(func(){ conn.Close() })
    r := buildProtectedRouter(conn)

    // Seed manager to prepare data
    mgrName := fmt.Sprintf("manager_%d", time.Now().UnixNano())
    _ = createUser(t, conn, mgrName, "manager", "Mgr123!")
    mgrToken, _ := login(t, r, mgrName, "Mgr123!")

    // Prepare chain
    orderID := seedOrder(t, r, mgrToken)
    planID, layoutID, taskID := seedPlanLayoutTask(t, r, mgrToken, orderID)

    // Seed worker and login
    workerName := fmt.Sprintf("worker_%d", time.Now().UnixNano())
    workerID := createUser(t, conn, workerName, "worker", "Wkr123!")
    workerToken, worker := login(t, r, workerName, "Wkr123!")
    if worker.UserID != workerID { t.Fatalf("worker id mismatch") }

    // Worker can create a log
    logBody := fmt.Sprintf(`{"task_id":%d,"layers_completed":2,"log_time":"%s"}`, taskID, time.Now().UTC().Format(time.RFC3339))
    w, _ := doJSONAuth(r, http.MethodPost, "/api/v1/logs", logBody, workerToken)
    if w.Code != http.StatusCreated { t.Fatalf("worker create log code=%d body=%s", w.Code, w.Body.String()) }
    var log struct{ LogID int `json:"log_id"` }
    decodeJSON(t, w, &log)

    // Worker can void a log
    voidBody := fmt.Sprintf(`{"void_reason":"mistake","voided_by":%d}`, worker.UserID)
    w, _ = doJSONAuth(r, http.MethodPatch, fmt.Sprintf("/api/v1/logs/%d", log.LogID), voidBody, workerToken)
    if w.Code != http.StatusNoContent { t.Fatalf("worker void log code=%d body=%s", w.Code, w.Body.String()) }

    // Worker cannot list logs
    w, _ = doJSONAuth(r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d/logs", taskID), "", workerToken)
    if w.Code != http.StatusForbidden { t.Fatalf("worker list logs should be 403, got %d", w.Code) }

    // Worker can read task and list tasks
    w, _ = doJSONAuth(r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", taskID), "", workerToken)
    if w.Code != http.StatusOK { t.Fatalf("worker read task code=%d body=%s", w.Code, w.Body.String()) }

    w, _ = doJSONAuth(r, http.MethodGet, fmt.Sprintf("/api/v1/layouts/%d/tasks", layoutID), "", workerToken)
    if w.Code != http.StatusOK { t.Fatalf("worker list tasks code=%d body=%s", w.Code, w.Body.String()) }

    // Use planID to avoid unused warning
    _ = planID
}

func TestRBAC_PatternMaker_ModuleAccess(t *testing.T) {
    conn := openDBAndMigrate(t)
    t.Cleanup(func(){ conn.Close() })
    r := buildProtectedRouter(conn)

    // Seed manager to create order
    mgrName := fmt.Sprintf("manager_%d", time.Now().UnixNano())
    _ = createUser(t, conn, mgrName, "manager", "Mgr123!")
    mgrToken, _ := login(t, r, mgrName, "Mgr123!")
    orderID := seedOrder(t, r, mgrToken)

    // Seed pattern_maker and login
    pmName := fmt.Sprintf("pm_%d", time.Now().UnixNano())
    _ = createUser(t, conn, pmName, "pattern_maker", "Pm123!")
    pmToken, _ := login(t, r, pmName, "Pm123!")

    // Create plan and layout without publish
    w, _ := doJSONAuth(r, http.MethodPost, "/api/v1/plans", fmt.Sprintf(`{"plan_name":"Plan-RBAC","order_id":%d}`, orderID), pmToken)
    if w.Code != http.StatusCreated { t.Fatalf("pm create plan code=%d body=%s", w.Code, w.Body.String()) }
    var plan struct{ PlanID int `json:"plan_id"` }
    decodeJSON(t, w, &plan)

    w, _ = doJSONAuth(r, http.MethodPost, "/api/v1/layouts", fmt.Sprintf(`{"layout_name":"L-RBAC","plan_id":%d}`, plan.PlanID), pmToken)
    if w.Code != http.StatusCreated { t.Fatalf("pm create layout code=%d body=%s", w.Code, w.Body.String()) }
    var layout struct{ LayoutID int `json:"layout_id"` }
    decodeJSON(t, w, &layout)

    // Update layout name BEFORE publish
    w, _ = doJSONAuth(r, http.MethodPatch, fmt.Sprintf("/api/v1/layouts/%d/name", layout.LayoutID), `{"name":"L-RBAC-2"}`, pmToken)
    if w.Code != http.StatusNoContent { t.Fatalf("pm update layout name code=%d body=%s", w.Code, w.Body.String()) }

    // Create a task
    w, _ = doJSONAuth(r, http.MethodPost, "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Red","planned_layers":3}`, layout.LayoutID), pmToken)
    if w.Code != http.StatusCreated { t.Fatalf("pm create task code=%d body=%s", w.Code, w.Body.String()) }
    var task struct{ TaskID int `json:"task_id"` }
    decodeJSON(t, w, &task)

    // Publish plan
    w, _ = doJSONAuth(r, http.MethodPost, fmt.Sprintf("/api/v1/plans/%d/publish", plan.PlanID), "{}", pmToken)
    if w.Code != http.StatusNoContent { t.Fatalf("pm publish plan code=%d body=%s", w.Code, w.Body.String()) }

    // Update plan note after publish is allowed
    w, _ = doJSONAuth(r, http.MethodPatch, fmt.Sprintf("/api/v1/plans/%d/note", plan.PlanID), `{"note":"updated"}`, pmToken)
    if w.Code != http.StatusNoContent { t.Fatalf("pm update plan note code=%d body=%s", w.Code, w.Body.String()) }

    // Read task
    w, _ = doJSONAuth(r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", task.TaskID), "", pmToken)
    if w.Code != http.StatusOK { t.Fatalf("pm read task code=%d body=%s", w.Code, w.Body.String()) }

    // Pattern maker cannot list logs (restricted to admin/manager)
    w, _ = doJSONAuth(r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d/logs", task.TaskID), "", pmToken)
    if w.Code != http.StatusForbidden { t.Fatalf("pm list logs should be 403, got %d", w.Code) }
}

func TestRBAC_Admin_OrdersAndLogs(t *testing.T) {
    conn := openDBAndMigrate(t)
    t.Cleanup(func(){ conn.Close() })
    r := buildProtectedRouter(conn)

    // Seed admin and login
    admName := fmt.Sprintf("admin_%d", time.Now().UnixNano())
    _ = createUser(t, conn, admName, "admin", "Adm123!")
    admToken, _ := login(t, r, admName, "Adm123!")

    orderID := seedOrder(t, r, admToken)
    _, _, taskID := seedPlanLayoutTask(t, r, admToken, orderID)

    // Admin can list participants (restricted to admin/manager)
    w, _ := doJSONAuth(r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d/participants", taskID), "", admToken)
    if w.Code != http.StatusOK { t.Fatalf("admin list participants code=%d body=%s", w.Code, w.Body.String()) }
}