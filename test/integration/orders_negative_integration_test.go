package integration

import (
    "fmt"
    "net/http"
    "testing"
    "time"
)

func TestOrders_Negatives_CRUD(t *testing.T) {
    conn := openDBAndMigrate(t)
    defer conn.Close()
    r := buildRouter(conn)

    now := time.Now().UTC()
    orderNumber := fmt.Sprintf("ORD-NEG-%d", now.UnixNano())

    // 1) invalid JSON -> 400
    w, _ := doJSONAuth(r, "POST", "/api/v1/orders", "{", "")
    if w.Code != http.StatusBadRequest { t.Fatalf("invalid json: want 400 got %d: %s", w.Code, w.Body.String()) }

    // 2) missing order_number -> 500
    bodyMissingNumber := fmt.Sprintf(`{
        "style_number": "STYLE-ONE",
        "customer_name": "ACME",
        "order_start_date": "%s",
        "items": [ {"color":"Red","size":"M","quantity":1} ]
    }`, now.Format(time.RFC3339))
    w, _ = doJSONAuth(r, "POST", "/api/v1/orders", bodyMissingNumber, "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("missing order_number: want 500 got %d: %s", w.Code, w.Body.String()) }

    // 3) missing style_number -> 500
    bodyMissingStyle := fmt.Sprintf(`{
        "order_number": "%s",
        "customer_name": "ACME",
        "order_start_date": "%s",
        "items": [ {"color":"Red","size":"M","quantity":1} ]
    }`, orderNumber, now.Format(time.RFC3339))
    w, _ = doJSONAuth(r, "POST", "/api/v1/orders", bodyMissingStyle, "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("missing style_number: want 500 got %d: %s", w.Code, w.Body.String()) }

    // 4) empty items -> 500
    bodyEmptyItems := fmt.Sprintf(`{
        "order_number": "%s",
        "style_number": "STYLE-EMPTY",
        "customer_name": "ACME",
        "order_start_date": "%s",
        "items": []
    }`, orderNumber, now.Format(time.RFC3339))
    w, _ = doJSONAuth(r, "POST", "/api/v1/orders", bodyEmptyItems, "")
    if w.Code != http.StatusInternalServerError { t.Fatalf("empty items: want 500 got %d: %s", w.Code, w.Body.String()) }

    // Create a valid order for subsequent checks
    createValid := fmt.Sprintf(`{
        "order_number": "%s",
        "style_number": "STYLE-OK",
        "customer_name": "ACME",
        "order_start_date": "%s",
        "items": [
            {"color":"Red","size":"M","quantity":10},
            {"color":"Blue","size":"L","quantity":5}
        ]
    }`, orderNumber, now.Format(time.RFC3339))
    w, _ = doJSONAuth(r, "POST", "/api/v1/orders", createValid, "")
    if w.Code != http.StatusCreated { t.Fatalf("create valid: want 201 got %d: %s", w.Code, w.Body.String()) }
    var created struct{ OrderID int `json:"order_id"` }
    decodeJSON(t, w, &created)

    // 5) get invalid id parse -> 400
    w, _ = doJSONAuth(r, "GET", "/api/v1/orders/abc", "", "")
    if w.Code != http.StatusBadRequest { t.Fatalf("get invalid id parse: want 400 got %d: %s", w.Code, w.Body.String()) }

    // 6) get not found -> 404
    w, _ = doJSONAuth(r, "GET", "/api/v1/orders/99999999", "", "")
    if w.Code != http.StatusNotFound { t.Fatalf("get not found: want 404 got %d: %s", w.Code, w.Body.String()) }

    // 7) get full invalid id parse -> 400
    w, _ = doJSONAuth(r, "GET", "/api/v1/orders/abc/full", "", "")
    if w.Code != http.StatusBadRequest { t.Fatalf("get full invalid id parse: want 400 got %d: %s", w.Code, w.Body.String()) }

    // 8) get full not found -> 404
    w, _ = doJSONAuth(r, "GET", "/api/v1/orders/99999999/full", "", "")
    if w.Code != http.StatusNotFound { t.Fatalf("get full not found: want 404 got %d: %s", w.Code, w.Body.String()) }

    // 9) update note invalid id parse -> 400
    w, _ = doJSONAuth(r, "PATCH", "/api/v1/orders/abc/note", `{"note":"xx"}`, "")
    if w.Code != http.StatusBadRequest { t.Fatalf("update note invalid id parse: want 400 got %d: %s", w.Code, w.Body.String()) }

    // 10) update note invalid JSON -> 400
    w, _ = doJSONAuth(r, "PATCH", fmt.Sprintf("/api/v1/orders/%d/note", created.OrderID), "{", "")
    if w.Code != http.StatusBadRequest { t.Fatalf("update note invalid json: want 400 got %d: %s", w.Code, w.Body.String()) }

    // 11) update finish date invalid id parse -> 400
    w, _ = doJSONAuth(r, "PATCH", "/api/v1/orders/abc/finish-date", `{"order_finish_date":"`+now.Format(time.RFC3339)+`"}`, "")
    if w.Code != http.StatusBadRequest { t.Fatalf("update finish date invalid id parse: want 400 got %d: %s", w.Code, w.Body.String()) }

    // 12) update finish date invalid JSON -> 400
    w, _ = doJSONAuth(r, "PATCH", fmt.Sprintf("/api/v1/orders/%d/finish-date", created.OrderID), "{", "")
    if w.Code != http.StatusBadRequest { t.Fatalf("update finish date invalid json: want 400 got %d: %s", w.Code, w.Body.String()) }

    // 13) delete invalid id parse -> 400
    w, _ = doJSONAuth(r, "DELETE", "/api/v1/orders/abc", "", "")
    if w.Code != http.StatusBadRequest { t.Fatalf("delete invalid id parse: want 400 got %d: %s", w.Code, w.Body.String()) }

    // 14) delete not found -> 404
    w, _ = doJSONAuth(r, "DELETE", "/api/v1/orders/99999999", "", "")
    if w.Code != http.StatusNotFound { t.Fatalf("delete not found: want 404 got %d: %s", w.Code, w.Body.String()) }
}