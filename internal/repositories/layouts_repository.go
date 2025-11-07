package repositories

import (
    "context"
    "cutrix-backend/internal/models"
)

// LayoutsRepository defines data access for cutting layouts.
// 设计约束：
// - 布局在所属计划发布后（status = in_progress/后续），INSERT/UPDATE/DELETE 将被触发器拒绝。
// - 允许在发布前更新 layout_name 与 note；发布后请通过计划层接口控制。
// - 删除受外键约束：会级联删除其任务与比例（若未发布）。
type LayoutsRepository interface {
    // Basic
    Create(ctx context.Context, layout *models.CuttingLayout) (int, error)
    Delete(ctx context.Context, id int) error

    // Mutations (触发器在发布后拒绝)
    UpdateName(ctx context.Context, id int, name string) error
    UpdateNote(ctx context.Context, id int, note *string) error

    // Queries
    GetByID(ctx context.Context, id int) (*models.CuttingLayout, error)
    ListByPlan(ctx context.Context, planID int) ([]models.CuttingLayout, error)
}