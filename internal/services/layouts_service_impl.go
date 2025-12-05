package services

import (
    "context"
    "errors"
    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
)

// layoutsService 实现 LayoutsService，负责在服务层封装布局相关的受控写入与只读查询。
// 设计要点：
// - 状态约束：创建/删除/更新仅在计划 pending 时允许；发布后仅 note 可改。
// - 输入校验：对 name/planID 等进行基本校验，复杂约束由仓储与触发器保证。
// - 上下文：统一使用 context.Background() 调用仓储，避免外部 context 泄漏。
// - 错误策略：原样透传仓储返回的业务错误，便于处理器映射 HTTP 状态码。
 type layoutsService struct {
    repo repositories.LayoutsRepository
}

// NewLayoutsService 以给定仓储实现创建 LayoutsService。
// repo：LayoutsRepository 实例，必须非 nil。
// 返回：可用的 LayoutsService；nil 仓储将 panic（与 users/orders 保持一致）。
 func NewLayoutsService(repo repositories.LayoutsRepository) LayoutsService {
    if repo == nil {
        panic("nil LayoutsRepository")
    }
    return &layoutsService{repo: repo}
}

// Create 创建布局，要求提供 planID 与名称；成功后返回填充的 ID。
// layout：待创建的布局实体指针；可能被填充 ID。
// 返回：错误信息；计划状态不允许或其它约束失败由仓储层返回。
 func (s *layoutsService) Create(layout *models.CuttingLayout) error {
    if layout == nil {
        return errors.New("nil layout")
    }
    if layout.PlanID <= 0 {
        return errors.New("plan_id required")
    }
    if layout.LayoutName == "" {
        return errors.New("layout_name required")
    }
    _, err := s.repo.Create(context.Background(), layout)
    return err
}

// Delete 删除指定布局，仅在计划 pending 时允许。
// id：布局 ID。
// 返回：错误信息；不允时仓储返回约束错误。
 func (s *layoutsService) Delete(id int) error {
    if id <= 0 {
        return errors.New("invalid layout_id")
    }
    return s.repo.Delete(context.Background(), id)
}

// UpdateName 更新布局名称，仅在计划 pending 时允许。
// id：布局 ID；name：新名称。
// 返回：错误信息；状态不允由仓储返回。
 func (s *layoutsService) UpdateName(id int, name string) error {
    if id <= 0 {
        return errors.New("invalid layout_id")
    }
    if name == "" {
        return errors.New("layout_name required")
    }
    return s.repo.UpdateName(context.Background(), id, name)
}

// UpdateNote 更新布局备注，发布后也允许。
// id：布局 ID；note：备注，可为 nil 表示清空。
// 返回：错误信息；状态不允由仓储返回。
 func (s *layoutsService) UpdateNote(id int, note *string) error {
    if id <= 0 {
        return errors.New("invalid layout_id")
    }
    return s.repo.UpdateNote(context.Background(), id, note)
}

// GetByID 查询单个布局详情。
// id：布局 ID。
// 返回：布局实体只读副本与错误；不存在时返回仓储层 NotFound 错误。
 func (s *layoutsService) GetByID(id int) (*models.CuttingLayout, error) {
    if id <= 0 {
        return nil, errors.New("invalid layout_id")
    }
    return s.repo.GetByID(context.Background(), id)
}

// ListByPlan 按计划列出布局集合。
// planID：计划 ID。
// 返回：布局列表与错误；若计划不存在或无布局返回空列表或仓储层错误。
func (s *layoutsService) ListByPlan(planID int) ([]models.CuttingLayout, error) {
    if planID <= 0 {
        return nil, errors.New("invalid plan_id")
    }
    return s.repo.ListByPlan(context.Background(), planID)
}

// SetRatios 设置布局的尺码比例，仅在计划 pending 时允许。
// id：布局 ID；ratios：尺码到比例的映射。
// 返回：错误信息；状态不允由仓储层返回。
func (s *layoutsService) SetRatios(id int, ratios map[string]int) error {
    if id <= 0 {
        return errors.New("invalid layout_id")
    }
    if ratios == nil {
        return errors.New("ratios required")
    }
    return s.repo.SetRatios(context.Background(), id, ratios)
}

// GetRatios 获取布局的尺码比例。
// id：布局 ID。
// 返回：尺码比例列表与错误。
func (s *layoutsService) GetRatios(id int) ([]models.LayoutSizeRatio, error) {
    if id <= 0 {
        return nil, errors.New("invalid layout_id")
    }
    return s.repo.GetRatios(context.Background(), id)
}