// models layer: domain models for laying-up (拉布) process
package models

import "time"

type User struct {
    UserID       int     `json:"user_id" db:"user_id"`
    Name         string  `json:"name" db:"name"`
    PasswordHash string  `json:"-" db:"password_hash"`
    Role         string  `json:"role" db:"role"`
    IsActive     bool    `json:"is_active" db:"is_active"`
    Group        *string `json:"user_group,omitempty" db:"user_group"`
    Note         *string `json:"note,omitempty" db:"note"`
}

type ProductionOrder struct {
    OrderID             int     `json:"order_id"`
    OrderNumber         string  `json:"order_number"`
    StyleNumber         string  `json:"style_number"`
    CustomerName        *string `json:"customer_name,omitempty"`
    OrderStartDate      *time.Time `json:"order_start_date,omitempty"`
    OrderFinishDate     *time.Time `json:"order_finish_date,omitempty"`
    Note                *string `json:"note,omitempty"`
    CreatedAt           time.Time `json:"created_at"`
    UpdatedAt           time.Time `json:"updated_at"`
}

type OrderItem struct {
    ItemID   int    `json:"item_id"`
    OrderID  int    `json:"order_id"`
    Color    string `json:"color"`
    Size     string `json:"size"`
    Quantity int    `json:"quantity"`
}

type ProductionPlan struct {
    PlanID              int        `json:"plan_id"`
    PlanName            string     `json:"plan_name"`
    OrderID             int        `json:"order_id"`
    Note                *string    `json:"note,omitempty"`
    PlannedPublishDate  *time.Time `json:"planned_publish_date,omitempty"`
    PlannedFinishDate   *time.Time `json:"planned_finish_date,omitempty"`
    Status              string     `json:"status"`
}

type CuttingLayout struct {
    LayoutID   int     `json:"layout_id"`
    PlanID     int     `json:"plan_id"`
    LayoutName string  `json:"layout_name"`
    Note       *string `json:"note,omitempty"`
}

type LayoutSizeRatio struct {
    RatioID  int    `json:"ratio_id"`
    LayoutID int    `json:"layout_id"`
    Size     string `json:"size"`
    Ratio    int    `json:"ratio"`
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
    LogID           int        `json:"log_id"`
    TaskID          int        `json:"task_id"`
    WorkerID        *int       `json:"worker_id,omitempty"`
    WorkerName      *string    `json:"worker_name,omitempty"`
    LayersCompleted int        `json:"layers_completed"`
    LogTime         time.Time  `json:"log_time"`
    Note            *string    `json:"note,omitempty"`
    Voided          bool       `json:"voided"`
    VoidReason      *string    `json:"void_reason,omitempty"`
    VoidedAt        *time.Time `json:"voided_at,omitempty"`
    VoidedBy        *int       `json:"voided_by,omitempty"`
    VoidedByName    *string    `json:"voided_by_name,omitempty"`
}