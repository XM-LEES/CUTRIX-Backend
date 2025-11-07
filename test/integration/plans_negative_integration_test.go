package integration

import (
    "fmt"
    "net/http"
    "testing"
    "time"
)

func TestPlans_Negatives_Create_And_State(t *testing.T) {
    conn := openDBAndMigrate(t)
    defer conn.Close()
    r := buildRouter(conn)

    now := time.Now().UTC()
    orderNumber := fmt.Sprintf("ORD-%d", now.UnixNano())

    // Create an order first
    createOrder := fmt.Sprintf(`{
        "order_number": "%s",
        "style_number": "STYLE-PNEG-001",
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

    // 1) invalid JSON -> 400
    w, _ = doJSONAuth(r, "POST", "/api/v1/plans", "{", "")
    if w.Code != http.StatusBadRequest { t.Fatalf("invalid json: want 400 got %d: %s", w.Code, w.Body.String()) }

    // 2) missing plan_name -> 500 (service validation error mapped to internal)
    w, _ = doJSONAuth(r, "POST", "/api/v1/plans", fmt.Sprintf(`{"order_id":%d}`, createdOrder.OrderID), "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("missing plan_name: want 500 got %d: %s", w.Code, w.Body.String()) }

    // 3) invalid order_id -> 500
    w, _ = doJSONAuth(r, "POST", "/api/v1/plans", `{"plan_name":"Plan-Neg","order_id":0}`, "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("invalid order_id: want 500 got %d: %s", w.Code, w.Body.String()) }

    // Create a valid plan
    w, _ = doJSONAuth(r, "POST", "/api/v1/plans", fmt.Sprintf(`{"plan_name":"Plan-Valid","order_id":%d}`, createdOrder.OrderID), "")
    if w.Code != http.StatusCreated { t.Fatalf("create plan: want 201 got %d: %s", w.Code, w.Body.String()) }
    var createdPlan struct{ PlanID int `json:"plan_id"` }
    decodeJSON(t, w, &createdPlan)

    // 4) publish without tasks -> 500 (trigger guard)
    w, _ = doJSONAuth(r, "POST", fmt.Sprintf("/api/v1/plans/%d/publish", createdPlan.PlanID), "", "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("publish without tasks: want 500 got %d: %s", w.Code, w.Body.String()) }

    // 5) freeze while pending -> 500 (trigger guard)
    w, _ = doJSONAuth(r, "POST", fmt.Sprintf("/api/v1/plans/%d/freeze", createdPlan.PlanID), "", "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("freeze pending: want 500 got %d: %s", w.Code, w.Body.String()) }

    // 6) get not found -> 404
    w, _ = doJSONAuth(r, "GET", "/api/v1/plans/99999999", "", "")
    if w.Code != http.StatusNotFound { t.Fatalf("get not found: want 404 got %d: %s", w.Code, w.Body.String()) }

    // 7) delete not found -> 404
    w, _ = doJSONAuth(r, "DELETE", "/api/v1/plans/99999999", "", "")
    if w.Code != http.StatusNotFound { t.Fatalf("delete not found: want 404 got %d: %s", w.Code, w.Body.String()) }

    // 8) update note invalid id -> 400
    w, _ = doJSONAuth(r, "PATCH", "/api/v1/plans/abc/note", `{"note":"x"}`, "")
    if w.Code != http.StatusBadRequest { t.Fatalf("update note invalid id: want 400 got %d: %s", w.Code, w.Body.String()) }
}