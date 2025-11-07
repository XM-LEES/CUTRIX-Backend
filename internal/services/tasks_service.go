package services

import "cutrix-backend/internal/models"

// TasksService 管理任务的受控变更与查询：创建/删除、按布局查询与按 ID 查询。
// 约束与约定：
// - 创建/删除：仅允许在所属计划为 pending 时执行；发布后任务结构不可新增/删除。
// - 状态更新：不直接暴露 UpdateStatus；任务进度通过 LogsService 记录，触发器汇总到任务/计划完成度，以保证审计与一致性。
// - 查询：提供按 ID 与按布局列出的只读视图。
// - 上下文：接口不透传 context；实现使用 context.Background() 调用仓储，与处理器层解耦。
 type TasksService interface {
    // 基本：创建任务（必须关联布局）；成功返回填充的 TaskID。
    Create(task *models.ProductionTask) error
    // 基本：删除任务；受计划状态限制。
    Delete(id int) error

    // 查询：按 ID 获取任务详情。
    GetByID(id int) (*models.ProductionTask, error)
    // 查询：按布局列出任务列表。
    ListByLayout(layoutID int) ([]models.ProductionTask, error)
}