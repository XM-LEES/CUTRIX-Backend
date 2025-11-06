package services

import (
    "errors"
    "cutrix-backend/internal/models"
    "cutrix-backend/internal/repositories"
)

type LogsServiceImpl struct {
    repo repositories.LogsRepository
}

func NewLogsService(repo repositories.LogsRepository) *LogsServiceImpl { return &LogsServiceImpl{repo: repo} }

func (s *LogsServiceImpl) Create(log *models.ProductionLog) error {
    if log == nil { return errors.New("nil log") }
    if log.TaskID == 0 { return errors.New("task_id required") }
    if log.LayersCompleted <= 0 { return errors.New("layers_completed must be > 0") }
    return s.repo.Create(log)
}

func (s *LogsServiceImpl) ListParticipants(taskID int) ([]string, error) {
    if taskID <= 0 { return nil, errors.New("invalid task_id") }
    return s.repo.ListParticipants(taskID)
}

func (s *LogsServiceImpl) SetVoided(logID int, voided bool, reason *string, voidedBy *int) error {
    if logID <= 0 { return errors.New("invalid log_id") }
    return s.repo.SetVoided(logID, voided, reason, voidedBy)
}