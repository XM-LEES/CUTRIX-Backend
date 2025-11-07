package repositories

import (
    "context"
    "cutrix-backend/internal/models"
)

// PlansRepository defines data access for production plans.
// 设计约束：
// - 发布/完成仅更新 status，余下自动行为由触发器处理（publish/finish 日期、数量校验等）。
// - 发布后允许更新的字段仅 note；其它字段由触发器限制不可写。
// - 删除允许在任何状态执行，将级联删除其布局/任务/比例；子表的发布后约束由各自触发器控制。
// - 跨表原子创建（计划+布局+任务）如需支持应在服务层组合或另行定义聚合方法。
// 如需扩展查询（分页、筛选），建议统一由服务层定义 filter 结构体，仓储层使用参数化方法避免循环依赖。
type PlansRepository interface {
    // Basic
    Create(ctx context.Context, plan *models.ProductionPlan) (int, error)
    Delete(ctx context.Context, id int) error

    // Mutations
    UpdateNote(ctx context.Context, id int, note *string) error

    // Business actions (rely on DB triggers for validation & auto dates)
    Publish(ctx context.Context, id int) error   // status -> in_progress
    Freeze(ctx context.Context, id int) error    // status -> frozen

    // Queries
    GetByID(ctx context.Context, id int) (*models.ProductionPlan, error)
    ListByOrder(ctx context.Context, orderID int) ([]models.ProductionPlan, error)
}