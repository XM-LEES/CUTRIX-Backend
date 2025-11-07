package services

import (
    "log/slog"
    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
    "cutrix-backend/internal/logger"
)

type LogsServiceImpl struct {
    repo repositories.LogsRepository
}

func NewLogsService(repo repositories.LogsRepository) *LogsServiceImpl { return &LogsServiceImpl{repo: repo} }

func (s *LogsServiceImpl) Create(log *models.ProductionLog) error {
    if log == nil { return ErrValidation }
    if log.TaskID == 0 { return ErrValidation }
    if log.LayersCompleted <= 0 { return ErrValidation }
    err := s.repo.Create(log)
    if err == nil {
        // 事件日志：生产日志创建成功
        // 字段：log_id（若已填充）、task_id、worker_id、layers_completed
        logger.L.Info("log_created",
            slog.Int("log_id", log.LogID),
            slog.Int("task_id", log.TaskID),
            slog.Any("worker_id", log.WorkerID),
            slog.Int("layers_completed", log.LayersCompleted),
        )
    }
    return err
}

func (s *LogsServiceImpl) ListParticipants(taskID int) ([]string, error) {
    if taskID <= 0 { return nil, ErrValidation }
    return s.repo.ListParticipants(taskID)
}

func (s *LogsServiceImpl) Void(logID int, reason *string, voidedBy *int) error {
    if logID <= 0 { return ErrValidation }
    err := s.repo.Void(logID, reason, voidedBy)
    if err == nil {
        // 事件日志：日志作废成功（不可反作废）
        // 字段：log_id、voided_by、void_reason
        logger.L.Info("log_voided",
            slog.Int("log_id", logID),
            slog.Any("voided_by", voidedBy),
            slog.Any("void_reason", reason),
        )
    }
    return err
}

func (s *LogsServiceImpl) ListByTask(taskID int) ([]models.ProductionLog, error) {
    if taskID <= 0 { return nil, ErrValidation }
    return s.repo.ListByTask(taskID)
}

func (s *LogsServiceImpl) ListByLayout(layoutID int) ([]models.ProductionLog, error) {
    if layoutID <= 0 { return nil, ErrValidation }
    return s.repo.ListByLayout(layoutID)
}

func (s *LogsServiceImpl) ListByPlan(planID int) ([]models.ProductionLog, error) {
    if planID <= 0 { return nil, ErrValidation }
    return s.repo.ListByPlan(planID)
}