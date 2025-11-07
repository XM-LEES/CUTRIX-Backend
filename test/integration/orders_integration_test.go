package integration

import (
    "fmt"
    "net/http"
    "net/url"
    "testing"
    "time"

    "cutrix-backend/internal/models"
)

func TestOrders_CRUD_Queries_Updates(t *testing.T) {
    conn := openDBAndMigrate(t)
    defer conn.Close()
    r := buildRouter(conn)

    now := time.Now().UTC()
    nowDate := now.Truncate(time.Second)
    orderNumber := fmt.Sprintf("ORD-%d", now.UnixNano())

    createBody := fmt.Sprintf(`{
        "order_number": "%s",
        "style_number": "STYLE-100",
        "customer_name": "ACME",
        "order_start_date": "%s",
        "note": "first order",
        "items": [
            {"color":"Red","size":"M","quantity":10},
            {"color":"Blue","size":"L","quantity":5}
        ]
    }`, orderNumber, nowDate.Format(time.RFC3339))

    // Create with items
    w, _ := doJSONAuth(r, "POST", "/api/v1/orders", createBody, "")
    if w.Code != http.StatusCreated {
        t.Fatalf("create: want 201 got %d: %s", w.Code, w.Body.String())
    }
    var created models.ProductionOrder
    decodeJSON(t, w, &created)
    if created.OrderID == 0 { t.Fatalf("created OrderID is 0") }
    if created.OrderNumber != orderNumber { t.Fatalf("created order_number mismatch: %s", created.OrderNumber) }

    id := created.OrderID

    // List all orders and ensure created order exists
    w, _ = doJSONAuth(r, "GET", "/api/v1/orders", "", "")
    if w.Code != http.StatusOK {
        t.Fatalf("list: want 200 got %d: %s", w.Code, w.Body.String())
    }
    var list []models.ProductionOrder
    decodeJSON(t, w, &list)
    found := false
    for _, o := range list {
        if o.OrderID == id { found = true; break }
    }
    if !found { t.Fatalf("list: created order not found in list") }

    // Get by ID
    w, _ = doJSONAuth(r, "GET", fmt.Sprintf("/api/v1/orders/%d", id), "", "")
    if w.Code != http.StatusOK {
        t.Fatalf("get: want 200 got %d: %s", w.Code, w.Body.String())
    }
    var got models.ProductionOrder
    decodeJSON(t, w, &got)
    if got.OrderID != id { t.Fatalf("get: id mismatch") }

    // Get by number
    w, _ = doJSONAuth(r, "GET", fmt.Sprintf("/api/v1/orders/by-number/%s", url.PathEscape(orderNumber)), "", "")
    if w.Code != http.StatusOK {
        t.Fatalf("get by number: want 200 got %d: %s", w.Code, w.Body.String())
    }
    var byNum models.ProductionOrder
    decodeJSON(t, w, &byNum)
    if byNum.OrderID != id { t.Fatalf("get by number: id mismatch") }

    // Get full
    w, _ = doJSONAuth(r, "GET", fmt.Sprintf("/api/v1/orders/%d/full", id), "", "")
    if w.Code != http.StatusOK {
        t.Fatalf("get full: want 200 got %d: %s", w.Code, w.Body.String())
    }
    var full struct {
        Order models.ProductionOrder `json:"order"`
        Items []models.OrderItem `json:"items"`
    }
    decodeJSON(t, w, &full)
    if full.Order.OrderID != id { t.Fatalf("get full: order id mismatch") }
    if len(full.Items) != 2 { t.Fatalf("get full: want 2 items got %d", len(full.Items)) }

    // Update note
    w, _ = doJSONAuth(r, "PATCH", fmt.Sprintf("/api/v1/orders/%d/note", id), `{"note":"updated note"}`, "")
    if w.Code != http.StatusNoContent {
        t.Fatalf("update note: want 204 got %d: %s", w.Code, w.Body.String())
    }

    // Update finish date
    finish := nowDate.Add(24 * time.Hour)
    w, _ = doJSONAuth(r, "PATCH", fmt.Sprintf("/api/v1/orders/%d/finish-date", id), fmt.Sprintf(`{"order_finish_date":"%s"}`, finish.Format(time.RFC3339)), "")
    if w.Code != http.StatusNoContent {
        t.Fatalf("update finish date: want 204 got %d: %s", w.Code, w.Body.String())
    }

    // Verify updates
    w, _ = doJSONAuth(r, "GET", fmt.Sprintf("/api/v1/orders/%d", id), "", "")
    if w.Code != http.StatusOK {
        t.Fatalf("get after updates: want 200 got %d: %s", w.Code, w.Body.String())
    }
    var updated models.ProductionOrder
    decodeJSON(t, w, &updated)
    if updated.Note == nil || *updated.Note != "updated note" {
        t.Fatalf("note not updated: %+v", updated.Note)
    }
    if updated.OrderFinishDate == nil || !updated.OrderFinishDate.Equal(finish) {
        t.Fatalf("finish date not updated: %+v", updated.OrderFinishDate)
    }

    // Delete
    w, _ = doJSONAuth(r, "DELETE", fmt.Sprintf("/api/v1/orders/%d", id), "", "")
    if w.Code != http.StatusNoContent {
        t.Fatalf("delete: want 204 got %d: %s", w.Code, w.Body.String())
    }

    // Ensure 404 after deletion
    w, _ = doJSONAuth(r, "GET", fmt.Sprintf("/api/v1/orders/%d", id), "", "")
    if w.Code != http.StatusNotFound {
        t.Fatalf("get after delete: want 404 got %d: %s", w.Code, w.Body.String())
    }
    w, _ = doJSONAuth(r, "GET", fmt.Sprintf("/api/v1/orders/%d/full", id), "", "")
    if w.Code != http.StatusNotFound {
        t.Fatalf("get full after delete: want 404 got %d: %s", w.Code, w.Body.String())
    }
    w, _ = doJSONAuth(r, "GET", fmt.Sprintf("/api/v1/orders/by-number/%s", url.PathEscape(orderNumber)), "", "")
    if w.Code != http.StatusNotFound {
        t.Fatalf("get by number after delete: want 404 got %d: %s", w.Code, w.Body.String())
    }
}