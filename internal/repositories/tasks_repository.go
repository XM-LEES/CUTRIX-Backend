package repositories

import (
    "context"
    "cutrix-backend/internal/models"
)

// TasksRepository defines data access for production tasks.
// 设计约束：
// - 任务在所属计划发布后，INSERT/DELETE 被触发器拒绝；UPDATE 仅允许 status 与 completed_layers。
// - 为保证审计与回滚一致性，completed_layers 不在任务仓储层直接更新，统一通过 LogsRepository.Create 写日志实现累计。
// - 删除在发布后不可用；发布前受外键约束。
type TasksRepository interface {
    // Basic
    Create(ctx context.Context, task *models.ProductionTask) (int, error)
    Delete(ctx context.Context, id int) error

    // Mutations
    UpdateStatus(ctx context.Context, id int, status string) error
    // 注意：不提供 UpdateCompletedLayers；请使用 LogsRepository.Create 来记录完工并由触发器自动汇总。

    // Queries
    GetByID(ctx context.Context, id int) (*models.ProductionTask, error)
    List(ctx context.Context) ([]models.ProductionTask, error)
    ListByLayout(ctx context.Context, layoutID int) ([]models.ProductionTask, error)
}