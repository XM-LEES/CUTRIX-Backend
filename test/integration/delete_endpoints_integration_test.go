package integration

import (
    "context"
    "fmt"
    "net/http"
    "testing"
    "time"
)

import (
    "cutrix-backend/internal/models"
)

func Test_Delete_Layout_Cascade(t *testing.T) {
    conn := openDBAndMigrate(t)
    t.Cleanup(func(){ conn.Close() })
    r := buildRouter(conn)

    // Create chain: order -> plan -> layout -> task -> log
    orderNumber := fmt.Sprintf("ORD-IT-%d", time.Now().UnixNano())
    w1, _ := doJSON(r, http.MethodPost, "/api/v1/orders", fmt.Sprintf(`{"order_number":"%s","style_number":"STY-001"}`, orderNumber))
    if w1.Code != http.StatusCreated { t.Fatalf("create order code=%d body=%s", w1.Code, w1.Body.String()) }
    var order models.ProductionOrder
    decodeJSON(t, w1, &order)

    w2, _ := doJSON(r, http.MethodPost, "/api/v1/plans", fmt.Sprintf(`{"order_id":%d,"plan_name":"Plan-Del"}`, order.OrderID))
    if w2.Code != http.StatusCreated { t.Fatalf("create plan code=%d body=%s", w2.Code, w2.Body.String()) }
    var plan models.ProductionPlan
    decodeJSON(t, w2, &plan)

    w3, _ := doJSON(r, http.MethodPost, "/api/v1/layouts", fmt.Sprintf(`{"plan_id":%d,"layout_name":"Layout-Del"}`, plan.PlanID))
    if w3.Code != http.StatusCreated { t.Fatalf("create layout code=%d body=%s", w3.Code, w3.Body.String()) }
    var layout models.CuttingLayout
    decodeJSON(t, w3, &layout)

    w4, _ := doJSON(r, http.MethodPost, "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Blue","planned_layers":2}`, layout.LayoutID))
    if w4.Code != http.StatusCreated { t.Fatalf("create task code=%d body=%s", w4.Code, w4.Body.String()) }
    var task models.ProductionTask
    decodeJSON(t, w4, &task)

    w5, _ := doJSON(r, http.MethodPost, "/api/v1/logs", fmt.Sprintf(`{"task_id":%d,"worker_name":"Temp","layers_completed":1}`, task.TaskID))
    if w5.Code != http.StatusCreated { t.Fatalf("create log code=%d body=%s", w5.Code, w5.Body.String()) }
    var log models.ProductionLog
    decodeJSON(t, w5, &log)

    // Delete layout and assert cascades for tasks and logs
    wd, _ := doJSON(r, http.MethodDelete, fmt.Sprintf("/api/v1/layouts/%d", layout.LayoutID), "")
    if wd.Code != http.StatusNoContent { t.Fatalf("delete layout code=%d body=%s", wd.Code, wd.Body.String()) }

    var exists int
    if err := conn.QueryRowContext(context.Background(), `SELECT 1 FROM production.cutting_layouts WHERE layout_id = $1`, layout.LayoutID).Scan(&exists); err == nil {
        t.Fatalf("layout still exists after deletion")
    }
    if err := conn.QueryRowContext(context.Background(), `SELECT 1 FROM production.tasks WHERE task_id = $1`, task.TaskID).Scan(&exists); err == nil {
        t.Fatalf("task still exists after layout deletion")
    }
    if err := conn.QueryRowContext(context.Background(), `SELECT 1 FROM production.logs WHERE log_id = $1`, log.LogID).Scan(&exists); err == nil {
        t.Fatalf("log still exists after layout deletion")
    }
}

func Test_Delete_Plan_Cascade(t *testing.T) {
    conn := openDBAndMigrate(t)
    t.Cleanup(func(){ conn.Close() })
    r := buildRouter(conn)

    // Create chain: order -> plan -> layout -> task -> log
    orderNumber := fmt.Sprintf("ORD-IT-%d", time.Now().UnixNano())
    w1, _ := doJSON(r, http.MethodPost, "/api/v1/orders", fmt.Sprintf(`{"order_number":"%s","style_number":"STY-002"}`, orderNumber))
    if w1.Code != http.StatusCreated { t.Fatalf("create order code=%d body=%s", w1.Code, w1.Body.String()) }
    var order models.ProductionOrder
    decodeJSON(t, w1, &order)

    w2, _ := doJSON(r, http.MethodPost, "/api/v1/plans", fmt.Sprintf(`{"order_id":%d,"plan_name":"Plan-Del2"}`, order.OrderID))
    if w2.Code != http.StatusCreated { t.Fatalf("create plan code=%d body=%s", w2.Code, w2.Body.String()) }
    var plan models.ProductionPlan
    decodeJSON(t, w2, &plan)

    w3, _ := doJSON(r, http.MethodPost, "/api/v1/layouts", fmt.Sprintf(`{"plan_id":%d,"layout_name":"Layout-Del2"}`, plan.PlanID))
    if w3.Code != http.StatusCreated { t.Fatalf("create layout code=%d body=%s", w3.Code, w3.Body.String()) }
    var layout models.CuttingLayout
    decodeJSON(t, w3, &layout)

    w4, _ := doJSON(r, http.MethodPost, "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Green","planned_layers":1}`, layout.LayoutID))
    if w4.Code != http.StatusCreated { t.Fatalf("create task code=%d body=%s", w4.Code, w4.Body.String()) }
    var task models.ProductionTask
    decodeJSON(t, w4, &task)

    w5, _ := doJSON(r, http.MethodPost, "/api/v1/logs", fmt.Sprintf(`{"task_id":%d,"worker_name":"Temp","layers_completed":1}`, task.TaskID))
    if w5.Code != http.StatusCreated { t.Fatalf("create log code=%d body=%s", w5.Code, w5.Body.String()) }

    // Delete plan and assert cascades for layouts, tasks and logs
    wd, _ := doJSON(r, http.MethodDelete, fmt.Sprintf("/api/v1/plans/%d", plan.PlanID), "")
    if wd.Code != http.StatusNoContent { t.Fatalf("delete plan code=%d body=%s", wd.Code, wd.Body.String()) }

    var exists int
    if err := conn.QueryRowContext(context.Background(), `SELECT 1 FROM production.plans WHERE plan_id = $1`, plan.PlanID).Scan(&exists); err == nil {
        t.Fatalf("plan still exists after deletion")
    }
    if err := conn.QueryRowContext(context.Background(), `SELECT 1 FROM production.cutting_layouts WHERE layout_id = $1`, layout.LayoutID).Scan(&exists); err == nil {
        t.Fatalf("layout still exists after plan deletion")
    }
    if err := conn.QueryRowContext(context.Background(), `SELECT 1 FROM production.tasks WHERE task_id = $1`, task.TaskID).Scan(&exists); err == nil {
        t.Fatalf("task still exists after plan deletion")
    }
    // Any logs for the task should also be gone
    if err := conn.QueryRowContext(context.Background(), `SELECT 1 FROM production.logs WHERE task_id = $1`, task.TaskID).Scan(&exists); err == nil {
        t.Fatalf("logs still exist after plan deletion")
    }
}