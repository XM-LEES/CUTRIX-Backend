package services

import "cutrix-backend/internal/models"

// PlansService 管理生产计划的生命周期与受控变更：创建/删除、发布、冻结、备注更新与查询。
// 约束与约定：
// - 发布（Publish）：仅允许从 pending 发布到 in_progress，需至少存在一个任务；发布时间由触发器自动写入。
// - 完成（completed）：由系统根据任务完成情况自动推进，不提供直接接口；外部人工终态动作为冻结（Freeze）。
// - 冻结（Freeze）：仅允许在 completed 状态下执行；冻结会锁定计划并保留完成时间（由触发器控制）。
// - 字段更新：计划发布后仅允许更新 note；其它字段由触发器限制不可写。
// - 查询：提供按 ID 与按订单列出的只读视图。
// - 事务边界：复杂聚合写入（如计划+布局+任务原子创建）建议由服务层组合实现，仓储层保持单资源写入。
// - 审计一致性：任务进度更新统一通过日志记录，触发器汇总，不在任务仓储层直接改 completed_layers。
// 注意：接口签名不传 context；实现中使用 context.Background() 调用仓储，保持与 handlers 解耦。
 type PlansService interface {
    // 基本：创建计划（必须关联订单）；成功返回填充的 PlanID 与默认状态。
    Create(plan *models.ProductionPlan) error
    // 基本：删除计划（任意状态），级联其布局/任务/比例。
    Delete(id int) error

    // 变更：更新备注（发布后允许）；其它字段由触发器限制。
    UpdateNote(id int, note *string) error
    // 变更：发布计划（pending -> in_progress），触发器设置发布时间并推进子任务状态。
    Publish(id int) error
    // 变更：冻结计划（completed -> frozen），由触发器校验完成态与完成时间。
    Freeze(id int) error

    // 查询：按 ID 获取计划详情。
    GetByID(id int) (*models.ProductionPlan, error)
    // 查询：列出所有计划。
    List() ([]models.ProductionPlan, error)
    // 查询：按订单列出所有计划。
    ListByOrder(orderID int) ([]models.ProductionPlan, error)
}