package models

// Todo 最小示例实体
type Todo struct {
    ID        int64  `json:"id"`
    Title     string `json:"title"`
    Completed bool   `json:"completed"`
    CreatedAt string `json:"created_at"`
    UserID    *int64 `json:"user_id,omitempty"`
}