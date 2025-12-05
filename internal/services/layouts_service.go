package services

import "cutrix-backend/internal/models"

// LayoutsService 管理生产布局的受控变更与查询：创建/删除、名称与备注更新、按计划查询。
// 约束与约定：
// - 创建/删除/更新：仅允许在所属计划处于 pending 状态时执行；发布后（in_progress/completed/frozen）不可修改结构。
// - 字段更新：发布后仅允许更新 note；名称更新必须在 pending 阶段完成。
// - 查询：提供按 ID 与按计划列出的只读视图；用于上层处理器渲染或校验。
// - 审计一致性：任务状态更新统一走日志，不在布局服务直接影响任务状态或计划完成度。
// - 上下文：接口不透传 context；实现使用 context.Background() 调用仓储，与处理器层解耦。
 type LayoutsService interface {
    // 基本：创建布局（必须关联计划）；成功返回填充的 LayoutID。
    Create(layout *models.CuttingLayout) error
    // 基本：删除布局；受计划状态限制。
    Delete(id int) error

    // 变更：更新布局名称（仅 pending 允许）。
    UpdateName(id int, name string) error
    // 变更：更新布局备注（发布后允许）。
    UpdateNote(id int, note *string) error

    // 查询：按 ID 获取布局详情。
    GetByID(id int) (*models.CuttingLayout, error)
    // 查询：按计划列出布局列表。
    ListByPlan(planID int) ([]models.CuttingLayout, error)

    // 尺码比例：设置布局的尺码比例（仅 pending 允许）。
    SetRatios(id int, ratios map[string]int) error
    // 尺码比例：获取布局的尺码比例。
    GetRatios(id int) ([]models.LayoutSizeRatio, error)
}