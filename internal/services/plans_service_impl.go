package services

import (
    "context"
    "errors"
    "log/slog"
    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
    "cutrix-backend/internal/logger"
)

// plansService 实现 PlansService，封装与 ProductionPlan 相关的业务规则与仓储调用。
// 设计要点：
// - 输入校验：对必填字段（OrderID、PlanName 等）进行基础校验；复杂约束由触发器与仓储层保证。
// - 状态机：仅暴露 Publish/Freeze；完成态由系统自动推进，不直接提供 Complete API。
// - 上下文：不透传外部 context，统一使用 context.Background() 调用仓储，避免处理器泄漏。
// - 只读查询：GetByID/ListByOrder 返回只读视图，不在服务层做拼装计算。
// - 错误策略：仓储返回的业务错误（如状态不允）保持原样透传，便于处理器按约定映射 HTTP 状态码。
 type plansService struct {
    repo repositories.PlansRepository
}

// NewPlansService 以给定的仓储实现创建 PlansService。
// repo：PlansRepository 实例，必须非 nil。
// 返回：可用的 PlansService；若 repo 为 nil 将 panic（与 users/orders 保持一致）。
 func NewPlansService(repo repositories.PlansRepository) PlansService {
    if repo == nil {
        panic("nil PlansRepository")
    }
    return &plansService{repo: repo}
}

// Create 创建新的生产计划。需要至少包含订单信息与名称，状态初始为 pending。
// plan：待创建的计划实体指针；方法可能会填充其 ID 与默认状态。
// 返回：错误信息；输入缺失或仓储约束失败将返回错误。
 func (s *plansService) Create(plan *models.ProductionPlan) error {
    if plan == nil {
        return errors.New("nil plan")
    }
    if plan.OrderID <= 0 {
        return errors.New("order_id required")
    }
    if plan.PlanName == "" {
        return errors.New("plan_name required")
    }
    _, err := s.repo.Create(context.Background(), plan)
    if err == nil {
        // 事件日志：计划创建成功
        // 字段：plan_id（若已填充）、order_id、plan_name
        logger.L.Info("plan_created",
            slog.Int("plan_id", plan.PlanID),
            slog.Int("order_id", plan.OrderID),
            slog.String("plan_name", plan.PlanName),
        )
    }
    return err
}

// Delete 删除指定生产计划。删除为受控操作，可能级联布局与任务。
// id：计划 ID，必须为正数。
// 返回：错误信息；当计划已发布且受限时，由仓储层返回约束错误。
 func (s *plansService) Delete(id int) error {
    if id <= 0 {
        return errors.New("invalid plan_id")
    }
    err := s.repo.Delete(context.Background(), id)
    if err == nil {
        // 事件日志：计划删除成功
        // 字段：plan_id
        logger.L.Info("plan_deleted", slog.Int("plan_id", id))
    }
    return err
}

// UpdateNote 更新计划备注。发布后的计划允许更新 note，其它字段由触发器限制。
// id：计划 ID；note：新的备注内容，可为 nil 表示清空。
// 返回：错误信息；状态不允许或触发器校验失败由仓储层返回。
 func (s *plansService) UpdateNote(id int, note *string) error {
    if id <= 0 {
        return errors.New("invalid plan_id")
    }
    err := s.repo.UpdateNote(context.Background(), id, note)
    if err == nil {
        // 事件日志：计划备注更新
        // 字段：plan_id、note（文本可能较长，谨慎设置级别）
        logger.L.Info("plan_note_updated",
            slog.Int("plan_id", id),
            slog.Any("note", note),
        )
    }
    return err
}

// Publish 发布计划，将状态从 pending 推进至 in_progress。发布时间由触发器自动记录。
// id：计划 ID。
// 返回：错误信息；如果计划没有任务或状态不为 pending，则返回仓储层错误。
 func (s *plansService) Publish(id int) error {
    if id <= 0 {
        return errors.New("invalid plan_id")
    }
    err := s.repo.Publish(context.Background(), id)
    if err == nil {
        // 事件日志：计划发布成功
        // 字段：plan_id
        logger.L.Info("plan_published", slog.Int("plan_id", id))
    }
    return err
}

// Freeze 冻结计划，仅允许在 completed 状态下执行。冻结后计划不可变更。
// id：计划 ID。
// 返回：错误信息；若未完成或触发器校验失败，由仓储层返回错误。
 func (s *plansService) Freeze(id int) error {
    if id <= 0 {
        return errors.New("invalid plan_id")
    }
    err := s.repo.Freeze(context.Background(), id)
    if err == nil {
        // 事件日志：计划冻结成功
        // 字段：plan_id
        logger.L.Info("plan_frozen", slog.Int("plan_id", id))
    }
    return err
}

// GetByID 查询单个计划详情。
// id：计划 ID。
// 返回：计划实体只读副本与错误；不存在时返回仓储层 NotFound 错误。
 func (s *plansService) GetByID(id int) (*models.ProductionPlan, error) {
    if id <= 0 {
        return nil, errors.New("invalid plan_id")
    }
    return s.repo.GetByID(context.Background(), id)
}

// List 列出所有计划。
// 返回：计划列表与错误。
 func (s *plansService) List() ([]models.ProductionPlan, error) {
    return s.repo.List(context.Background())
}

// ListByOrder 按订单列出计划集合。
// orderID：订单 ID。
// 返回：计划列表与错误；若订单不存在或无计划返回空列表或仓储层错误。
 func (s *plansService) ListByOrder(orderID int) ([]models.ProductionPlan, error) {
    if orderID <= 0 {
        return nil, errors.New("invalid order_id")
    }
    return s.repo.ListByOrder(context.Background(), orderID)
}