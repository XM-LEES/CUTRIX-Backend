package integration

import (
    "fmt"
    "net/http"
    "testing"
    "time"
)

func TestTasks_Negatives_Create_Delete_Get(t *testing.T) {
    conn := openDBAndMigrate(t)
    defer conn.Close()
    r := buildRouter(conn)

    now := time.Now().UTC()
    orderNumber := fmt.Sprintf("ORD-%d", now.UnixNano())

    // Create order
    createOrder := fmt.Sprintf(`{
        "order_number": "%s",
        "style_number": "STYLE-TNEG-001",
        "customer_name": "ACME",
        "order_start_date": "%s",
        "items": [
            {"color":"Red","size":"M","quantity":1}
        ]
    }`, orderNumber, now.Format(time.RFC3339))
    w, _ := doJSONAuth(r, "POST", "/api/v1/orders", createOrder, "")
    if w.Code != http.StatusCreated { t.Fatalf("create order want 201 got %d: %s", w.Code, w.Body.String()) }
    var createdOrder struct{ OrderID int `json:"order_id"` }
    decodeJSON(t, w, &createdOrder)

    // Create plan
    w, _ = doJSONAuth(r, "POST", "/api/v1/plans", fmt.Sprintf(`{"plan_name":"Plan-TNEG","order_id":%d}`, createdOrder.OrderID), "")
    if w.Code != http.StatusCreated { t.Fatalf("create plan want 201 got %d: %s", w.Code, w.Body.String()) }
    var createdPlan struct{ PlanID int `json:"plan_id"` }
    decodeJSON(t, w, &createdPlan)

    // Create layout
    w, _ = doJSONAuth(r, "POST", "/api/v1/layouts", fmt.Sprintf(`{"layout_name":"L1","plan_id":%d}`, createdPlan.PlanID), "")
    if w.Code != http.StatusCreated { t.Fatalf("create layout want 201 got %d: %s", w.Code, w.Body.String()) }
    var createdLayout struct{ LayoutID int `json:"layout_id"` }
    decodeJSON(t, w, &createdLayout)

    // 1) invalid JSON -> 400
    w, _ = doJSONAuth(r, "POST", "/api/v1/tasks", "{", "")
    if w.Code != http.StatusBadRequest { t.Fatalf("invalid json: want 400 got %d: %s", w.Code, w.Body.String()) }

    // 2) planned_layers <= 0 -> 500
    w, _ = doJSONAuth(r, "POST", "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Red","planned_layers":0}`, createdLayout.LayoutID), "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("planned_layers<=0: want 500 got %d: %s", w.Code, w.Body.String()) }

    // 3) color not in order items -> 500 (trigger)
    w, _ = doJSONAuth(r, "POST", "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Green","planned_layers":1}`, createdLayout.LayoutID), "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("color not in order items: want 500 got %d: %s", w.Code, w.Body.String()) }

    // Create a valid task
    w, _ = doJSONAuth(r, "POST", "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Red","planned_layers":1}`, createdLayout.LayoutID), "")
    if w.Code != http.StatusCreated { t.Fatalf("create task want 201 got %d: %s", w.Code, w.Body.String()) }
    var createdTask struct{ TaskID int `json:"task_id"` }
    decodeJSON(t, w, &createdTask)

    // 4) get not found -> 404
    w, _ = doJSONAuth(r, "GET", "/api/v1/tasks/99999999", "", "")
    if w.Code != http.StatusNotFound { t.Fatalf("get not found: want 404 got %d: %s", w.Code, w.Body.String()) }

    // 5) delete not found -> 404
    w, _ = doJSONAuth(r, "DELETE", "/api/v1/tasks/99999999", "", "")
    if w.Code != http.StatusNotFound { t.Fatalf("delete not found: want 404 got %d: %s", w.Code, w.Body.String()) }

    // 6) publish plan then create task -> 500
    w, _ = doJSONAuth(r, "POST", fmt.Sprintf("/api/v1/plans/%d/publish", createdPlan.PlanID), "", "")
    if w.Code != http.StatusNoContent { t.Fatalf("publish: want 204 got %d: %s", w.Code, w.Body.String()) }

    w, _ = doJSONAuth(r, "POST", "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Red","planned_layers":1}`, createdLayout.LayoutID), "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("create after publish: want 500 got %d: %s", w.Code, w.Body.String()) }

    // 7) delete after publish -> 500
    w, _ = doJSONAuth(r, "DELETE", fmt.Sprintf("/api/v1/tasks/%d", createdTask.TaskID), "", "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("delete after publish: want 500 got %d: %s", w.Code, w.Body.String()) }
}