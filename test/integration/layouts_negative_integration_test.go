package integration

import (
    "fmt"
    "net/http"
    "testing"
    "time"
)

func TestLayouts_Negatives_Create_Update_Delete(t *testing.T) {
    conn := openDBAndMigrate(t)
    defer conn.Close()
    r := buildRouter(conn)

    now := time.Now().UTC()
    orderNumber := fmt.Sprintf("ORD-%d", now.UnixNano())

    // Create order
    createOrder := fmt.Sprintf(`{
        "order_number": "%s",
        "style_number": "STYLE-LNEG-001",
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
    w, _ = doJSONAuth(r, "POST", "/api/v1/plans", fmt.Sprintf(`{"plan_name":"Plan-LNEG","order_id":%d}`, createdOrder.OrderID), "")
    if w.Code != http.StatusCreated { t.Fatalf("create plan want 201 got %d: %s", w.Code, w.Body.String()) }
    var createdPlan struct{ PlanID int `json:"plan_id"` }
    decodeJSON(t, w, &createdPlan)

    // 1) invalid JSON -> 400
    w, _ = doJSONAuth(r, "POST", "/api/v1/layouts", "{", "")
    if w.Code != http.StatusBadRequest { t.Fatalf("invalid json: want 400 got %d: %s", w.Code, w.Body.String()) }

    // 2) missing name -> 500
    w, _ = doJSONAuth(r, "POST", "/api/v1/layouts", fmt.Sprintf(`{"plan_id":%d}`, createdPlan.PlanID), "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("missing layout_name: want 500 got %d: %s", w.Code, w.Body.String()) }

    // 3) invalid plan_id -> 500
    w, _ = doJSONAuth(r, "POST", "/api/v1/layouts", `{"layout_name":"L1","plan_id":0}`, "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("invalid plan_id: want 500 got %d: %s", w.Code, w.Body.String()) }

    // Create a valid layout
    w, _ = doJSONAuth(r, "POST", "/api/v1/layouts", fmt.Sprintf(`{"layout_name":"L1","plan_id":%d}`, createdPlan.PlanID), "")
    if w.Code != http.StatusCreated { t.Fatalf("create layout want 201 got %d: %s", w.Code, w.Body.String()) }
    var createdLayout struct{ LayoutID int `json:"layout_id"` }
    decodeJSON(t, w, &createdLayout)

    // 4) update name with empty -> 500
    w, _ = doJSONAuth(r, "PATCH", fmt.Sprintf("/api/v1/layouts/%d/name", createdLayout.LayoutID), `{"name":""}`, "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("update name empty: want 500 got %d: %s", w.Code, w.Body.String()) }

    // 5) update name invalid id -> 400
    w, _ = doJSONAuth(r, "PATCH", "/api/v1/layouts/abc/name", `{"name":"X"}`, "")
    if w.Code != http.StatusBadRequest { t.Fatalf("update name invalid id: want 400 got %d: %s", w.Code, w.Body.String()) }

    // 6) get not found -> 404
    w, _ = doJSONAuth(r, "GET", "/api/v1/layouts/99999999", "", "")
    if w.Code != http.StatusNotFound { t.Fatalf("get not found: want 404 got %d: %s", w.Code, w.Body.String()) }

    // 7) delete not found -> 404
    w, _ = doJSONAuth(r, "DELETE", "/api/v1/layouts/99999999", "", "")
    if w.Code != http.StatusNotFound { t.Fatalf("delete not found: want 404 got %d: %s", w.Code, w.Body.String()) }

    // 8) publish plan and then try update name -> 500
    w, _ = doJSONAuth(r, "POST", fmt.Sprintf("/api/v1/plans/%d/publish", createdPlan.PlanID), "", "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("publish without tasks should fail: want 500 got %d: %s", w.Code, w.Body.String()) }

    // 为了强制布局后置负例，先创建一个任务，再发布计划
    w, _ = doJSONAuth(r, "POST", "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Red","planned_layers":1}`, createdLayout.LayoutID), "")
    if w.Code != http.StatusCreated { t.Fatalf("create task for publish want 201 got %d: %s", w.Code, w.Body.String()) }

    // 重新发布（现在有任务）
    w, _ = doJSONAuth(r, "POST", fmt.Sprintf("/api/v1/plans/%d/publish", createdPlan.PlanID), "", "")
    if w.Code != http.StatusNoContent { t.Fatalf("publish with task: want 204 got %d: %s", w.Code, w.Body.String()) }

    // 发布后更新布局名称 -> 500
    w, _ = doJSONAuth(r, "PATCH", fmt.Sprintf("/api/v1/layouts/%d/name", createdLayout.LayoutID), `{"name":"Renamed"}`, "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("update name after publish: want 500 got %d: %s", w.Code, w.Body.String()) }

    // 发布后新增布局 -> 500
    w, _ = doJSONAuth(r, "POST", "/api/v1/layouts", fmt.Sprintf(`{"layout_name":"L2","plan_id":%d}`, createdPlan.PlanID), "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("create layout after publish: want 500 got %d: %s", w.Code, w.Body.String()) }
}