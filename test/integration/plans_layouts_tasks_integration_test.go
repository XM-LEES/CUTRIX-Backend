package integration

import (
    "fmt"
    "net/http"
    "testing"
    "time"

    "cutrix-backend/internal/models"
)

func TestPlansLayoutsTasks_EndToEnd(t *testing.T) {
    conn := openDBAndMigrate(t)
    defer conn.Close()
    r := buildRouter(conn)

    now := time.Now().UTC()
    orderNumber := fmt.Sprintf("ORD-%d", now.UnixNano())

    // 1) 创建订单（含两个颜色的条目）
    createOrderBody := fmt.Sprintf(`{
        "order_number": "%s",
        "style_number": "STYLE-PLT-001",
        "customer_name": "ACME",
        "order_start_date": "%s",
        "items": [
            {"color":"Red","size":"M","quantity":10},
            {"color":"Blue","size":"L","quantity":5}
        ]
    }`, orderNumber, now.Format(time.RFC3339))
    w, _ := doJSONAuth(r, "POST", "/api/v1/orders", createOrderBody, "")
    if w.Code != http.StatusCreated {
        t.Fatalf("create order: want 201 got %d: %s", w.Code, w.Body.String())
    }
    var order models.ProductionOrder
    decodeJSON(t, w, &order)

    // 2) 创建计划
    w, _ = doJSONAuth(r, "POST", "/api/v1/plans", fmt.Sprintf(`{"plan_name":"Plan-A","order_id":%d}`, order.OrderID), "")
    if w.Code != http.StatusCreated {
        t.Fatalf("create plan: want 201 got %d: %s", w.Code, w.Body.String())
    }
    var plan models.ProductionPlan
    decodeJSON(t, w, &plan)
    if plan.Status != "pending" {
        t.Fatalf("plan status should be pending: %s", plan.Status)
    }

    // 3) 按订单列出计划，确保存在
    w, _ = doJSONAuth(r, "GET", fmt.Sprintf("/api/v1/orders/%d/plans", order.OrderID), "", "")
    if w.Code != http.StatusOK {
        t.Fatalf("list plans: want 200 got %d: %s", w.Code, w.Body.String())
    }

    // 4) 创建布局
    w, _ = doJSONAuth(r, "POST", "/api/v1/layouts", fmt.Sprintf(`{"layout_name":"Layout-1","plan_id":%d}`, plan.PlanID), "")
    if w.Code != http.StatusCreated {
        t.Fatalf("create layout: want 201 got %d: %s", w.Code, w.Body.String())
    }
    var layout models.CuttingLayout
    decodeJSON(t, w, &layout)

    // 5) 创建任务（颜色需在订单项中出现）
    w, _ = doJSONAuth(r, "POST", "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Red","planned_layers":10}`, layout.LayoutID), "")
    if w.Code != http.StatusCreated {
        t.Fatalf("create task red: want 201 got %d: %s", w.Code, w.Body.String())
    }
    var taskRed models.ProductionTask
    decodeJSON(t, w, &taskRed)

    w, _ = doJSONAuth(r, "POST", "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Blue","planned_layers":5}`, layout.LayoutID), "")
    if w.Code != http.StatusCreated {
        t.Fatalf("create task blue: want 201 got %d: %s", w.Code, w.Body.String())
    }
    var taskBlue models.ProductionTask
    decodeJSON(t, w, &taskBlue)

    // 6) 发布计划
    w, _ = doJSONAuth(r, "POST", fmt.Sprintf("/api/v1/plans/%d/publish", plan.PlanID), "", "")
    if w.Code != http.StatusNoContent {
        t.Fatalf("publish plan: want 204 got %d: %s", w.Code, w.Body.String())
    }

    // 发布后创建任务应失败
    w, _ = doJSONAuth(r, "POST", "/api/v1/tasks", fmt.Sprintf(`{"layout_id":%d,"color":"Blue","planned_layers":3}`, layout.LayoutID), "")
    if w.Code == http.StatusCreated {
        t.Fatalf("create task after publish should fail, got 201")
    }

    // 7) 列出布局下的任务，状态应为 in_progress
    w, _ = doJSONAuth(r, "GET", fmt.Sprintf("/api/v1/layouts/%d/tasks", layout.LayoutID), "", "")
    if w.Code != http.StatusOK {
        t.Fatalf("list tasks: want 200 got %d: %s", w.Code, w.Body.String())
    }
    var tasks []models.ProductionTask
    decodeJSON(t, w, &tasks)
    if len(tasks) != 2 {
        t.Fatalf("list tasks: want 2 got %d", len(tasks))
    }
    for _, tt := range tasks {
        if tt.Status != "in_progress" {
            t.Fatalf("task status expected in_progress after publish: %+v", tt)
        }
    }

    // 8) 记录日志以完成任务（触发器会累计并更新任务/计划状态）
    w, _ = doJSONAuth(r, "POST", "/api/v1/logs", fmt.Sprintf(`{"task_id":%d,"layers_completed":10,"worker_name":"张三"}`, taskRed.TaskID), "")
    if w.Code != http.StatusCreated {
        t.Fatalf("log red: want 201 got %d: %s", w.Code, w.Body.String())
    }
    w, _ = doJSONAuth(r, "POST", "/api/v1/logs", fmt.Sprintf(`{"task_id":%d,"layers_completed":5,"worker_name":"王五"}`, taskBlue.TaskID), "")
    if w.Code != http.StatusCreated {
        t.Fatalf("log blue: want 201 got %d: %s", w.Code, w.Body.String())
    }

    // 9) 验证任务完成
    w, _ = doJSONAuth(r, "GET", fmt.Sprintf("/api/v1/tasks/%d", taskRed.TaskID), "", "")
    if w.Code != http.StatusOK {
        t.Fatalf("get task red: want 200 got %d: %s", w.Code, w.Body.String())
    }
    var gotRed models.ProductionTask
    decodeJSON(t, w, &gotRed)
    if gotRed.Status != "completed" || gotRed.CompletedLayers != 10 {
        t.Fatalf("task red not completed: %+v", gotRed)
    }

    w, _ = doJSONAuth(r, "GET", fmt.Sprintf("/api/v1/tasks/%d", taskBlue.TaskID), "", "")
    if w.Code != http.StatusOK {
        t.Fatalf("get task blue: want 200 got %d: %s", w.Code, w.Body.String())
    }
    var gotBlue models.ProductionTask
    decodeJSON(t, w, &gotBlue)
    if gotBlue.Status != "completed" || gotBlue.CompletedLayers != 5 {
        t.Fatalf("task blue not completed: %+v", gotBlue)
    }

    // 10) 计划应自动进度到 completed
    w, _ = doJSONAuth(r, "GET", fmt.Sprintf("/api/v1/plans/%d", plan.PlanID), "", "")
    if w.Code != http.StatusOK {
        t.Fatalf("get plan: want 200 got %d: %s", w.Code, w.Body.String())
    }
    var gotPlan models.ProductionPlan
    decodeJSON(t, w, &gotPlan)
    if gotPlan.Status != "completed" {
        t.Fatalf("plan should be completed after all tasks done: %s", gotPlan.Status)
    }

    // 11) 冻结计划
    w, _ = doJSONAuth(r, "POST", fmt.Sprintf("/api/v1/plans/%d/freeze", plan.PlanID), "", "")
    if w.Code != http.StatusNoContent {
        t.Fatalf("freeze plan: want 204 got %d: %s", w.Code, w.Body.String())
    }

    // 冻结后更新布局名称应失败
    w, _ = doJSONAuth(r, "PATCH", fmt.Sprintf("/api/v1/layouts/%d/name", layout.LayoutID), `{"name":"Renamed"}`, "")
    if w.Code == http.StatusNoContent {
        t.Fatalf("update layout name after freeze should fail")
    }

    // 发布后删除任务应失败
    w, _ = doJSONAuth(r, "DELETE", fmt.Sprintf("/api/v1/tasks/%d", taskRed.TaskID), "", "")
    if w.Code == http.StatusNoContent {
        t.Fatalf("delete task after publish should fail")
    }
}