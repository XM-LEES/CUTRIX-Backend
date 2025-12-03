package services

import (
    "context"
    "errors"
    "log/slog"
    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
    "cutrix-backend/internal/logger"
)

// tasksService 实现 TasksService，封装任务的受控写入与只读查询。
// 设计要点：
// - 状态约束：创建/删除仅在所属计划 pending 时允许；状态更新不在此服务暴露。
// - 审计与一致性：任务进度通过 LogsService 记录，由触发器汇总到任务/计划，避免绕过审计。
// - 输入校验：对 layoutID/color/planned_layers 等进行基础校验；复杂约束交由仓储与触发器。
// - 上下文：统一使用 context.Background() 调用仓储，避免外部 context 泄漏。
 type tasksService struct {
    repo repositories.TasksRepository
}

// NewTasksService 以给定仓储实现创建 TasksService。
// repo：TasksRepository 实例，必须非 nil。
// 返回：可用的 TasksService；nil 仓储将 panic（与 users/orders 保持一致）。
 func NewTasksService(repo repositories.TasksRepository) TasksService {
    if repo == nil {
        panic("nil TasksRepository")
    }
    return &tasksService{repo: repo}
}

// Create 创建任务，要求提供布局关联与必要信息；成功后返回填充的 ID。
// task：待创建的任务实体指针；可能被填充 ID。
// 返回：错误信息；计划状态不允许或其它约束失败由仓储层返回。
 func (s *tasksService) Create(task *models.ProductionTask) error {
    if task == nil {
        return errors.New("nil task")
    }
    if task.LayoutID <= 0 {
        return errors.New("layout_id required")
    }
    if task.Color == "" {
        return errors.New("color required")
    }
    if task.PlannedLayers <= 0 {
        return errors.New("planned_layers must be > 0")
    }
    _, err := s.repo.Create(context.Background(), task)
    if err == nil {
        // 事件日志：任务创建成功
        // 字段：task_id（若已填充）、layout_id、color、planned_layers
        logger.L.Info("task_created",
            slog.Int("task_id", task.TaskID),
            slog.Int("layout_id", task.LayoutID),
            slog.String("color", task.Color),
            slog.Int("planned_layers", task.PlannedLayers),
        )
    }
    return err
}

// Delete 删除指定任务，仅在所属计划 pending 时允许。
// id：任务 ID。
// 返回：错误信息；不允时仓储返回约束错误。
 func (s *tasksService) Delete(id int) error {
    if id <= 0 {
        return errors.New("invalid task_id")
    }
    err := s.repo.Delete(context.Background(), id)
    if err == nil {
        // 事件日志：任务删除成功
        // 字段：task_id
        logger.L.Info("task_deleted", slog.Int("task_id", id))
    }
    return err
}

// GetByID 查询单个任务详情。
// id：任务 ID。
// 返回：任务实体只读副本与错误；不存在时返回仓储层 NotFound 错误。
 func (s *tasksService) GetByID(id int) (*models.ProductionTask, error) {
    if id <= 0 {
        return nil, errors.New("invalid task_id")
    }
    return s.repo.GetByID(context.Background(), id)
}

// List 列出所有任务。
// 返回：任务列表与错误。
 func (s *tasksService) List() ([]models.ProductionTask, error) {
    return s.repo.List(context.Background())
}

// ListByLayout 按布局列出任务集合。
// layoutID：布局 ID。
// 返回：任务列表与错误；若布局不存在或无任务返回空列表或仓储层错误。
 func (s *tasksService) ListByLayout(layoutID int) ([]models.ProductionTask, error) {
    if layoutID <= 0 {
        return nil, errors.New("invalid layout_id")
    }
    return s.repo.ListByLayout(context.Background(), layoutID)
}