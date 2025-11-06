package models

// Core domain models for laying-up (拉布) process

type Worker struct {
    WorkerID int    `json:"worker_id"`
    Name     string `json:"name"`
}

type ProductionOrder struct {
    OrderID      int     `json:"order_id"`
    OrderNumber  string  `json:"order_number"`
    StyleNumber  string  `json:"style_number"`
    CustomerName *string `json:"customer_name,omitempty"`
    Notes        *string `json:"notes,omitempty"`
    Status       string  `json:"status"`
}

type OrderItem struct {
    ItemID   int    `json:"item_id"`
    OrderID  int    `json:"order_id"`
    Color    string `json:"color"`
    Size     string `json:"size"`
    Quantity int    `json:"quantity"`
}

type ProductionPlan struct {
    PlanID   int    `json:"plan_id"`
    PlanName string `json:"plan_name"`
    OrderID  int    `json:"order_id"`
    Status   string `json:"status"`
}

type CuttingLayout struct {
    LayoutID   int     `json:"layout_id"`
    PlanID     int     `json:"plan_id"`
    LayoutName string  `json:"layout_name"`
    Description *string `json:"description,omitempty"`
}

type LayoutSizeRatio struct {
    RatioID int    `json:"ratio_id"`
    LayoutID int   `json:"layout_id"`
    Size    string `json:"size"`
    Ratio   int    `json:"ratio"`
}

type ProductionTask struct {
    TaskID          int    `json:"task_id"`
    LayoutID        int    `json:"layout_id"`
    Color           string `json:"color"`
    PlannedLayers   int    `json:"planned_layers"`
    CompletedLayers int    `json:"completed_layers"`
    Status          string `json:"status"`
}

type ProductionLog struct {
    LogID           int     `json:"log_id"`
    TaskID          int     `json:"task_id"`
    WorkerID        *int    `json:"worker_id,omitempty"`
    WorkerName      *string `json:"worker_name,omitempty"`
    LayersCompleted int     `json:"layers_completed"`
    Notes           *string `json:"notes,omitempty"`
    Voided          bool    `json:"voided"`
    VoidReason      *string `json:"void_reason,omitempty"`
    VoidedAt        *string `json:"voided_at,omitempty"`
    VoidedBy        *int    `json:"voided_by,omitempty"`
    VoidedByName    *string `json:"voided_by_name,omitempty"`
}