package integration

import (
    "context"
    "database/sql"
    "fmt"
    "net/http"
    "testing"
    "time"
)

import (
    "cutrix-backend/internal/models"
)

func Test_FullFlow_Triggers_And_Cascade(t *testing.T) {
    conn := openDBAndMigrate(t)
    t.Cleanup(func(){ conn.Close() })
    r := buildRouter(conn)

    // 1) Create Order
    orderNumber := fmt.Sprintf("ORD-IT-%d", time.Now().UnixNano())
    w1, _ := doJSON(r, http.MethodPost, "/api/v1/orders", fmt.Sprintf(`{"order_number":"%s","style_number":"STY-001","customer_name":"ACME"}`, orderNumber))
    if w1.Code != http.StatusCreated { t.Fatalf("create order code=%d body=%s", w1.Code, w1.Body.String()) }
    var order models.ProductionOrder
    decodeJSON(t, w1, &order)
    if order.OrderID == 0 { t.Fatalf("expected non-zero order_id") }

    // 2) Create Plan under Order
    w2, _ := doJSON(r, http.MethodPost, "/api/v1/plans", fmt.Sprintf(`{"order_id":%d,"plan_name":"Plan-A"}`, order.OrderID))
    if w2.Code != http.StatusCreated { t.Fatalf("create plan code=%d body=%s", w2.Code, w2.Body.String()) }
    var plan models.ProductionPlan
    decodeJSON(t, w2, &plan)

    // 3) Create Layout under Plan
    w3, _ := doJSON(r, http.MethodPost, "/api/v1/layouts", fmt.Sprintf(`{"plan_id":%d,"layout_name":"Layout-1"}`, plan.PlanID))
    if w3.Code != http.StatusCreated { t.Fatalf("create layout code=%d body=%s", w3.Code, w3.Body.String()) }
    var layout models.CuttingLayout
    decodeJSON(t, w3, &layout)

    // 4) Create Task under Layout (planned_layers = 3)
    w4, _ := doJSON(r, http.MethodPost, "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Red","planned_layers":3}`, layout.LayoutID))
    if w4.Code != http.StatusCreated { t.Fatalf("create task code=%d body=%s", w4.Code, w4.Body.String()) }
    var task models.ProductionTask
    decodeJSON(t, w4, &task)

    // Lookup workers
    var wZhangSan, wAdmin int
    if err := conn.QueryRowContext(context.Background(), `SELECT worker_id FROM public.Workers WHERE name = '张三'`).Scan(&wZhangSan); err != nil {
        t.Fatalf("lookup 张三: %v", err)
    }
    if err := conn.QueryRowContext(context.Background(), `SELECT worker_id FROM public.Workers WHERE name = 'admin'`).Scan(&wAdmin); err != nil {
        t.Fatalf("lookup admin: %v", err)
    }

    // 5) Submit Log1 (layers_completed=2, worker_id=张三), expect task in_progress and worker_name auto-filled
    w5, _ := doJSON(r, http.MethodPost, "/api/v1/logs", fmt.Sprintf(`{"task_id":%d,"worker_id":%d,"layers_completed":2}`, task.TaskID, wZhangSan))
    if w5.Code != http.StatusCreated { t.Fatalf("create log1 code=%d body=%s", w5.Code, w5.Body.String()) }
    var log1 models.ProductionLog
    decodeJSON(t, w5, &log1)
    if log1.LogID == 0 { t.Fatalf("expected non-zero log_id") }

    // Verify task progress via GET
    wtg1, _ := doJSON(r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", task.TaskID), "")
    if wtg1.Code != http.StatusOK { t.Fatalf("get task code=%d body=%s", wtg1.Code, wtg1.Body.String()) }
    var tstate1 models.ProductionTask
    decodeJSON(t, wtg1, &tstate1)
    if tstate1.CompletedLayers != 2 || tstate1.Status != "in_progress" {
        t.Fatalf("task state after log1: completed=%d status=%s", tstate1.CompletedLayers, tstate1.Status)
    }
    // Check worker_name filled in DB
    var workerName1 sql.NullString
    if err := conn.QueryRowContext(context.Background(), `SELECT worker_name FROM production.logs WHERE log_id = $1`, log1.LogID).Scan(&workerName1); err != nil {
        t.Fatalf("query log1 worker_name: %v", err)
    }
    if !workerName1.Valid || workerName1.String == "" { t.Fatalf("worker_name not filled by trigger") }

    // 6) Submit Log2 (layers_completed=1), expect task completed
    w6, _ := doJSON(r, http.MethodPost, "/api/v1/logs", fmt.Sprintf(`{"task_id":%d,"worker_id":%d,"layers_completed":1}`, task.TaskID, wZhangSan))
    if w6.Code != http.StatusCreated { t.Fatalf("create log2 code=%d body=%s", w6.Code, w6.Body.String()) }
    var log2 models.ProductionLog
    decodeJSON(t, w6, &log2)

    wtg2, _ := doJSON(r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", task.TaskID), "")
    if wtg2.Code != http.StatusOK { t.Fatalf("get task2 code=%d body=%s", wtg2.Code, wtg2.Body.String()) }
    var tstate2 models.ProductionTask
    decodeJSON(t, wtg2, &tstate2)
    if tstate2.CompletedLayers != 3 || tstate2.Status != "completed" {
        t.Fatalf("task state after log2: completed=%d status=%s", tstate2.CompletedLayers, tstate2.Status)
    }

    // 7) Void log1 (voided=true, void_reason, voided_by=admin), expect completed_layers decrease and void metadata filled
    wv1, _ := doJSON(r, http.MethodPatch, fmt.Sprintf("/api/v1/logs/%d", log1.LogID), fmt.Sprintf(`{"voided":true,"void_reason":"mistake","voided_by":%d}`, wAdmin))
    if wv1.Code != http.StatusNoContent { t.Fatalf("void log1 code=%d body=%s", wv1.Code, wv1.Body.String()) }

    wtg3, _ := doJSON(r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", task.TaskID), "")
    if wtg3.Code != http.StatusOK { t.Fatalf("get task3 code=%d body=%s", wtg3.Code, wtg3.Body.String()) }
    var tstate3 models.ProductionTask
    decodeJSON(t, wtg3, &tstate3)
    if tstate3.CompletedLayers != 1 || tstate3.Status != "in_progress" {
        t.Fatalf("task state after void log1: completed=%d status=%s", tstate3.CompletedLayers, tstate3.Status)
    }

    var voidedAt sql.NullTime
    var voidedByName sql.NullString
    if err := conn.QueryRowContext(context.Background(), `SELECT voided_at, voided_by_name FROM production.logs WHERE log_id = $1`, log1.LogID).Scan(&voidedAt, &voidedByName); err != nil {
        t.Fatalf("query log1 void meta: %v", err)
    }
    if !voidedAt.Valid || !voidedByName.Valid || voidedByName.String == "" {
        t.Fatalf("void metadata not filled by trigger")
    }

    // 8) Unvoid log1 (voided=false), expect completed_layers back to 3 and status completed
    wuv1, _ := doJSON(r, http.MethodPatch, fmt.Sprintf("/api/v1/logs/%d", log1.LogID), `{"voided":false}`)
    if wuv1.Code != http.StatusNoContent { t.Fatalf("unvoid log1 code=%d body=%s", wuv1.Code, wuv1.Body.String()) }

    wtg4, _ := doJSON(r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d", task.TaskID), "")
    if wtg4.Code != http.StatusOK { t.Fatalf("get task4 code=%d body=%s", wtg4.Code, wtg4.Body.String()) }
    var tstate4 models.ProductionTask
    decodeJSON(t, wtg4, &tstate4)
    if tstate4.CompletedLayers != 3 || tstate4.Status != "completed" {
        t.Fatalf("task state after unvoid log1: completed=%d status=%s", tstate4.CompletedLayers, tstate4.Status)
    }

    // 9) Participants should include 张三 (non-voided)
    wp1, _ := doJSON(r, http.MethodGet, fmt.Sprintf("/api/v1/tasks/%d/participants", task.TaskID), "")
    if wp1.Code != http.StatusOK { t.Fatalf("participants code=%d body=%s", wp1.Code, wp1.Body.String()) }
    var participants []string
    decodeJSON(t, wp1, &participants)
    if len(participants) == 0 || participants[0] == "" { t.Fatalf("participants empty: %v", participants) }

    // 10) Delete the order; expect cascade deletes for plan/layout/task/log
    wd, _ := doJSON(r, http.MethodDelete, fmt.Sprintf("/api/v1/orders/%d", order.OrderID), "")
    if wd.Code != http.StatusNoContent { t.Fatalf("delete order code=%d body=%s", wd.Code, wd.Body.String()) }

    // Verify cascade - no rows for created IDs
    var exists int
    if err := conn.QueryRowContext(context.Background(), `SELECT 1 FROM production.plans WHERE plan_id = $1`, plan.PlanID).Scan(&exists); err == nil {
        t.Fatalf("plan still exists after order deletion")
    }
    if err := conn.QueryRowContext(context.Background(), `SELECT 1 FROM production.cutting_layouts WHERE layout_id = $1`, layout.LayoutID).Scan(&exists); err == nil {
        t.Fatalf("layout still exists after order deletion")
    }
    if err := conn.QueryRowContext(context.Background(), `SELECT 1 FROM production.tasks WHERE task_id = $1`, task.TaskID).Scan(&exists); err == nil {
        t.Fatalf("task still exists after order deletion")
    }
    if err := conn.QueryRowContext(context.Background(), `SELECT 1 FROM production.logs WHERE log_id = $1`, log1.LogID).Scan(&exists); err == nil {
        t.Fatalf("log1 still exists after order deletion")
    }
}