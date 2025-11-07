package integration

import (
    "fmt"
    "net/http"
    "testing"
    "time"
)

func TestLogs_Negatives_Create_Void_ListParticipants(t *testing.T) {
    conn := openDBAndMigrate(t)
    defer conn.Close()
    r := buildRouter(conn)

    // First prepare a valid order -> plan -> layout -> task chain
    now := time.Now().UTC()
    orderNumber := fmt.Sprintf("ORD-LOGSNEG-%d", now.UnixNano())

    // Create order, with colors that tasks may reference
    orderBody := fmt.Sprintf(`{
        "order_number": "%s",
        "style_number": "STYLE-LOGSNEG",
        "customer_name": "ACME",
        "order_start_date": "%s",
        "items": [
            {"color":"Red","size":"M","quantity":10},
            {"color":"Blue","size":"L","quantity":5}
        ]
    }`, orderNumber, now.Format(time.RFC3339))
    w, _ := doJSONAuth(r, "POST", "/api/v1/orders", orderBody, "")
    if w.Code != http.StatusCreated { t.Fatalf("create order for logs: want 201 got %d: %s", w.Code, w.Body.String()) }
    var createdOrder struct{ OrderID int `json:"order_id"` }
    decodeJSON(t, w, &createdOrder)

    // Create plan
    planBody := fmt.Sprintf(`{"order_id": %d, "plan_name": "Plan-LogsNEG"}`, createdOrder.OrderID)
    w, _ = doJSONAuth(r, "POST", "/api/v1/plans", planBody, "")
    if w.Code != http.StatusCreated { t.Fatalf("create plan: want 201 got %d: %s", w.Code, w.Body.String()) }
    var createdPlan struct{ PlanID int `json:"plan_id"` }
    decodeJSON(t, w, &createdPlan)

    // Create layout
    layoutBody := fmt.Sprintf(`{"plan_id": %d, "layout_name": "Layout-LogsNEG"}`, createdPlan.PlanID)
    w, _ = doJSONAuth(r, "POST", "/api/v1/layouts", layoutBody, "")
    if w.Code != http.StatusCreated { t.Fatalf("create layout: want 201 got %d: %s", w.Code, w.Body.String()) }
    var createdLayout struct{ LayoutID int `json:"layout_id"` }
    decodeJSON(t, w, &createdLayout)

    // Create task (color must exist in order items)
    taskBody := fmt.Sprintf(`{"layout_id": %d, "color": "Red", "planned_layers": 2}`, createdLayout.LayoutID)
    w, _ = doJSONAuth(r, "POST", "/api/v1/tasks", taskBody, "")
    if w.Code != http.StatusCreated { t.Fatalf("create task: want 201 got %d: %s", w.Code, w.Body.String()) }
    var createdTask struct{ TaskID int `json:"task_id"` }
    decodeJSON(t, w, &createdTask)

    // Publish plan so tasks move to in_progress (logs require in_progress tasks)
    w, _ = doJSONAuth(r, "POST", fmt.Sprintf("/api/v1/plans/%d/publish", createdPlan.PlanID), "", "")
    if w.Code != http.StatusNoContent { t.Fatalf("publish plan: want 204 got %d: %s", w.Code, w.Body.String()) }

    // ---- Logs create negatives ----
    // invalid JSON -> 400
    w, _ = doJSONAuth(r, "POST", "/api/v1/logs", "{", "")
    if w.Code != http.StatusBadRequest { t.Fatalf("logs create invalid json: want 400 got %d: %s", w.Code, w.Body.String()) }

    // task_id == 0 -> 400 (service validation)
    w, _ = doJSONAuth(r, "POST", "/api/v1/logs", `{"task_id":0,"worker_id":1,"layers_completed":1}`, "")
    if w.Code != http.StatusBadRequest { t.Fatalf("logs create task_id=0: want 400 got %d: %s", w.Code, w.Body.String()) }

    // layers_completed <= 0 -> 400 (service validation)
    w, _ = doJSONAuth(r, "POST", "/api/v1/logs", fmt.Sprintf(`{"task_id":%d,"worker_id":1,"layers_completed":0}`, createdTask.TaskID), "")
    if w.Code != http.StatusBadRequest { t.Fatalf("logs create layers_completed=0: want 400 got %d: %s", w.Code, w.Body.String()) }

    // nonexistent worker_id -> 500 (FK violation)
    w, _ = doJSONAuth(r, "POST", "/api/v1/logs", fmt.Sprintf(`{"task_id":%d,"worker_id":999999,"layers_completed":1}`, createdTask.TaskID), "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("logs create invalid worker_id: want 500 got %d: %s", w.Code, w.Body.String()) }

    // nonexistent task_id -> 500 (FK violation)
    w, _ = doJSONAuth(r, "POST", "/api/v1/logs", `{"task_id":999999,"worker_id":1,"layers_completed":1}`, "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("logs create invalid task_id: want 500 got %d: %s", w.Code, w.Body.String()) }

    // Create a valid log (for void tests)
    w, _ = doJSONAuth(r, "POST", "/api/v1/logs", fmt.Sprintf(`{"task_id":%d,"worker_id":1,"layers_completed":1,"note":"ok"}`, createdTask.TaskID), "")
    if w.Code != http.StatusCreated { t.Fatalf("create valid log: want 201 got %d: %s", w.Code, w.Body.String()) }
    var createdLog struct{ LogID int `json:"log_id"` }
    decodeJSON(t, w, &createdLog)

    // ---- Logs void negatives ----
    // invalid id parse -> 400
    w, _ = doJSONAuth(r, "PATCH", "/api/v1/logs/abc", `{"voided_by":1,"void_reason":"wrong"}`, "")
    if w.Code != http.StatusBadRequest { t.Fatalf("logs void invalid id parse: want 400 got %d: %s", w.Code, w.Body.String()) }

    // invalid JSON -> 400
    w, _ = doJSONAuth(r, "PATCH", fmt.Sprintf("/api/v1/logs/%d", createdLog.LogID), "{", "")
    if w.Code != http.StatusBadRequest { t.Fatalf("logs void invalid json: want 400 got %d: %s", w.Code, w.Body.String()) }

    // invalid voided_by (FK violation) -> 500
    w, _ = doJSONAuth(r, "PATCH", fmt.Sprintf("/api/v1/logs/%d", createdLog.LogID), `{"voided_by":999999,"void_reason":"wrong"}`, "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("logs void invalid voided_by: want 500 got %d: %s", w.Code, w.Body.String()) }

    // invalid negative id value -> 400 (service validation)
    w, _ = doJSONAuth(r, "PATCH", "/api/v1/logs/-1", `{"voided_by":1}`, "")
    if w.Code != http.StatusBadRequest { t.Fatalf("logs void negative id: want 400 got %d: %s", w.Code, w.Body.String()) }

    // ---- Logs participants negatives ----
    // invalid task id parse -> 400
    w, _ = doJSONAuth(r, "GET", "/api/v1/tasks/abc/participants", "", "")
    if w.Code != http.StatusBadRequest { t.Fatalf("list participants invalid id parse: want 400 got %d: %s", w.Code, w.Body.String()) }
}